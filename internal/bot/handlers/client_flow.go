package handlers

import (
	"fmt"
	"html"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"zapisBot/internal/bot"
)

func (h *ClientHandler) sendOrEdit(chatID int64, msgID int, text string, kb tgbotapi.InlineKeyboardMarkup) {
	if msgID > 0 {
		editHTML(h.bot, chatID, msgID, text, &kb)
		return
	}
	sendHTML(h.bot, chatID, text, kb)
}

func (h *ClientHandler) startOrderFlow(chatID int64, session *bot.UserSession) {
	session.ResetOrderData()
	h.askCategory(chatID, 0, session)
}

func (h *ClientHandler) askCategory(chatID int64, msgID int, session *bot.UserSession) {
	session.Lock()
	session.State = bot.StateOrderCategory
	session.Unlock()

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💨 Пескоструй", "cat:пескоструй"),
			tgbotapi.NewInlineKeyboardButtonData("🎨 Покраска", "cat:покраска"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "flow:cancel"),
		),
	)
	h.sendOrEdit(chatID, msgID, "<b>Шаг 1/7. Выберите категорию услуги:</b>", kb)
}

func (h *ClientHandler) askItemType(chatID int64, msgID int, session *bot.UserSession) {
	session.Lock()
	session.State = bot.StateOrderItemType
	session.Unlock()

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🛞 Диски", "item:диски"),
			tgbotapi.NewInlineKeyboardButtonData("🚗 Детали авто", "item:детали авто"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏍 Мото-рамы", "item:мото-рамы"),
			tgbotapi.NewInlineKeyboardButtonData("🛑 Суппорты", "item:суппорты"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏗 Металлоконструкции", "item:металлоконструкции"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "back:cat"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "flow:cancel"),
		),
	)
	h.sendOrEdit(chatID, msgID, "<b>Шаг 2/7. Выберите тип изделия:</b>", kb)
}

func (h *ClientHandler) askParameters(chatID int64, msgID int, session *bot.UserSession) {
	session.Lock()
	session.State = bot.StateOrderParameters
	session.Unlock()

	itemType := session.GetData("item_type")
	if itemType == "диски" {
		var rows [][]tgbotapi.InlineKeyboardButton
		for start := 13; start <= 24; start += 4 {
			var row []tgbotapi.InlineKeyboardButton
			for r := start; r < start+4 && r <= 24; r++ {
				val := fmt.Sprintf("R%d", r)
				row = append(row, tgbotapi.NewInlineKeyboardButtonData(val, "param:"+val))
			}
			rows = append(rows, row)
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "back:item"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "flow:cancel"),
		))
		h.sendOrEdit(chatID, msgID, "<b>Шаг 3/7. Выберите радиус дисков (R13–R24):</b>", tgbotapi.NewInlineKeyboardMarkup(rows...))
		return
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Малые (до 30 см)", "param:Малые (до 30 см)"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Средние (30–100 см)", "param:Средние (30-100 см)"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Крупные (более 100 см)", "param:Крупные (>100 см)"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "back:item"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "flow:cancel"),
		),
	)
	h.sendOrEdit(chatID, msgID, "<b>Шаг 3/7. Выберите примерные габариты</b> или введите размеры текстом (например, <code>120x60 см</code>):", kb)
}

func (h *ClientHandler) askColor(chatID int64, msgID int, session *bot.UserSession) {
	session.Lock()
	session.State = bot.StateOrderColor
	session.Unlock()

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("RAL 9005 (Чёрный)", "color:RAL 9005"),
			tgbotapi.NewInlineKeyboardButtonData("RAL 9003 (Белый)", "color:RAL 9003"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("RAL 7016 (Антрацит)", "color:RAL 7016"),
			tgbotapi.NewInlineKeyboardButtonData("RAL 9006 (Серебро)", "color:RAL 9006"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("RAL 3020 (Красный)", "color:RAL 3020"),
			tgbotapi.NewInlineKeyboardButtonData("RAL 5002 (Синий)", "color:RAL 5002"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Без покраски (пескоструй)", "color:Без покраски"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "back:param"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "flow:cancel"),
		),
	)
	h.sendOrEdit(chatID, msgID, "<b>Шаг 4/7. Выберите цвет RAL</b> (или отправьте свой код RAL сообщением):", kb)
}

func (h *ClientHandler) askCoating(chatID int64, msgID int, session *bot.UserSession) {
	session.Lock()
	session.State = bot.StateOrderCoating
	session.Unlock()

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Матовый", "coat:матовый"),
			tgbotapi.NewInlineKeyboardButtonData("Глянец", "coat:глянец"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "back:color"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "flow:cancel"),
		),
	)
	h.sendOrEdit(chatID, msgID, "<b>Шаг 5/7. Выберите тип покрытия:</b>", kb)
}

