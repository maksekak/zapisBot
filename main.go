package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/xuri/excelize/v2"
)

var (
	findNoteButt = "Увидеть свободные записи"
	backButton   = "Назад↩"
	firstMenu    = "<b>Здрасте</b>"
	recButt      = "Записаться"

	bot             *tgbotapi.BotAPI
	idCheck         = make(map[int64][]string)
	firstMenuMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(findNoteButt, findNoteButt),
		),
	)
	recMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(recButt, recButt),
		),
	)
	BackMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(backButton, backButton),
		),
	)
	readyToRec = make(map[int64]bool)

	mu sync.Mutex

	idCheckMu sync.Mutex

	inputData = make(map[int64][]string)
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

var userStorage = make(map[int64]userStatus)

func main() {

	f, err := excelize.OpenFile("test.xlsx")
	if err != nil {
		fmt.Println("oшибка открытия файла", err)
	}
	defer f.Close()

	//инициализирую бота, а токен находится в файле env
	err = godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	bot, err = tgbotapi.NewBotAPI(os.Getenv("token"))
	// если ошибка инициализации паникуем

	if err != nil {
		log.Panic(err)
	}

	// отклбчение подробного дебага по боту
	bot.Debug = false

	// видимо настраиваю время полученя апдейтов
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	// создаю интерфейсы контекста
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	// наверное получаю апдейты с бота(скорее всего)
	updates := bot.GetUpdatesChan(u)

	go receiveUpdetes(f, ctx, updates)
	log.Println("Начал прослушивать обновления. Нажмите enter для остановки")

	bufio.NewReader(os.Stdin).ReadBytes('\n')
	cancel()

}

// idCheck map[id][]string{}
func receiveUpdetes(f *excelize.File, ctx context.Context, updates tgbotapi.UpdatesChannel) {
	//бесконечный for
	for {
		select {
		case <-ctx.Done():
			return
		case update := <-updates:
			handleUpdate(f, update)

		}
	}
}

func handleUpdate(f *excelize.File, update tgbotapi.Update) {

	switch {
	//обработка сообщения
	case update.Message != nil:
		handleMessage(update.Message, userStorage, inputData)
	case update.CallbackQuery != nil:
		handleBut(f, update.CallbackQuery)
	}
}

func handleMessage(message *tgbotapi.Message, userStorage map[int64]userStatus, inputData map[int64][]string) {
	text := message.Text
	chatID := message.From.ID

	// Логгирование с контекстом
	log.Printf("Пользователь %d отправил: %s", chatID, text)

	// Проверка статуса пользователя
	isReady := getUserStatus(chatID)
	fmt.Println(isReady)
	if !isReady {
		// Обрабатываем как команду, если не в режиме ввода

		err := handleCommand(message.Chat.ID, text)
		if err != nil {
			log.Printf("Ошибка выполнения команды для %d: %v", chatID, err)
			sendErrorReply(message.Chat.ID, "Ошибка при выполнении команды")
		}

		return
	}
	if isReady {

		// Безопасное добавление данных
		mu.Lock()
		inputData[chatID] = append(inputData[chatID], text)
		currentData := inputData[chatID]
		mu.Unlock()

		// Проверяем количество полей
		switch len(currentData) {

		case 1:
			if !IsDateFormat(currentData[0]) {
				sendReply(chatID, "Дата введенна неверно")
				validErr(chatID, currentData)
				break
			}

			sendReply(message.Chat.ID, "Введите время например(11)")
			return
		case 2:
			if !HasValidNumber(currentData[1]) {
				sendReply(chatID, "Время введенно неверно")
				validErr(chatID, currentData)
				break
			}
			sendReply(message.Chat.ID, "Введите имя")
			return
		case 3:
			sendReply(message.Chat.ID, "Введите фамилию")
			return
		case 4:
			if !IsValidName(currentData[2], currentData[3]) {
				sendReply(chatID, "Имя или фамилия введенна неверно")
				validErr(chatID, currentData)
				break
			}
			sendReply(message.Chat.ID, "Введите номер телефона")
			return
		case 5:
			if !IsValidPhoneRegex(currentData[4]) {
				sendReply(chatID, "Номер телефона введенн неверно")
				validErr(chatID, currentData)
				break
			}
			sendReply(message.Chat.ID, "Введите описания заказа")
			return
		}

		// Сохраняем данные
		if len(currentData) == 6 {
			err := dataToStruct(currentData, chatID, userStorage)
			// Очищаем временные данные
			mu.Lock()
			delete(inputData, chatID)
			currentData = nil
			mu.Unlock()
			sendRecButt(chatID)
			if err != nil {
				log.Printf("Ошибка сохранения данных для %d: %v", chatID, err)
				sendErrorReply(message.Chat.ID, "Не удалось сохранить данные. Попробуйте снова.")
				return
			}
		}

	}
}

