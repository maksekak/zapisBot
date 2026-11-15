package handlers

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"zapisBot/internal/bot"
	"zapisBot/internal/config"
	"zapisBot/internal/database"
	"zapisBot/internal/services"
)

// ClientHandler обрабатывает сценарии клиентов.
type ClientHandler struct {
	cfg      *config.Config
	db       *database.DB
	bot      *tgbotapi.BotAPI
	sessions *bot.SessionStore
	calc     *services.Calculator
	orders   *services.OrderService
	schedule *services.ScheduleService
	notify   *services.NotificationService
}

// NewClientHandler создаёт новый экземпляр ClientHandler.
func NewClientHandler(
	cfg *config.Config,
	db *database.DB,
	botAPI *tgbotapi.BotAPI,
	sessions *bot.SessionStore,
	calc *services.Calculator,
	orders *services.OrderService,
	schedule *services.ScheduleService,
	notify *services.NotificationService,
) *ClientHandler {
	return &ClientHandler{
		cfg:      cfg,
		db:       db,
		bot:      botAPI,
		sessions: sessions,
		calc:     calc,
		orders:   orders,
		schedule: schedule,
		notify:   notify,
	}
}

// HandleMessage обрабатывает входящие текстовые сообщения и контакты клиента.
func (h *ClientHandler) HandleMessage(update tgbotapi.Update) {
	msg := update.Message
	if msg == nil || msg.From == nil {
		return
	}
	chatID := msg.Chat.ID
	session := h.sessions.Get(chatID)
	h.syncClientFromDB(session, msg.From.ID)

	text := strings.TrimSpace(msg.Text)

	if text == "/start" {
		session.ResetOrderData()
		session.Lock()
		session.State = bot.StateMainMenu
		fullName := session.FullName
		session.Unlock()

		greeting := fmt.Sprintf("👋 Добро пожаловать в <b>%s</b>!\n\n"+
			"Мы занимаемся пескоструйной обработкой и порошковой покраской металлических изделий.",
			html.EscapeString(h.cfg.BotName))
		if fullName != "" {
			greeting = fmt.Sprintf("👋 Рады снова видеть вас, <b>%s</b>!\n\nВыберите действие в меню ниже:", html.EscapeString(fullName))
		}
		sendHTML(h.bot, chatID, greeting, mainMenuKeyboard())
		return
	}

	if text == btnCancel || text == "/cancel" {
		session.ResetOrderData()
		session.Lock()
		session.State = bot.StateMainMenu
		session.Unlock()
		sendHTML(h.bot, chatID, "Действие отменено. Вы в главном меню.", mainMenuKeyboard())
		return
	}

	switch text {
	case btnNewOrder:
		h.startOrderFlow(chatID, session)
		return
	case btnMyOrders:
		h.showMyOrders(chatID, session)
		return
	case btnCalcInfo:
		h.showServicesInfo(chatID)
		return
	}

	session.Lock()
	state := session.State
	session.Unlock()

	switch state {
	case bot.StateOrderParameters:
		if text == "" {
			sendHTML(h.bot, chatID, "Пожалуйста, введите примерные габариты или выберите вариант кнопкой.", nil)
			return
		}
		session.SetData("parameters", text)
		h.askColor(chatID, 0, session)
	case bot.StateOrderColor:
		if text == "" {
			sendHTML(h.bot, chatID, "Введите код цвета RAL (например, <code>RAL 9005</code>) или выберите из списка.", nil)
			return
		}
		session.SetData("color", text)
		h.askCoating(chatID, 0, session)
	case bot.StateRegistration:
		if len([]rune(text)) < 2 {
			sendHTML(h.bot, chatID, "Пожалуйста, введите корректное имя (минимум 2 символа):", nil)
			return
		}
		session.Lock()
		session.FullName = text
		inOrder := session.Data["pending_confirm"] == "1"
		hasPhone := session.Phone != ""
		session.Unlock()

		if inOrder {
			if hasPhone {
				h.finalizeOrder(chatID, session)
			} else {
				h.askContact(chatID, session)
			}
			return
		}
		session.Lock()
		session.State = bot.StateMainMenu
		session.Unlock()
		sendHTML(h.bot, chatID, fmt.Sprintf("Приятно познакомиться, <b>%s</b>!", html.EscapeString(text)), mainMenuKeyboard())
	case bot.StateOrderContact:
		if msg.Contact != nil {
			phone := normalizePhone(msg.Contact.PhoneNumber)
			if phone == "" {
				sendHTML(h.bot, chatID, "Не удалось распознать номер телефона. Нажмите кнопку «📱 Отправить номер телефона».", contactKeyboard())
				return
			}
			session.Lock()
			session.Phone = phone
			session.Unlock()
			h.finalizeOrder(chatID, session)
			return
		}
		sendHTML(h.bot, chatID, "Для подтверждения записи отправьте контакт с помощью кнопки «📱 Отправить номер телефона» ниже.", contactKeyboard())
	default:
		sendHTML(h.bot, chatID, "Выберите действие из меню ниже:", mainMenuKeyboard())
	}
}

