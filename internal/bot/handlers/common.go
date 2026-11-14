package handlers

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	btnNewOrder = "📝 Записаться"
	btnMyOrders = "📋 Мои записи"
	btnCalcInfo = "💰 Услуги и цены"
	btnCancel   = "❌ Отмена"
)

var nonDigitRe = regexp.MustCompile(`\D+`)

// mainMenuKeyboard возвращает основную Reply-клавиатуру клиента.
func mainMenuKeyboard() tgbotapi.ReplyKeyboardMarkup {
	kb := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(btnNewOrder),
			tgbotapi.NewKeyboardButton(btnMyOrders),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(btnCalcInfo),
		),
	)
	kb.ResizeKeyboard = true
	return kb
}

// contactKeyboard возвращает Reply-клавиатуру с кнопкой отправки контакта.
func contactKeyboard() tgbotapi.ReplyKeyboardMarkup {
	btn := tgbotapi.NewKeyboardButton("📱 Отправить номер телефона")
	btn.RequestContact = true
	kb := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(btn),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(btnCancel)),
	)
	kb.ResizeKeyboard = true
	kb.OneTimeKeyboard = true
	return kb
}

// sendHTML отправляет HTML-сообщение с опциональной клавиатурой.
func sendHTML(botAPI *tgbotapi.BotAPI, chatID int64, text string, markup interface{}) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	if markup != nil {
		msg.ReplyMarkup = markup
	}
	if _, err := botAPI.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения в чат %d: %v", chatID, err)
	}
}

// editHTML редактирует существующее сообщение с inline-клавиатурой.
func editHTML(botAPI *tgbotapi.BotAPI, chatID int64, messageID int, text string, markup *tgbotapi.InlineKeyboardMarkup) {
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ParseMode = tgbotapi.ModeHTML
	if markup != nil {
		edit.ReplyMarkup = markup
	}
	if _, err := botAPI.Send(edit); err != nil {
		if !strings.Contains(err.Error(), "message is not modified") {
			log.Printf("Ошибка редактирования сообщения %d в чате %d: %v", messageID, chatID, err)
		}
	}
}

// answerCallback отправляет подтверждение нажатия inline-кнопки.
func answerCallback(botAPI *tgbotapi.BotAPI, callbackID, text string, showAlert bool) {
	cb := tgbotapi.NewCallback(callbackID, text)
	cb.ShowAlert = showAlert
	if _, err := botAPI.Request(cb); err != nil {
		log.Printf("Ошибка ответа на callback %s: %v", callbackID, err)
	}
}

// normalizePhone приводит телефон к единому формату (+7XXXXXXXXXX или +<digits>).
func normalizePhone(raw string) string {
	digits := nonDigitRe.ReplaceAllString(strings.TrimSpace(raw), "")
	if len(digits) == 11 && strings.HasPrefix(digits, "8") {
		digits = "7" + digits[1:]
	}
	if len(digits) == 10 && strings.HasPrefix(digits, "9") {
		digits = "7" + digits
	}
	if digits == "" {
		return ""
	}
	return "+" + digits
}

// formatDateRu форматирует YYYY-MM-DD в "DD.MM (Пн)".
func formatDateRu(dateStr string) string {
	t, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
	if err != nil {
		return dateStr
	}
	weekdays := [...]string{"Вс", "Пн", "Вт", "Ср", "Чт", "Пт", "Сб"}
	return fmt.Sprintf("%s (%s)", t.Format("02.01"), weekdays[t.Weekday()])
}

// filterFutureSlots убирает прошедшие временные слоты, если выбран сегодняшний день.
func filterFutureSlots(dateStr string, slots []string) []string {
	now := time.Now()
	today := now.Format("2006-01-02")
	if dateStr != today {
		return slots
	}
	var filtered []string
	for _, s := range slots {
		slotDT, err := time.ParseInLocation("2006-01-02 15:04", dateStr+" "+s, time.Local)
		if err != nil {
			continue
		}
		if slotDT.After(now.Add(15 * time.Minute)) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}
