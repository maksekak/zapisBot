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

func (h *AdminHandler) showDaySchedule(chatID int64, date string) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		t = time.Now()
		date = t.Format("2006-01-02")
	}

	prevDate := t.AddDate(0, 0, -1).Format("2006-01-02")
	nextDate := t.AddDate(0, 0, 1).Format("2006-01-02")

	orders, err := h.db.GetOrdersDetailForDate(date)
	if err != nil {
		sendHTML(h.bot, chatID, fmt.Sprintf("Ошибка получения заказов: %v", err), adminMenuKeyboard())
		return
	}

	grid, err := h.db.GetScheduleGrid(date)
	if err != nil {
		sendHTML(h.bot, chatID, fmt.Sprintf("Ошибка получения сетки расписания: %v", err), adminMenuKeyboard())
		return
	}

	ordersMap := make(map[string]database.OrderDetail)
	for _, o := range orders {
		ordersMap[o.TimeSlot] = o
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📅 <b>Расписание на %s</b>\n\n", html.EscapeString(formatDateRu(date))))

	if len(grid) == 0 {
		sb.WriteString("Слоты не инициализированы на этот день.\n")
	} else {
		for _, s := range grid {
			if !s.IsWorkday {
				sb.WriteString(fmt.Sprintf("🔴 <b>%s</b> — Выходной день\n", s.TimeSlot))
				continue
			}
			if s.IsOccupied {
				if o, ok := ordersMap[s.TimeSlot]; ok {
					sb.WriteString(fmt.Sprintf("🔴 <b>%s</b> — #%d %s (%s)\n  └ %s %s (%s), %s, %s\n",
						s.TimeSlot, o.ID, html.EscapeString(o.ClientName), html.EscapeString(o.ClientPhone),
						html.EscapeString(o.Category), html.EscapeString(o.ItemType), html.EscapeString(o.Parameters),
						html.EscapeString(o.Color), html.EscapeString(o.Coating)))
				} else {
					sb.WriteString(fmt.Sprintf("🔴 <b>%s</b> — Занято\n", s.TimeSlot))
				}
			} else {
				sb.WriteString(fmt.Sprintf("🟢 <b>%s</b> — Свободно\n", s.TimeSlot))
			}
		}
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Пред. день", "adm:day:"+prevDate),
			tgbotapi.NewInlineKeyboardButtonData("След. день ▶️", "adm:day:"+nextDate),
		),
	)

	sendHTML(h.bot, chatID, sb.String(), kb)
}
func (h *AdminHandler) showWeekSchedule(chatID int64) {
	now := time.Now()
	var sb strings.Builder
	sb.WriteString("📆 <b>Расписание на ближайшую неделю:</b>\n\n")

	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton

	for i := 0; i < 7; i++ {
		d := now.AddDate(0, 0, i)
		dateStr := d.Format("2006-01-02")
		orders, _ := h.db.GetOrdersDetailForDate(dateStr)
		isWorkday, _ := h.db.IsWorkday(dateStr)

		status := "🟢 Рабочий"
		if !isWorkday {
			status = "🔴 Выходной"
		}

		sb.WriteString(fmt.Sprintf("• <b>%s</b> (%s): Записей: <b>%d</b> | %s\n",
			html.EscapeString(formatDateRu(dateStr)), dateStr, len(orders), status))

		btnText := fmt.Sprintf("%s (%d)", d.Format("02.01"), len(orders))
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(btnText, "adm:day:"+dateStr))
		if len(row) == 3 || i == 6 {
			rows = append(rows, row)
			row = []tgbotapi.InlineKeyboardButton{}
		}
	}

	sendHTML(h.bot, chatID, sb.String(), tgbotapi.NewInlineKeyboardMarkup(rows...))
}

func (h *AdminHandler) showWorkdaysMenu(chatID int64, page int, session *bot.UserSession) {
	session.Lock()
	session.State = bot.StateAdminWorkday
	session.Unlock()

	pageSize := 7
	totalDays := 28
	if page < 0 {
		page = 0
	}
	maxPage := (totalDays - 1) / pageSize

	now := time.Now()
	startIdx := page * pageSize

	var sb strings.Builder
	sb.WriteString("🛠 <b>Управление рабочими / выходными днями</b>\n\n")
	sb.WriteString("Нажмите на день, чтобы переключить статус (Рабочий / Выходной).\n")
	sb.WriteString("Или отправьте дату и статус текстом: <code>2026-11-20 off</code> или <code>2026-11-20 on</code>.\n\n")

	var rows [][]tgbotapi.InlineKeyboardButton

	for i := startIdx; i < startIdx+pageSize && i < totalDays; i++ {
		d := now.AddDate(0, 0, i)
		dateStr := d.Format("2006-01-02")

		isWorkday, _ := h.db.IsWorkday(dateStr)

		statusIcon := "🟢"
		statusText := "Рабочий"
		if !isWorkday {
			statusIcon = "🔴"
			statusText = "Выходной"
		}

		sb.WriteString(fmt.Sprintf("%s <b>%s</b> — %s\n", statusIcon, html.EscapeString(formatDateRu(dateStr)), statusText))

		btnLabel := fmt.Sprintf("%s %s (%s)", statusIcon, d.Format("02.01"), statusText)
		toggleData := fmt.Sprintf("adm:workday:toggle:%s:%d", dateStr, page)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(btnLabel, toggleData),
		))
	}

	var navRow []tgbotapi.InlineKeyboardButton
	if page > 0 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", fmt.Sprintf("adm:workdaypage:%d", page-1)))
	}
	if page < maxPage {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("Вперёд ▶️", fmt.Sprintf("adm:workdaypage:%d", page+1)))
	}
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)
	sendHTML(h.bot, chatID, sb.String(), kb)
}