func handleCommand(chatId int64, command string) error {
	var err error
	switch command {
	case "/start":
		//err = sendMenu(chatId)
		//readyToRec[chatId] = false

	}
	sendMenu(chatId)
	readyToRec[chatId] = false
	return err
}
func handleBut(f *excelize.File, query *tgbotapi.CallbackQuery) {
	if query.Message == nil {
		log.Println("Callback без сообщения")
		return
	}

	message := query.Message
	chatId := query.From.ID

	if query.Data == "" {
		log.Println("Пустой callback data")
		return
	}

	markup := tgbotapi.NewInlineKeyboardMarkup()
	text := firstMenu

	switch query.Data {
	case backButton:
		text = firstMenu
		markup = firstMenuMarkup
		log.Printf("%v ->", message.From)
		log.Printf("%v нажал %s", message.From, query.Data)
		readyToRec[chatId] = false

	case findNoteButt:
		sendReply(message.Chat.ID, "Введите дату например(01.01.2025)")
		setUserReadyToRec(chatId)
		freeDaysData := freeDays(f, 1)
		if len(freeDaysData) == 0 {
			text = "Свободных слотов нет"
		} else {
			var lines []string
			for k, i := range freeDaysData {
				lines = append(lines, fmt.Sprintf("Дата: %s, Время: %s", k, i))
			}
			text = strings.Join(lines, "\n")
		}
		fmt.Print(readyToRec)
		markup = BackMarkup

	case recButt:
		log.Printf("Обработка записи для chatId=%d", chatId)

		fmt.Println(userStorage)
		newName(f, userStorage, chatId)

		idCheckMu.Lock()
		idCheck[chatId] = nil
		readyToRec[chatId] = false
		idCheckMu.Unlock()

	default:
		log.Printf("Неизвестный callback: %s", query.Data)
		return
	}

	callbackcfg := tgbotapi.NewCallback(query.ID, "")
	if _, err := bot.Send(callbackcfg); err != nil {
		log.Printf("Ошибка отправки callback: %v", err)
	}

	msg := tgbotapi.NewEditMessageTextAndMarkup(
		message.Chat.ID,
		message.MessageID,
		text,
		markup,
	)
	msg.ParseMode = tgbotapi.ModeHTML
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка редактирования сообщения: %v", err)
	}
}
func sendMenu(chatId int64) error {
	msg := tgbotapi.NewMessage(chatId, firstMenu)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = firstMenuMarkup
	_, err := bot.Send(msg)
	return err
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
	user := userStatus{
		userDate:    slice[0],
		userTime:    slice[1],
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
func sendReply(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	bot.Send(msg)
}

func sendErrorReply(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, "⚠️ "+text)
	bot.Send(msg)
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
func sendRecButt(chatId int64) {
	msg := tgbotapi.NewMessage(chatId, "Данные успешно сохранены! Нажмите кнопку записаться")
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = recMarkup
	bot.Send(msg)
}

// IsDateFormat проверяет, что строка соответствует формату ДД.ММ.ГГГГ
func IsDateFormat(s string) bool {
	// Регулярное выражение: 2 цифры, точка, 2 цифры, точка, 4 цифры
	pattern := `^\d{2}\.\d{2}\.\d{4}$`
	matched, _ := regexp.MatchString(pattern, s)
	return matched
}
func HasValidNumber(s string) bool {
	// Удаляем пробелы и пытаемся преобразовать в int
	s = strings.TrimSpace(s)
	num, err := strconv.Atoi(s)
	if err != nil {
		return false // Не число
	}

	// Проверяем диапазон и исключения
	return num >= 9 && num <= 18 && num != 12 && num != 13
}
func IsValidName(firstName, lastName string) bool {
	// Проверка на пустоту
	if firstName == "" || lastName == "" {
		return false
	}

	// Минимальная длина
	if len(firstName) < 2 || len(lastName) < 2 {
		return false
	}

	// Проверка символов (только буквы и пробелы)
	for _, r := range firstName {
		if !unicode.IsLetter(r) && r != ' ' {
			return false
		}
	}
	for _, r := range lastName {
		if !unicode.IsLetter(r) && r != ' ' {
			return false
		}
	}

	// Проверка пробелов
	trimmedFirst := strings.TrimSpace(firstName)
	trimmedLast := strings.TrimSpace(lastName)
	if trimmedFirst != firstName || trimmedLast != lastName {
		return false // пробелы в начале/конце
	}
	if strings.Contains(firstName, "  ") || strings.Contains(lastName, "  ") {
		return false // несколько пробелов подряд
	}

	return true
}

var phoneRegex = regexp.MustCompile(
	`^(\+7|8)(\s*\(\d{3}\)\s*|\s*\d{3}\s*)?[\d\s\-]{7,15}$`,
)

func IsValidPhoneRegex(phone string) bool {
	return phoneRegex.MatchString(phone)
}
func validErr(chatID int64, currentData []string) {
	mu.Lock()
	delete(inputData, chatID)
	currentData = nil
	mu.Unlock()
	sendMenu(chatID)
}
