package main

import (
	"fmt"
	"log"
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
	userId      int64
	userDate    string
	userTime    string
	userPhone   string
	userName    string
	userSurname string
	userOrder   string
}

func getUserStatus(id int64) bool {
	mu.Lock()
	defer mu.Unlock()

	return readyToRec[id]
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
	time, _ := ParseHourFromTime(slice[1])
	t := fmt.Sprintf("%d", time)
	user := userStatus{
		userDate:    slice[0],
		userTime:    t,
		userName:    slice[2],
		userSurname: slice[3],
		userPhone:   slice[4],
		userOrder:   slice[5],
		userId:      id,
	}

	// Запись в мапу
	userStorage[id] = user

	// Логгирование
	log.Printf("Данные сохранены для id=%d: %+v", id, user)

	return nil // Успех
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
	lines = append(lines, "🕘<b> Выберите дату и время свободной записи:</b>\n")

	// 1. Извлекаем и сортируем даты (формат ДД.ММ.ГГ)
	sortedDates := make([]string, 0, len(dates))
	for date := range dates {
		sortedDates = append(sortedDates, date)
	}

	// Сортируем как строки (для формата ДД.ММ.ГГ это работает корректно)
	sort.Strings(sortedDates)

	// 2. Формируем строки по шаблону
	for _, dateStr := range sortedDates {
		times := dates[dateStr]

		// Преобразуем строку даты в time.Time для получения дня недели
		date, err := time.Parse("02.01.06", dateStr) // ДД.ММ.ГГ
		if err != nil {
			continue // пропускаем некорректные даты
		}

		// Форматируем дату как "20.10.2025, пн"
		formattedDate := date.Format("<b>02.1.06</b>") // ДД.ММ.ГГГГ
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
			formattedTime := fmt.Sprintf("%2d:00", hour)
			formattedTimes = append(formattedTimes, formattedTime)
		}

		// Объединяем время через запятую
		if len(formattedTimes) > 0 {
			timesStr := strings.Join(formattedTimes, ", ")
			lines = append(lines, timesStr)
		} else {
			lines = append(lines, "Нет свободных окошек")
		}

		// Добавляем пустую строку между записями (опционально)
		lines = append(lines, "----------------------------------------------------------------------")
	}

	// Удаляем последнюю пустую строку
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return strings.Join(lines, "\n")
}
