package handlers

import (
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"zapisBot/internal/bot"
	"zapisBot/internal/config"
	"zapisBot/internal/database"
)

const (
	adminBtnToday   = "📅 Расписание: сегодня"
	adminBtnWeek    = "📆 Расписание: неделя"
	adminBtnWorkday = "🛠 Рабочие / выходные дни"
	adminBtnPrices  = "💰 Прайс-лист (CRUD)"
	adminBtnExit    = "🚪 Выйти из админки"
)

// AdminHandler обрабатывает команды и меню администраторов.
type AdminHandler struct {
	cfg      *config.Config
	db       *database.DB
	bot      *tgbotapi.BotAPI
	sessions *bot.SessionStore
}

// NewAdminHandler создаёт новый экземпляр AdminHandler.
func NewAdminHandler(cfg *config.Config, db *database.DB, botAPI *tgbotapi.BotAPI, sessions *bot.SessionStore) *AdminHandler {
	return &AdminHandler{
		cfg:      cfg,
		db:       db,
		bot:      botAPI,
		sessions: sessions,
	}
}

func adminMenuKeyboard() tgbotapi.ReplyKeyboardMarkup {
	kb := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(adminBtnToday),
			tgbotapi.NewKeyboardButton(adminBtnWeek),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(adminBtnWorkday),
			tgbotapi.NewKeyboardButton(adminBtnPrices),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(adminBtnExit),
		),
	)
	kb.ResizeKeyboard = true
	return kb
}

// HandleMessage обрабатывает текстовые команды администратора.
func (h *AdminHandler) HandleMessage(update tgbotapi.Update) {
	msg := update.Message
	if msg == nil || msg.From == nil {
		return
	}
	chatID := msg.Chat.ID
	if !h.cfg.IsAdmin(msg.From.ID) {
		sendHTML(h.bot, chatID, "⛔ Доступ запрещён.", mainMenuKeyboard())
		return
	}

	session := h.sessions.Get(chatID)
	text := strings.TrimSpace(msg.Text)

	if text == "/admin" {
		session.Lock()
		session.State = bot.StateAdminMenu
		session.Unlock()
		sendHTML(h.bot, chatID, "⚙️ <b>Панель администратора</b>\nВыберите раздел:", adminMenuKeyboard())
		return
	}

	switch text {
	case adminBtnExit, "/start":
		session.Lock()
		session.State = bot.StateMainMenu
		session.Unlock()
		sendHTML(h.bot, chatID, "Вы вышли из панели администратора.", mainMenuKeyboard())
		return
	case adminBtnToday:
		session.Lock()
		session.State = bot.StateAdminMenu
		session.Unlock()
		h.showDaySchedule(chatID, time.Now().Format("2006-01-02"))
		return
	case adminBtnWeek:
		session.Lock()
		session.State = bot.StateAdminMenu
		session.Unlock()
		h.showWeekSchedule(chatID)
		return
	case adminBtnWorkday:
		h.showWorkdaysMenu(chatID, 0, session)
		return
	case adminBtnPrices:
		h.showPriceListMenu(chatID, 0, session)
		return
	}

	session.Lock()
	state := session.State
	session.Unlock()

	switch state {
	case bot.StateAdminPriceCRUD:
		h.handlePriceInput(chatID, text)
	case bot.StateAdminWorkday:
		h.handleWorkdayInput(chatID, text)
	default:
		sendHTML(h.bot, chatID, "⚙️ Выберите действие в меню администратора:", adminMenuKeyboard())
	}
}