func (h *ClientHandler) buildDateKeyboard(page int, prefix string, backCallback string) (tgbotapi.InlineKeyboardMarkup, int) {
	today := time.Now().Format("2006-01-02")
	avail, err := h.db.GetAvailableDates(today, 30)
	if err != nil {
		avail = map[string]int{}
	}

	const perPage = 10
	startIdx := page * perPage
	if startIdx < 0 {
		startIdx = 0
	}
	now := time.Now()
	var dates []string
	for i := 0; i < 30; i++ {
		d := now.AddDate(0, 0, i).Format("2006-01-02")
		if cnt := avail[d]; cnt > 0 {
			if d == today {
				slots, _ := h.db.GetFreeSlots(d)
				if len(filterFutureSlots(d, slots)) == 0 {
					continue
				}
			}
			dates = append(dates, d)
		}
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	if len(dates) == 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Нет свободных дат", "noop"),
		))
	} else {
		endIdx := startIdx + perPage
		if endIdx > len(dates) {
			endIdx = len(dates)
		}
		if startIdx >= len(dates) {
			startIdx = 0
		}
		slice := dates[startIdx:endIdx]
		for i := 0; i < len(slice); i += 2 {
			var row []tgbotapi.InlineKeyboardButton
			row = append(row, tgbotapi.NewInlineKeyboardButtonData("📅 "+formatDateRu(slice[i]), prefix+":"+slice[i]))
			if i+1 < len(slice) {
				row = append(row, tgbotapi.NewInlineKeyboardButtonData("📅 "+formatDateRu(slice[i+1]), prefix+":"+slice[i+1]))
			}
			rows = append(rows, row)
		}
		var nav []tgbotapi.InlineKeyboardButton
		if page > 0 {
			nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("%spage:%d", prefix, page-1)))
		}
		if endIdx < len(dates) {
			nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("Вперёд ➡️", fmt.Sprintf("%spage:%d", prefix, page+1)))
		}
		if len(nav) > 0 {
			rows = append(rows, nav)
		}
	}

	var footer []tgbotapi.InlineKeyboardButton
	if backCallback != "" {
		footer = append(footer, tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", backCallback))
	}
	footer = append(footer, tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "flow:cancel"))
	rows = append(rows, footer)
	return tgbotapi.NewInlineKeyboardMarkup(rows...), len(dates)
}

func (h *ClientHandler) askDate(chatID int64, msgID int, page int, session *bot.UserSession) {
	session.Lock()
	session.State = bot.StateOrderDate
	session.Unlock()

	kb, total := h.buildDateKeyboard(page, "date", "back:coat")
	title := "<b>Шаг 6/7. Выберите удобную дату визита:</b>"
	if total == 0 {
		title = "⚠️ К сожалению, на ближайшие 30 дней нет свободных слотов."
	}
	h.sendOrEdit(chatID, msgID, title, kb)
}

func (h *ClientHandler) askTime(chatID int64, msgID int, session *bot.UserSession) {
	session.Lock()
	session.State = bot.StateOrderTime
	session.Unlock()

	date := session.GetData("date")
	slots, err := h.db.GetFreeSlots(date)
	if err != nil {
		h.sendOrEdit(chatID, msgID, "⚠️ Ошибка загрузки слотов. Попробуйте выбрать другую дату.", tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("◀️ Другая дата", "back:date")),
		))
		return
	}
	slots = filterFutureSlots(date, slots)
	if len(slots) == 0 {
		h.sendOrEdit(chatID, msgID, "⚠️ На эту дату свободных слотов больше нет. Выберите другую дату.", tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("◀️ Другая дата", "back:date")),
		))
		return
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < len(slots); i += 3 {
		var row []tgbotapi.InlineKeyboardButton
		for j := i; j < i+3 && j < len(slots); j++ {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData("🕒 "+slots[j], "time:"+slots[j]))
		}
		rows = append(rows, row)
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("◀️ Другая дата", "back:date"),
		tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "flow:cancel"),
	))
	h.sendOrEdit(chatID, msgID, fmt.Sprintf("<b>Шаг 6/7. Свободное время на %s:</b>", html.EscapeString(formatDateRu(date))), tgbotapi.NewInlineKeyboardMarkup(rows...))
}

func (h *ClientHandler) showSummary(chatID int64, msgID int, session *bot.UserSession) {
	session.Lock()
	session.State = bot.StateOrderConfirm
	cat := session.Data["category"]
	item := session.Data["item_type"]
	params := session.Data["parameters"]
	color := session.Data["color"]
	coating := session.Data["coating"]
	date := session.Data["date"]
	slot := session.Data["time_slot"]
	session.Unlock()

	calcRes := h.calc.Calculate(item, params, coating)
	if calcRes.HasPrice {
		session.SetData("price", fmt.Sprintf("%.2f", calcRes.Total))
	} else {
		session.SetData("price", "0")
	}

	summary := fmt.Sprintf(
		"<b>Шаг 7/7. Проверьте детали записи:</b>\n\n"+
			"%s\n\n"+
			"📅 Дата: <b>%s</b>\n"+
			"🕒 Время: <b>%s</b>\n\n"+
			"Подтвердить запись?",
		h.calc.FormatResult(calcRes, cat, item, params, coating, color),
		html.EscapeString(formatDateRu(date)),
		html.EscapeString(slot),
	)

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Подтвердить запись", "confirm:yes"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "confirm:no"),
		),
	)
	h.sendOrEdit(chatID, msgID, summary, kb)
}

