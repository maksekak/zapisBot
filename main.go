package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/xuri/excelize/v2"
)

var (
	findNoteButt = "Увидеть свободные записи"
	firstMenu    = "<b>Здрасте</b>"
	recButt      = "Записаться"
	bot          *tgbotapi.BotAPI

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
	mu        sync.Mutex
	idCheckMu sync.Mutex
)

func main() {
	f, err := excelize.OpenFile("test.xlsx")
	if err != nil {
		fmt.Println("oшибка открытия файла", err)
	}
	defer f.Close()
	go runDailyUpdater(f)
	delPast(f) //не забудь настроить
	addFut(f)
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
		handleMessage(f, update.Message, userStorage, inputData)
	case update.CallbackQuery != nil:
		handleBut(f, update.CallbackQuery)
	}
}
func handleMessage(f *excelize.File, message *tgbotapi.Message, userStorage map[int64]userStatus, inputData map[int64][]string) {
	text := message.Text
	chatID := message.From.ID
	// Логгирование с контекстом
	log.Printf("Пользователь %d отправил: %s", chatID, text)
	// Проверка статуса пользователя
	if text == "bCVUWfOPzuWWVLP$?4jq" {
		copyTable(f)
		doco := tgbotapi.NewDocument(chatID, tgbotapi.FilePath("test.xlsx"))
		doc := tgbotapi.NewDocument(chatID, tgbotapi.FilePath("styleTable.xlsx"))
		bot.Send(doco)
		bot.Send(doc)
	}
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
		currentData = inputData[chatID]
		mu.Unlock()
		// Проверяем количество полей
		switch len(currentData) {
		case 1:
			if !IsDateFormat(currentData[0]) {
				if currentData[0] != "" {
					sendReply(chatID, "Дата введенна неверно")
					validErr(chatID, currentData)
					break
				}
			}
			sendReply(message.Chat.ID, "Пожалуйста, введите время например:<b>11</b>)")
			return
		case 2:
			if !HasValidTime(currentData[1]) {
				sendReply(chatID, "Время введенно неверно")
				validErr(chatID, currentData)
				break
			}
			sendReply(message.Chat.ID, "Пожалуйста, введите имя")
			return
		case 3:
			sendReply(message.Chat.ID, "Пожалуйста, ведите фамилию")
			return
		case 4:
			if !IsValidName(currentData[2], currentData[3]) {
				sendReply(chatID, "Имя или фамилия введенна неверно")
				validErr(chatID, currentData)
				break
			}
			sendReply(message.Chat.ID, "Пожалуйста, укажите ваш номер телефона (начинайте с +7 или 8)")
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
			if !reFind(f, userStorage, chatID) {
				sendReply(chatID, "Запись занята")
				readyToRec[chatID] = false
			}
			// Очищаем временные данные
			mu.Lock()
			delete(inputData, chatID)
			currentData = nil
			mu.Unlock()
			if readyToRec[chatID] {
				sendRecButt(chatID)
			}
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
		readyToRec[chatId] = false
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
	case findNoteButt:
		setUserReadyToRec(chatId)
		freeDaysData := freeDays(f, 1)
		if len(freeDaysData) == 0 {
			text = "Свободных слотов нет"
		} else {
			msg := tgbotapi.NewMessage(chatId, sendFreeDays(freeDaysData))
			msg.ParseMode = tgbotapi.ModeHTML
			bot.Send(msg)
		}
		fmt.Print(readyToRec)
		sendReply(message.Chat.ID, "Пожалуйста, введите дату например:<b>01.01.25</b>")
	case recButt:
		log.Printf("Обработка записи для chatId=%d", chatId)
		fmt.Println(userStorage)
		if reFind(f, userStorage, chatId) {
			newName(f, userStorage, chatId)
		} else {
			sendReply(chatId, "Запись занята")
		}
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
func sendReply(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	bot.Send(msg)
}
func sendErrorReply(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, "⚠️ "+text)
	bot.Send(msg)
}
func sendRecButt(chatId int64) {
	msg := tgbotapi.NewMessage(chatId, "Данные успешно сохранены! Нажмите кнопку записаться")
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = recMarkup
	bot.Send(msg)
}