func (h *AdminHandler) showPriceListMenu(chatID int64, page int, session *bot.UserSession) {
	session.Lock()
	session.State = bot.StateAdminPriceCRUD
	session.Unlock()

	prices, err := h.db.GetAllPrices()
	if err != nil {
		sendHTML(h.bot, chatID, fmt.Sprintf("Ошибка загрузки прайс-листа: %v", err), adminMenuKeyboard())
		return
	}

	pageSize := 5
	if page < 0 {
		page = 0
	}
	total := len(prices)
	maxPage := 0
	if total > 0 {
		maxPage = (total - 1) / pageSize
	}

	var sb strings.Builder
	sb.WriteString("💰 <b>Управление прайс-листом (CRUD)</b>\n\n")

	if total == 0 {
		sb.WriteString("Прайс-лист пока пуст.\n\n")
	} else {
		startIdx := page * pageSize
		endIdx := startIdx + pageSize
		if endIdx > total {
			endIdx = total
		}

		for i := startIdx; i < endIdx; i++ {
			p := prices[i]
			sb.WriteString(fmt.Sprintf("• <b>#%d</b> %s | %s | %s — <b>%.0f ₽</b>\n",
				p.ID, html.EscapeString(p.ItemType), html.EscapeString(p.Parameters), html.EscapeString(p.Coating), p.Price))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Для добавления или обновления тарифа отправьте сообщение в формате:\n")
	sb.WriteString("<code>тип_изделия; параметры; покрытие; цена</code>\n")
	sb.WriteString("<i>Пример:</i> <code>диски; R17; глянец; 4500</code>\n")

	var rows [][]tgbotapi.InlineKeyboardButton
	startIdx := page * pageSize
	endIdx := startIdx + pageSize
	if endIdx > total {
		endIdx = total
	}

	for i := startIdx; i < endIdx; i++ {
		p := prices[i]
		btnText := fmt.Sprintf("❌ Удалить #%d (%s %.0f₽)", p.ID, p.ItemType, p.Price)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(btnText, fmt.Sprintf("adm:price:del:%d", p.ID)),
		))
	}

	var navRow []tgbotapi.InlineKeyboardButton
	if page > 0 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", fmt.Sprintf("adm:pricepage:%d", page-1)))
	}
	if page < maxPage {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("Вперёд ▶️", fmt.Sprintf("adm:pricepage:%d", page+1)))
	}
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)
	sendHTML(h.bot, chatID, sb.String(), kb)
}

func (h *AdminHandler) handlePriceInput(chatID int64, text string) {
	parts := strings.Split(text, ";")
	if len(parts) < 4 {
		sendHTML(h.bot, chatID, "Формат ввода: <code>тип_изделия; параметры; покрытие; цена</code>\nПример: <code>диски; R17; глянец; 4500</code>", nil)
		return
	}

	itemType := strings.TrimSpace(parts[0])
	parameters := strings.TrimSpace(parts[1])
	coating := strings.TrimSpace(parts[2])
	priceStr := strings.TrimSpace(parts[3])

	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil || price < 0 {
		sendHTML(h.bot, chatID, "Некорректная цена. Укажите число в рублях (например, 4500).", nil)
		return
	}

	if err := h.db.UpsertPriceList(itemType, parameters, coating, price); err != nil {
		sendHTML(h.bot, chatID, fmt.Sprintf("Ошибка сохранения прайс-листа: %v", err), nil)
		return
	}

	sendHTML(h.bot, chatID, fmt.Sprintf("✅ Прайс обновлён!\n<b>%s</b> (%s, %s) — <b>%.0f ₽</b>",
		html.EscapeString(itemType), html.EscapeString(parameters), html.EscapeString(coating), price), adminMenuKeyboard())
}