func (h *ClientHandler) handleConfirmOrder(chatID int64, msgID int, session *bot.UserSession) {
	session.SetData("pending_confirm", "1")
	session.Lock()
	fullName := session.FullName
	phone := session.Phone
	session.Unlock()

	if fullName == "" {
		session.Lock()
		session.State = bot.StateRegistration
		session.Unlock()
		editHTML(h.bot, chatID, msgID, "Для завершения записи укажите ваше <b>имя</b>:", nil)
		return
	}
	if phone == "" {
		editHTML(h.bot, chatID, msgID, "Почти готово! Осталось подтвердить номер телефона.", nil)
		h.askContact(chatID, session)
		return
	}
	h.finalizeOrder(chatID, session)
}

func (h *ClientHandler) askContact(chatID int64, session *bot.UserSession) {
	session.Lock()
	session.State = bot.StateOrderContact
	session.Unlock()
	sendHTML(h.bot, chatID, "📱 Нажмите кнопку <b>«Отправить номер телефона»</b> ниже для подтверждения записи:", contactKeyboard())
}

func (h *ClientHandler) finalizeOrder(chatID int64, session *bot.UserSession) {
	session.Lock()
	fullName := session.FullName
	phone := session.Phone
	cat := session.Data["category"]
	item := session.Data["item_type"]
	params := session.Data["parameters"]
	color := session.Data["color"]
	coating := session.Data["coating"]
	date := session.Data["date"]
	slot := session.Data["time_slot"]
	priceStr := session.Data["price"]
	session.Unlock()

	var price float64
	fmt.Sscanf(priceStr, "%f", &price)

	// 1. Проверяем лимит: максимум 2 активные записи на один номер телефона
	allowed, reason := h.orders.CheckPhoneLimit(phone)
	if !allowed {
		session.ResetOrderData()
		session.Lock()
		session.State = bot.StateMainMenu
		session.Unlock()
		sendHTML(h.bot, chatID, "⛔ "+html.EscapeString(reason), mainMenuKeyboard())
		return
	}

	// 2. Создаём или обновляем клиента в БД
	client, err := h.db.UpsertClient(fullName, phone, chatID)
	if err != nil {
		sendHTML(h.bot, chatID, "⚠️ Не удалось сохранить данные клиента. Попробуйте позже.", mainMenuKeyboard())
		return
	}
	session.Lock()
	session.ClientID = client.ID
	session.Unlock()

	// 3. Создаём заказ в транзакции
	order, err := h.db.CreateOrder(client.ID, cat, item, params, color, coating, date, slot, price)
	if err != nil {
		sendHTML(h.bot, chatID, fmt.Sprintf("⚠️ Не удалось оформить запись: %s\nПопробуйте выбрать другое время.", html.EscapeString(err.Error())), mainMenuKeyboard())
		return
	}

	session.ResetOrderData()
	session.Lock()
	session.State = bot.StateMainMenu
	session.Unlock()

	priceText := "уточняется у менеджера"
	if order.Price > 0 {
		priceText = fmt.Sprintf("%.0f ₽", order.Price)
	}

	confirmMsg := fmt.Sprintf(
		"✅ <b>Запись #%d подтверждена!</b>\n\n"+
			"👤 Клиент: %s (%s)\n"+
			"🔧 Услуга: %s — %s (%s)\n"+
			"🎨 Цвет: %s, покрытие: %s\n"+
			"📅 Дата: <b>%s</b>\n"+
			"🕒 Время: <b>%s</b>\n"+
			"💰 Стоимость: <b>%s</b>\n\n"+
			"Мы напомним вам о визите за 24 часа и за 2 часа.",
		order.ID,
		html.EscapeString(fullName), html.EscapeString(phone),
		html.EscapeString(cat), html.EscapeString(item), html.EscapeString(params),
		html.EscapeString(color), html.EscapeString(coating),
		html.EscapeString(formatDateRu(order.Date)), html.EscapeString(order.TimeSlot),
		priceText,
	)
	sendHTML(h.bot, chatID, confirmMsg, mainMenuKeyboard())

	if h.notify != nil {
		adminOrderInfo := fmt.Sprintf(
			"Заказ: <b>#%d</b>\nКлиент: %s (%s)\nУслуга: %s — %s (%s)\nЦвет: %s, %s\nДата и время: <b>%s %s</b>\nСтоимость: <b>%s</b>",
			order.ID,
			html.EscapeString(fullName), html.EscapeString(phone),
			html.EscapeString(cat), html.EscapeString(item), html.EscapeString(params),
			html.EscapeString(color), html.EscapeString(coating),
			order.Date, order.TimeSlot, priceText,
		)
		h.notify.SendAdminNotify(chatID, adminOrderInfo)
	}
}
