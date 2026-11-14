package handlers

import (
	"fmt"
	"html"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"zapisBot/internal/bot"
)

func (h *ClientHandler) showMyOrders(chatID int64, session *bot.UserSession) {
	session.Lock()
	clientID := session.ClientID
	session.Unlock()

	if clientID == 0 {
		sendHTML(h.bot, chatID, "📭 У вас нет активных записей.", mainMenuKeyboard())
		return
	}

	orders, err := h.db.GetOrderDetailsByClient(clientID)
	if err != nil {
		sendHTML(h.bot, chatID, "⚠️ Не удалось загрузить записи. Попробуйте позже.", mainMenuKeyboard())
		return
	}
	if len(orders) == 0 {
		sendHTML(h.bot, chatID, "📭 У вас нет активных записей.", mainMenuKeyboard())
		return
	}

	var activeText strings.Builder
	var archiveText strings.Builder
	var buttons [][]tgbotapi.InlineKeyboardButton

	activeCount := 0
	archiveCount := 0

	for _, o := range orders {
		priceStr := "уточняется у менеджера"
		if o.Price > 0 {
			priceStr = fmt.Sprintf("%.0f ₽", o.Price)
		}
		line := fmt.Sprintf("• <b>Заказ #%d</b> — %s %s\n  %s (%s, %s, %s, %s) — %s\n",
			o.ID, html.EscapeString(formatDateRu(o.Date)), html.EscapeString(o.TimeSlot),
			html.EscapeString(o.Category), html.EscapeString(o.ItemType),
			html.EscapeString(o.Parameters), html.EscapeString(o.Color),
			html.EscapeString(o.Coating), priceStr)

		if o.Status == "confirmed" {
			activeCount++
			activeText.WriteString(line)
			buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("🔄 Перенести #%d", o.ID), fmt.Sprintf("resched:%d", o.ID)),
				tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("❌ Отменить #%d", o.ID), fmt.Sprintf("cancel:%d", o.ID)),
			))
		} else {
			archiveCount++
			if archiveCount <= 5 {
				statusRu := "отменён"
				if o.Status == "completed" {
					statusRu = "выполнен"
				}
				archiveText.WriteString(fmt.Sprintf("• #%d (%s %s) — %s [%s]\n",
					o.ID, html.EscapeString(o.Date), html.EscapeString(o.TimeSlot),
					html.EscapeString(o.ItemType), statusRu))
			}
		}
	}

	var out strings.Builder
	out.WriteString("📋 <b>Ваши записи</b>\n\n")
	if activeCount > 0 {
		out.WriteString("<b>Активные:</b>\n" + activeText.String() + "\n")
	} else {
		out.WriteString("У вас нет активных записей.\n\n")
	}
	if archiveCount > 0 {
		out.WriteString("<b>Архив (последние):</b>\n" + archiveText.String())
	}

	if len(buttons) > 0 {
		kb := tgbotapi.NewInlineKeyboardMarkup(buttons...)
		sendHTML(h.bot, chatID, out.String(), kb)
	} else {
		sendHTML(h.bot, chatID, out.String(), mainMenuKeyboard())
	}
}

func (h *ClientHandler) handleCancelOrder(chatID int64, msgID int, session *bot.UserSession, orderID int64) {
	detail, err := h.db.GetOrderDetail(orderID)
	if err != nil || detail == nil {
		sendHTML(h.bot, chatID, "⚠️ Запись не найдена.", mainMenuKeyboard())
		return
	}
	session.Lock()
	clientID := session.ClientID
	session.Unlock()
	if detail.ClientID != clientID && detail.ClientTelegramID != chatID {
		sendHTML(h.bot, chatID, "⛔ Нет доступа к этой записи.", mainMenuKeyboard())
		return
	}
	if detail.Status != "confirmed" {
		sendHTML(h.bot, chatID, "Эта запись уже отменена или завершена.", mainMenuKeyboard())
		return
	}

	ok, reason := h.orders.CanCancel(detail.Date, detail.TimeSlot)
	if !ok {
		sendHTML(h.bot, chatID, "⚠️ "+html.EscapeString(reason), mainMenuKeyboard())
		return
	}

	if err := h.db.CancelOrder(orderID); err != nil {
		sendHTML(h.bot, chatID, fmt.Sprintf("⚠️ Не удалось отменить запись: %v", err), mainMenuKeyboard())
		return
	}

	editHTML(h.bot, chatID, msgID, fmt.Sprintf("❌ <b>Заказ #%d отменён.</b>\nСлот %s %s освобождён.", orderID, detail.Date, detail.TimeSlot), nil)
	if h.notify != nil {
		h.notify.SendAdminText(fmt.Sprintf("🗑 <b>Отмена записи #%d</b>\nКлиент: %s (%s)\nСлот: %s %s",
			orderID, html.EscapeString(detail.ClientName), html.EscapeString(detail.ClientPhone), detail.Date, detail.TimeSlot))
	}
}

