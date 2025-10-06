package main

import (
	"bufio"
	"context"
	"fmt"

	"log"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/xuri/excelize/v2"
)

var (
	jokeButton = "Похихикать"
	backButton = "Назад↩"
	firstMenu  = "<b>Напишите ячейку</b>\n"
	secondMenu = "<b>Напишите данные</b>\n"
	ex, _      = excelize.OpenFile("example.xlsx")
	bot        *tgbotapi.BotAPI
	/*firstMenuMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(jokeButton, jokeButton),
		),
	)

	secondMenuMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(backButton, backButton),
		),
	)*/
)

func main() {

	//fmt.Print(rows[0])
	if err := ex.Save(); err != nil {
		fmt.Println("Ошибка сохранения файла:", err)
	} else {
		fmt.Println("Данные успешно записаны и файл сохранен")
	}

	//инициализирую бота, а токен находится в файле env
	err := godotenv.Load()
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

	go receiveUpdetes(ctx, updates)
	log.Println("Начал прослушивать обновления. Нажмите enter для остановки")

	bufio.NewReader(os.Stdin).ReadBytes('\n')
	cancel()

}
func receiveUpdetes(ctx context.Context, updates tgbotapi.UpdatesChannel) {
	//бесконечный for
	for {
		select {
		case <-ctx.Done():
			return
		case update := <-updates:
			handleUpdate(update)

		}
	}

}
func handleUpdate(update tgbotapi.Update) {

	switch {
	//обработка сообщения
	case update.Message != nil:

		handleMessage(update.Message)

	case update.CallbackQuery != nil:
		handleBut(update.CallbackQuery)

	}

}
func handleMessage(message *tgbotapi.Message) {
	text := message.Text
	user := message.From

	if user == nil {
		return
	}
	log.Println(user, "написал", text)
	handleEx("A2", text)
	var err error
	if strings.HasPrefix(text, "/") {
		err = handleCommand(message.Chat.ID, text)

	}

	if err != nil {
		log.Printf("An error occured: %s", err.Error())
	}

}
func handleCommand(chatId int64, command string) error {
	var err error
	switch command {
	case "/start":
		err = sendMenu(chatId)

	}
	return err
}
func handleBut(query *tgbotapi.CallbackQuery) {
	message := query.Message
	markup := tgbotapi.NewInlineKeyboardMarkup()
	var text string = firstMenu
	switch query.Data {
	case backButton:
		text = firstMenu
		//markup = firstMenuMarkup
		log.Println(message.From, "->")
		log.Println(message.From, "нажал", query.Data)
	case jokeButton:
		text = secondMenu
		//markup = secondMenuMarkup
	}

	callbackcfg := tgbotapi.NewCallback(query.ID, "")
	bot.Send(callbackcfg)
	msg := tgbotapi.NewEditMessageTextAndMarkup(message.Chat.ID, message.MessageID, text, markup)
	msg.ParseMode = tgbotapi.ModeHTML
	bot.Send(msg)

}
func sendMenu(chatId int64) error {
	msg := tgbotapi.NewMessage(chatId, firstMenu)
	msg.ParseMode = tgbotapi.ModeHTML
	//msg.ReplyMarkup = firstMenuMarkup
	_, err := bot.Send(msg)
	return err
}
func handleEx(k string, v string) {
	ex.SetCellValue("Sheet1", k, v)
	if err := ex.Save(); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Файл успешно сохранен")
	}

}
