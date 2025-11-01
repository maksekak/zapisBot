package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/xuri/excelize/v2"
)

var (
	findNoteButt    = "Все свободные даты"
	firstMenu       = "<b>Здравстуйте я телеграмм бот PulverFarbe вы можете записаться на пескоструйную обработку</b>"
	recButt         = "Записаться"
	cancelButt      = "Отменить запись"
	userRecButt     = "Увидеть свою запись"
	nearDateButt    = "Ближайщая свободная дата"
	bot             *tgbotapi.BotAPI
	daychange       = -22
	firstMenuMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(nearDateButt, nearDateButt),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(userRecButt, userRecButt),
		),
	)
	allDateMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(findNoteButt, findNoteButt),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(userRecButt, userRecButt),
		),
	)
	cancelMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(cancelButt, cancelButt),
		),
	)
	recMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(recButt, recButt),
		),
	)
	userRec   = make(map[int64][]string)
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
	//delPast(f) //не забудь настроить
	//addFut(f)
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
		doc := tgbotapi.NewDocument(chatID, tgbotapi.FilePath("styleTable.xlsx"))
		bot.Send(doc)
	}
	isReady := getUserStatus(chatID)
	fmt.Println(readyToRec, "readytorec")
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
		if !getUserHasRec(chatID, userStorage) {
			// Проверяем количество полей
			switch len(currentData) {
			case 1:
				freeDaysData := freeDays(f, daychange)
				//data,present:=freeDaysData[currentData[1]]
				if !IsDateFormat(currentData[0]) {
					if currentData[0] != "" {
						sendReply(chatID, "Дата введена неверно")
						validErr(chatID, currentData)
						break
					}
				}
				if _, ok := freeDaysData[currentData[0]]; ok {
					sendReply(message.Chat.ID, "Пожалуйста, введите время например: <b>11</b>")
					return
				} else {
					sendReply(chatID, "Дата введена неверно")
					validErr(chatID, currentData)
					break
				}

			case 2:

				if !HasValidTime(currentData[1]) {
					sendReply(chatID, "Время введено неверно")
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
					sendReply(chatID, "Имя или фамилия введена неверно")
					validErr(chatID, currentData)
					break
				}
				sendReply(message.Chat.ID, "Пожалуйста, укажите ваш номер телефона (начинайте с +7 или 8)")
				return
			case 5:
				if !IsValidPhoneRegex(currentData[4]) {
					sendReply(chatID, "Номер телефона введен неверно")
					validErr(chatID, currentData)
					break
				}
				sendReply(message.Chat.ID, "Введите описания заказа")
				return
			}
		} else {
			sendReply(chatID, "У вас уже есть запись")
			sendCancelButt(chatID)
			sendMenu(chatID)
		}
		// Сохраняем данные
		if len(currentData) == 6 {
			err := dataToStruct(currentData, chatID, userStorage)

			if !reFind(f, userStorage, chatID) {
				sendReply(chatID, "Запись занята")
				mu.Lock()
				user := userStatus{
					userHasRec: false,
				}
				userStorage[chatID] = user
				readyToRec[chatID] = false
				mu.Unlock()
			}
			// Очищаем временные данные
			mu.Lock()
			delete(inputData, chatID)
			currentData = nil
			mu.Unlock()
			if readyToRec[chatID] {
				sendRecButt(chatID)
				fmt.Println("dfgsdfsdfgsdgdfsgsgsdfg")
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
		freeDaysData := freeDays(f, daychange)
		if len(freeDaysData) == 0 {
			text = "Свободных слотов нет"
		} else {
			msg := tgbotapi.NewMessage(chatId, sendFreeDays(freeDaysData))
			msg.ParseMode = tgbotapi.ModeHTML
			bot.Send(msg)
		}
		fmt.Print(readyToRec)
		if !getUserHasRec(chatId, userStorage) {
			sendReply(message.Chat.ID, "Если хотите записаться, введите дату например: <b>01.01.25</b>")
		}

	case recButt:
		log.Printf("Обработка записи для chatId=%d", chatId)
		fmt.Println(userStorage)
		if reFind(f, userStorage, chatId) {
			newName(f, userStorage, chatId)
			user := userStorage[chatId]
			msg := tgbotapi.NewEditMessageText(chatId, message.MessageID, fmt.Sprintf("<b>Запись успешно произведена на: %s - %s</b>", user.userDate, user.userTime))

			msg.ReplyMarkup = nil
			msg.ParseMode = tgbotapi.ModeHTML
			bot.Send(msg)
			sendMenu(chatId)
		} else {
			sendReply(chatId, "Запись занята")
		}
		idCheckMu.Lock()
		idCheck[chatId] = nil
		readyToRec[chatId] = false
		idCheckMu.Unlock()
	case nearDateButt:
		setUserReadyToRec(chatId)
		i := daychange
		freeDaysData := freeDays(f, i)

		for key, val := range freeDaysData {
			if key == tomorrowDate(i) {
				if val != nil {
					var str = fmt.Sprintf("Ближайшие часы записи доступные на: <b><u>%s</u></b>\n", key)
					sendReply(chatId, str)
					val = append(val, "")
					sendReply(chatId, strings.Join(val, ":00, "))
					sendReply(chatId, fmt.Sprintf("Если хотите записаться, введите дату: <b>%s</b>", key))
				} else {
					i++
				}
			}
		}
		msg := tgbotapi.NewEditMessageReplyMarkup(chatId, message.MessageID, allDateMarkup)

		bot.Send(msg)
	case userRecButt:

		if len(userRec[chatId]) == 0 {
			sendReply(chatId, "Вы пока что не записались")
		} else {
			usr := fmt.Sprintf("<b>📝 Данные вашей записи:</b>\n<b>Дата:</b> %s - %s\n<b>Имя:</b> %s\n<b>Фамилия:</b> %s\n<b>Телефон:</b> %s\n<b>Заказ:</b> %s", userRec[chatId][0], userRec[chatId][1], userRec[chatId][2], userRec[chatId][3], userRec[chatId][4], userRec[chatId][5])
			sendReply(chatId, usr)
			sendCancelButt(chatId)
		}
	case cancelButt:
		cancelRec(f, chatId, userStorage)
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
	msg := tgbotapi.NewMessage(chatId, "Нажимая кнопку Записаться вы даёте согласие на обработку ваших личных данных (Имя, Фамилия, Номер телефона)")
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = recMarkup
	bot.Send(msg)
}
func sendCancelButt(chatId int64) {
	msg := tgbotapi.NewMessage(chatId, "Если хотите отменить запись, нажмите кнопку")
	msg.ReplyMarkup = cancelMarkup
	bot.Send(msg)
}