// HandleCallback обрабатывает нажатия inline-кнопок клиента.
func (h *ClientHandler) HandleCallback(update tgbotapi.Update) {
	cb := update.CallbackQuery
	if cb == nil || cb.Message == nil {
		return
	}
	chatID := cb.Message.Chat.ID
	msgID := cb.Message.MessageID
	session := h.sessions.Get(chatID)
	h.syncClientFromDB(session, cb.From.ID)

	data := cb.Data
	answerCallback(h.bot, cb.ID, "", false)

	switch {
	case data == "noop":
		return
	case data == "flow:cancel":
		session.ResetOrderData()
		session.Lock()
		session.State = bot.StateMainMenu
		session.Unlock()
		editHTML(h.bot, chatID, msgID, "❌ Оформление отменено.", nil)
		sendHTML(h.bot, chatID, "Вы в главном меню.", mainMenuKeyboard())
	case strings.HasPrefix(data, "cat:"):
		session.SetData("category", strings.TrimPrefix(data, "cat:"))
		h.askItemType(chatID, msgID, session)
	case data == "back:cat":
		h.askCategory(chatID, msgID, session)
	case strings.HasPrefix(data, "item:"):
		session.SetData("item_type", strings.TrimPrefix(data, "item:"))
		h.askParameters(chatID, msgID, session)
	case data == "back:item":
		h.askItemType(chatID, msgID, session)
	case strings.HasPrefix(data, "param:"):
		session.SetData("parameters", strings.TrimPrefix(data, "param:"))
		h.askColor(chatID, msgID, session)
	case data == "back:param":
		h.askParameters(chatID, msgID, session)
	case strings.HasPrefix(data, "color:"):
		session.SetData("color", strings.TrimPrefix(data, "color:"))
		h.askCoating(chatID, msgID, session)
	case data == "back:color":
		h.askColor(chatID, msgID, session)
	case strings.HasPrefix(data, "coat:"):
		session.SetData("coating", strings.TrimPrefix(data, "coat:"))
		h.askDate(chatID, msgID, 0, session)
	case data == "back:coat":
		h.askCoating(chatID, msgID, session)
	case strings.HasPrefix(data, "datepage:"):
		page, _ := strconv.Atoi(strings.TrimPrefix(data, "datepage:"))
		h.askDate(chatID, msgID, page, session)
	case strings.HasPrefix(data, "date:"):
		session.SetData("date", strings.TrimPrefix(data, "date:"))
		h.askTime(chatID, msgID, session)
	case data == "back:date":
		h.askDate(chatID, msgID, 0, session)
	case strings.HasPrefix(data, "time:"):
		session.SetData("time_slot", strings.TrimPrefix(data, "time:"))
		h.showSummary(chatID, msgID, session)
	case data == "confirm:yes":
		h.handleConfirmOrder(chatID, msgID, session)
	case data == "confirm:no":
		session.ResetOrderData()
		session.Lock()
		session.State = bot.StateMainMenu
		session.Unlock()
		editHTML(h.bot, chatID, msgID, "❌ Запись отменена.", nil)
		sendHTML(h.bot, chatID, "Вы в главном меню.", mainMenuKeyboard())
	case strings.HasPrefix(data, "cancel:"):
		orderID, err := strconv.ParseInt(strings.TrimPrefix(data, "cancel:"), 10, 64)
		if err == nil {
			h.handleCancelOrder(chatID, msgID, session, orderID)
		}
	case strings.HasPrefix(data, "resched:"):
		orderID, err := strconv.ParseInt(strings.TrimPrefix(data, "resched:"), 10, 64)
		if err == nil {
			h.startReschedule(chatID, msgID, session, orderID)
		}
	case strings.HasPrefix(data, "rdatepage:"):
		page, _ := strconv.Atoi(strings.TrimPrefix(data, "rdatepage:"))
		h.askRescheduleDate(chatID, msgID, page, session)
	case strings.HasPrefix(data, "rdate:"):
		session.SetData("resched_date", strings.TrimPrefix(data, "rdate:"))
		h.askRescheduleTime(chatID, msgID, session)
	case strings.HasPrefix(data, "rtime:"):
		h.finishReschedule(chatID, msgID, session, strings.TrimPrefix(data, "rtime:"))
	}
}

func (h *ClientHandler) syncClientFromDB(session *bot.UserSession, telegramID int64) {
	session.Lock()
	alreadySynced := session.ClientID != 0
	session.Unlock()
	if alreadySynced || h.db == nil {
		return
	}
	c, err := h.db.ClientByTelegramID(telegramID)
	if err != nil || c == nil {
		return
	}
	session.Lock()
	session.ClientID = c.ID
	if session.FullName == "" {
		session.FullName = c.FullName
	}
	if session.Phone == "" {
		session.Phone = c.Phone
	}
	session.Unlock()
}

func (h *ClientHandler) showServicesInfo(chatID int64) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🏭 <b>%s — услуги и ориентировочные цены</b>\n\n", html.EscapeString(h.cfg.BotName)))
	sb.WriteString("Мы выполняем пескоструйную обработку и порошковую покраску:\n")
	for _, item := range []string{"диски", "детали авто", "мото-рамы", "суппорты", "металлоконструкции"} {
		if price, ok := h.cfg.BasePrices[item]; ok {
			sb.WriteString(fmt.Sprintf("• %s — от <b>%.0f ₽</b>\n", html.EscapeString(item), price))
		} else {
			sb.WriteString(fmt.Sprintf("• %s\n", html.EscapeString(item)))
		}
	}
	sb.WriteString("\nНажмите <b>«📝 Записаться»</b>, чтобы рассчитать точную стоимость и выбрать удобное время.")
	sendHTML(h.bot, chatID, sb.String(), mainMenuKeyboard())
}

