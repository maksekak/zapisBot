package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	readyToRec  = make(map[int64]bool)
	inputData   = make(map[int64][]string)
	idCheck     = make(map[int64][]string)
	userStorage = make(map[int64]userStatus)
	currentData []string
)

type userStatus struct {
	UserId      int64  `json:"UserId"`
	UserDate    string `json:"UserDate"`
	UserTime    string `json:"UserTime"`
	UserPhone   string `json:"UserPhone"`
	UserName    string `json:"UserName"`
	UserSurname string `json:"UserSurname"`
	UserOrder   string `json:"UserOrder"`
	UserHasRec  bool   `json:"UserHasRec"`
}

func getUserStatus(id int64) bool {
	mu.Lock()
	defer mu.Unlock()

	return readyToRec[id]
}
func getUserHasRec(id int64, userStorage map[int64]userStatus) bool {
	mu.Lock()
	defer mu.Unlock()
	// Безопасное получение с проверкой существования
	user := userStorage[id]
	return user.UserHasRec

}
func dataToStruct(slice []string, id int64, userStorage map[int64]userStatus) error {
	// Валидация входных данных
	if slice == nil {
		return fmt.Errorf("срез данных равен nil")
	}
	if len(slice) < 6 {
		return fmt.Errorf("недостаточно данных в срезе: ожидается 6, получено %d", len(slice))
	}
	if id <= 0 {
		return fmt.Errorf("недопустимый id: %d", id)
	}
	// Блокировка мьютекса
	mu.Lock()
	defer mu.Unlock()
	// Создание структуры
	timee, _ := ParseHourFromTime(slice[1])
	t := fmt.Sprintf("%d", timee)
	user := userStatus{
		UserDate:    slice[0],
		UserTime:    t,
		UserName:    slice[2],
		UserSurname: slice[3],
		UserPhone:   slice[4],
		UserOrder:   slice[5],
		UserId:      id,
		UserHasRec:  true,
	}

	// Запись в мапу
	userStorage[id] = user

	// Логгирование
	log.Printf("Данные сохранены для id=%d: %+v", id, user)

	return nil // Успех
}
func SaveUserToFile(userStorage map[int64]userStatus, id int64) error {

	// Проходим по всем пользователям в карте
	for userID, data := range userStorage {
		// Формируем путь к файлу: <директория>/<userID>.json
		filename := fmt.Sprintf("%d.json", userID)

		// Преобразуем структуру в JSON
		jsonData, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("ошибка при кодировании данных пользователя %d в JSON: %w", userID, err)
		}

		// Записываем данные в файл (создаёт или перезаписывает)
		err = os.WriteFile(filename, jsonData, 0755)
		if err != nil {
			return fmt.Errorf("ошибка при записи в файл %s: %w", filename, err)
		}

		fmt.Printf("Данные пользователя %d успешно записаны в файл %s\n", userID, filename)
	}

	return nil
}

func LoadUserByID(id int64) userStatus {
	filename := fmt.Sprintf("%d.json", id)
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return userStatus{} // Пользователь не найден
		}
		// Логируем ошибку (опционально)
		log.Printf("Ошибка при открытии файла %s: %v", filename, err)
		return userStatus{}
	}
	defer file.Close()

	var status userStatus
	err = json.NewDecoder(file).Decode(&status)
	if err != nil {
		// Логируем ошибку (опционально)
		log.Printf("Ошибка при декодировании JSON из файла %s: %v", filename, err)
		return userStatus{}
	}
	return status
}

func setUserReadyToRec(chatId int64) error {
	// Валидация входных данных
	if chatId <= 0 {
		return fmt.Errorf("недопустимый chatId: %d", chatId)
	}

	// Блокировка мьютекса
	mu.Lock()
	defer mu.Unlock() // Гарантирует разблокировку даже при панике

	// Логирование перед изменением

	readyToRec[chatId] = true
	// Логирование результата
	log.Printf("Статус обновлен для chatId=%d", chatId)

	return nil
}

func sendFreeDays(dates map[string][]string) string {
	if len(dates) == 0 {
		return "🚫 Все даты заняты"
	}

	var lines []string
	lines = append(lines, "🕘<b>Выберите дату и время свободной записи:</b>\n")

	// 1. Извлекаем и сортируем даты (формат ДД.ММ.ГГ)
	sortedDates := make([]string, 0, len(dates))
	for date := range dates {
		sortedDates = append(sortedDates, date)
	}
	// Сортировка с учётом дня, месяца и года
	sort.Slice(sortedDates, func(i, j int) bool {
		// Преобразуем строки в формат time.Time
		t1, err1 := time.Parse("02.01.06", sortedDates[i])
		t2, err2 := time.Parse("02.01.06", sortedDates[j])

		// Если одна из дат некорректна — ставим корректную раньше
		if err1 != nil || err2 != nil {
			return err1 == nil // если t1 корректна, она должна быть раньше
		}

		// Сравниваем даты: true → dates[i] должна быть раньше dates[j]
		return t1.Before(t2)
	})

	// 2. Формируем строки по шаблону
	for _, dateStr := range sortedDates {
		times := dates[dateStr]
		// Преобразуем строку даты в time.Time для получения дня недели
		date, err := time.Parse("02.01.06", dateStr) // ДД.ММ.ГГ
		if err != nil {
			continue // пропускаем некорректные даты
		}
		// Форматируем дату как "20.10.2025, пн"
		formattedDate := date.Format("<b><u>02.01.06</u></b>") // ДД.ММ.ГГГГ
		weekday := date.Weekday()
		dayAbbr := map[time.Weekday]string{
			time.Monday:    "пн",
			time.Tuesday:   "вт",
			time.Wednesday: "ср",
			time.Thursday:  "чт",
			time.Friday:    "пт",
			time.Saturday:  "сб",
			time.Sunday:    "вс",
		}[weekday]
		lines = append(lines, formattedDate+", "+dayAbbr)
		// Обрабатываем время: преобразуем "11" → "11:00", "9" → "09:00"
		var formattedTimes []string
		for _, t := range times {
			// Добавляем ":00" и форматируем с ведущим нулём для часов < 10
			hour, err := strconv.Atoi(t)
			if err != nil {
				continue // пропускаем некорректное время
			}
			formattedTime := fmt.Sprintf("%d:00", hour)
			formattedTimes = append(formattedTimes, formattedTime)
		}

		// Объединяем время через запятую
		if len(formattedTimes) > 0 {
			timesStr := strings.Join(formattedTimes, ", ")
			lines = append(lines, timesStr)
		} else {
			lines = append(lines, "Нет свободных записей")
		}

		// Добавляем пустую строку между записями (опционально)
		//textSize := len(lines)
		lines = append(lines, strings.Repeat("·", 70))
		//textSize = 0
	}
	// Удаляем последнюю пустую строку
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return strings.Join(lines, "\n")
}