func (h *AdminHandler) handleWorkdayInput(chatID int64, text string) {
	parts := strings.Fields(text)
	if len(parts) < 2 {
		sendHTML(h.bot, chatID, "Формат ввода: <code>YYYY-MM-DD on</code> или <code>YYYY-MM-DD off</code>.", nil)
		return
	}
	dateStr := parts[0]
	cmd := strings.ToLower(parts[1])

	_, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		sendHTML(h.bot, chatID, "Некорректный формат даты. Используйте <code>YYYY-MM-DD</code>.", nil)
		return
	}

	isWorkday := true
	if cmd == "off" || cmd == "false" || cmd == "выходной" || cmd == "0" {
		isWorkday = false
	}

	if err := h.db.SetWorkday(dateStr, isWorkday); err != nil {
		sendHTML(h.bot, chatID, fmt.Sprintf("Ошибка обновления даты %s: %v", dateStr, err), nil)
		return
	}

	statusText := "рабочий"
	if !isWorkday {
		statusText = "выходной"
	}
	sendHTML(h.bot, chatID, fmt.Sprintf("✅ Статус даты <b>%s</b> изменён на <b>%s</b>.", html.EscapeString(formatDateRu(dateStr)), statusText), adminMenuKeyboard())
}

// HandleMessage обрабатывает текстовые команды администратора.
func (h *AdminHandler) HandleMessage(update tgbotapi.Update) {
	msg := update.Message
	if msg == nil || msg.From == nil {
		return
	}
	chatID := msg.Chat.ID
	if !h.cfg.IsAdmin(msg.From.ID) {
		sendHTML(h.bot, chatID, "Доступ запрещён.", mainMenuKeyboard())
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
		sendHTML(h.bot, chatID, "Выберите действие в меню администратора:", adminMenuKeyboard())
	}
}

// HandleCallback обрабатывает нажатия inline-кнопок администратора.
func (h *AdminHandler) HandleCallback(update tgbotapi.Update) {
	cb := update.CallbackQuery
	if cb == nil || cb.Message == nil {
		return
	}
	chatID := cb.Message.Chat.ID
	msgID := cb.Message.MessageID
	session := h.sessions.Get(chatID)

	if !h.cfg.IsAdmin(cb.From.ID) {
		answerCallback(h.bot, cb.ID, "Доступ запрещён.", true)
		return
	}

	data := cb.Data
	answerCallback(h.bot, cb.ID, "", false)

	switch {
	case data == "noop":
		return
	case strings.HasPrefix(data, "adm:day:"):
		dateStr := strings.TrimPrefix(data, "adm:day:")
		session.Lock()
		session.State = bot.StateAdminMenu
		session.Unlock()
		h.showDaySchedule(chatID, dateStr)
	case strings.HasPrefix(data, "adm:pricepage:"):
		page, err := strconv.Atoi(strings.TrimPrefix(data, "adm:pricepage:"))
		if err != nil {
			page = 0
		}
		session.Lock()
		session.State = bot.StateAdminPriceCRUD
		session.Unlock()
		h.showPriceListMenu(chatID, page, session)
	case strings.HasPrefix(data, "adm:price:del:"):
		idStr := strings.TrimPrefix(data, "adm:price:del:")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err == nil {
			if err := h.db.DeletePriceList(id); err != nil {
				editHTML(h.bot, chatID, msgID, fmt.Sprintf("🗑 Ошибка удаления: %v", err), nil)
			} else {
				editHTML(h.bot, chatID, msgID, "✅ Тариф удалён.", nil)
				h.showPriceListMenu(chatID, 0, session)
			}
		}
	case strings.HasPrefix(data, "adm:workday:toggle:"):
		prefix := "adm:workday:toggle:"
		rest := strings.TrimPrefix(data, prefix)
		idx := strings.LastIndex(rest, ":")
		if idx < 0 {
			return
		}
		dateStr := rest[:idx]
		pageStr := rest[idx+1:]
		page, _ := strconv.Atoi(pageStr)

		isWorkday, _ := h.db.IsWorkday(dateStr)
		if err := h.db.SetWorkday(dateStr, !isWorkday); err != nil {
			answerCallback(h.bot, cb.ID, fmt.Sprintf("Ошибка: %v", err), true)
			return
		}
		session.Lock()
		session.State = bot.StateAdminWorkday
		session.Unlock()
		h.showWorkdaysMenu(chatID, page, session)
	case strings.HasPrefix(data, "adm:workdaypage:"):
		page, err := strconv.Atoi(strings.TrimPrefix(data, "adm:workdaypage:"))
		if err != nil {
			page = 0
		}
		session.Lock()
		session.State = bot.StateAdminWorkday
		session.Unlock()
		h.showWorkdaysMenu(chatID, page, session)
	}
}