func (h *ClientHandler) startReschedule(chatID int64, msgID int, session *bot.UserSession, orderID int64) {
	detail, err := h.db.GetOrderDetail(orderID)
	if err != nil || detail == nil {
		sendHTML(h.bot, chatID, "⚠️ Запись не найдена.", mainMenuKeyboard())
		return
	}
	session.Lock()
	clientID := session.ClientID
	session.Unlock()
	if detail.ClientID != clientID && detail.ClientTelegramID != chatID {
		sendHTML(h.bot, chatID, "⛔ Нет доступа к этой записи.", mainMenuKeyboard())
		return
	}
	if detail.Status != "confirmed" {
		sendHTML(h.bot, chatID, "Перенести можно только активную запись.", mainMenuKeyboard())
		return
	}

	ok, reason := h.orders.CanCancel(detail.Date, detail.TimeSlot)
	if !ok {
		sendHTML(h.bot, chatID, "⚠️ "+html.EscapeString(reason), mainMenuKeyboard())
		return
	}

	session.SetData("resched_order_id", fmt.Sprintf("%d", orderID))
	h.askRescheduleDate(chatID, msgID, 0, session)
}

func (h *ClientHandler) askRescheduleDate(chatID int64, msgID int, page int, session *bot.UserSession) {
	session.Lock()
	session.State = bot.StateRescheduleDate
	session.Unlock()

	kb, total := h.buildDateKeyboard(page, "rdate", "")
	title := "🔄 <b>Перенос записи. Выберите новую дату:</b>"
	if total == 0 {
		title = "⚠️ Нет свободных дат для переноса."
	}
	h.sendOrEdit(chatID, msgID, title, kb)
}

func (h *ClientHandler) askRescheduleTime(chatID int64, msgID int, session *bot.UserSession) {
	session.Lock()
	session.State = bot.StateRescheduleTime
	session.Unlock()

	date := session.GetData("resched_date")
	slots, err := h.db.GetFreeSlots(date)
	if err != nil {
		sendHTML(h.bot, chatID, "⚠️ Ошибка загрузки слотов.", mainMenuKeyboard())
		return
	}
	slots = filterFutureSlots(date, slots)
	if len(slots) == 0 {
		h.sendOrEdit(chatID, msgID, "⚠️ Свободного времени на эту дату нет. Выберите другую дату.", tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("◀️ Другая дата", "rdatepage:0")),
		))
		return
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < len(slots); i += 3 {
		var row []tgbotapi.InlineKeyboardButton
		for j := i; j < i+3 && j < len(slots); j++ {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData("🕒 "+slots[j], "rtime:"+slots[j]))
		}
		rows = append(rows, row)
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("◀️ Другая дата", "rdatepage:0"),
		tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "flow:cancel"),
	))
	h.sendOrEdit(chatID, msgID, fmt.Sprintf("🔄 <b>Выберите новое время на %s:</b>", html.EscapeString(formatDateRu(date))), tgbotapi.NewInlineKeyboardMarkup(rows...))
}

func (h *ClientHandler) finishReschedule(chatID int64, msgID int, session *bot.UserSession, newTime string) {
	idStr := session.GetData("resched_order_id")
	var orderID int64
	fmt.Sscanf(idStr, "%d", &orderID)
	newDate := session.GetData("resched_date")

	if orderID == 0 || newDate == "" {
		sendHTML(h.bot, chatID, "⚠️ Сессия переноса устарела. Попробуйте снова через «Мои записи».", mainMenuKeyboard())
		return
	}

	updated, err := h.db.RescheduleOrder(orderID, newDate, newTime)
	if err != nil {
		sendHTML(h.bot, chatID, fmt.Sprintf("⚠️ Не удалось перенести запись: %v", err), mainMenuKeyboard())
		return
	}

	session.ResetOrderData()
	session.Lock()
	session.State = bot.StateMainMenu
	session.Unlock()

	editHTML(h.bot, chatID, msgID,
		fmt.Sprintf("✅ <b>Запись #%d перенесена!</b>\nНовая дата: <b>%s</b>\nНовое время: <b>%s</b>",
			updated.ID, html.EscapeString(formatDateRu(updated.Date)), html.EscapeString(updated.TimeSlot)), nil)

	if h.notify != nil {
		h.notify.SendAdminText(fmt.Sprintf("🔄 <b>Перенос записи #%d</b>\nНовая дата и время: <b>%s %s</b>",
			updated.ID, updated.Date, updated.TimeSlot))
	}
}
