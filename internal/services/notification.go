package services

import (
	"fmt"
	"html"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"zapisBot/internal/config"
	"zapisBot/internal/database"
)

// NotificationService — сервис уведомлений (напоминания клиентам и админам).
type NotificationService struct {
	cfg *config.Config
	db  *database.DB
	bot *tgbotapi.BotAPI
}

// NewNotificationService создаёт новый сервис уведомлений.
func NewNotificationService(cfg *config.Config, db *database.DB, bot *tgbotapi.BotAPI) *NotificationService {
	return &NotificationService{cfg: cfg, db: db, bot: bot}
}

// SendAdminNotify отправляет мгновенное уведомление админам о новом заказе.
func (n *NotificationService) SendAdminNotify(chatID int64, orderInfo string) {
	if n.bot == nil {
		return
	}
	for _, adminID := range n.cfg.AdminIDs {
		msg := tgbotapi.NewMessage(adminID, fmt.Sprintf(
			"🔔 <b>Новая запись!</b>\n\nTelegram ID клиента: <code>%d</code>\n%s",
			chatID, orderInfo,
		))
		msg.ParseMode = tgbotapi.ModeHTML
		if _, err := n.bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки уведомления админу %d: %v", adminID, err)
		}
	}
}

// SendAdminText отправляет произвольное HTML-уведомление админам (например, об отмене/переносе).
func (n *NotificationService) SendAdminText(text string) {
	if n.bot == nil {
		return
	}
	for _, adminID := range n.cfg.AdminIDs {
		msg := tgbotapi.NewMessage(adminID, text)
		msg.ParseMode = tgbotapi.ModeHTML
		if _, err := n.bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки уведомления админу %d: %v", adminID, err)
		}
	}
}

// SendReminder отправляет напоминание клиенту о предстоящей записи.
func (n *NotificationService) SendReminder(clientTelegramID int64, orderDate, orderTime, orderInfo, leadText string) error {
	if n.bot == nil || clientTelegramID == 0 {
		return nil
	}
	msg := tgbotapi.NewMessage(clientTelegramID, fmt.Sprintf(
		"📅 <b>Напоминание о записи (%s)</b>\n\n%s\n\n📆 Дата: %s\n🕒 Время: %s\n\nЖдём вас!",
		html.EscapeString(leadText), orderInfo, html.EscapeString(orderDate), html.EscapeString(orderTime),
	))
	msg.ParseMode = tgbotapi.ModeHTML
	if _, err := n.bot.Send(msg); err != nil {
		// По ТЗ: если клиент заблокировал бота — записываем в лог, повторная отправка не требуется.
		log.Printf("Ошибка отправки напоминания клиенту %d: %v", clientTelegramID, err)
		return err
	}
	return nil
}

// CheckReminders проверяет и отправляет напоминания за 24 часа и за 2 часа до визита.
func (n *NotificationService) CheckReminders() error {
	now := time.Now()
	fromDate := now.Format("2006-01-02")
	toDate := now.Add(26 * time.Hour).Format("2006-01-02")

	orders, err := n.db.GetPendingReminders(fromDate, toDate)
	if err != nil {
		return fmt.Errorf("получение заказов для напоминаний: %w", err)
	}

	for _, o := range orders {
		visitDT, err := time.ParseInLocation("2006-01-02 15:04", o.Date+" "+o.TimeSlot, time.Local)
		if err != nil {
			continue
		}
		until := visitDT.Sub(now)
		if until <= 0 {
			continue
		}

		orderInfo := fmt.Sprintf(
			"Заказ #%d\nУслуга: %s — %s\nПараметры: %s\nЦвет: %s, покрытие: %s",
			o.ID,
			html.EscapeString(o.Category),
			html.EscapeString(o.ItemType),
			html.EscapeString(o.Parameters),
			html.EscapeString(o.Color),
			html.EscapeString(o.Coating),
		)

		// Напоминание за ~24 часа (окно от 23ч до 24ч30м, или если до визита от 2ч15м до 24ч30м и 24ч ещё не отправляли)
		if !o.Reminded24h && until <= 24*time.Hour+30*time.Minute && until > 2*time.Hour+15*time.Minute {
			_ = n.SendReminder(o.ClientTelegramID, o.Date, o.TimeSlot, orderInfo, "через 24 часа")
			if err := n.db.MarkReminderSent(o.ID, true); err != nil {
				log.Printf("Ошибка отметки 24ч напоминания для заказа #%d: %v", o.ID, err)
			}
		}

		// Напоминание за ~2 часа (окно <= 2ч15м)
		if !o.Reminded2h && until <= 2*time.Hour+15*time.Minute {
			_ = n.SendReminder(o.ClientTelegramID, o.Date, o.TimeSlot, orderInfo, "скоро визит")
			if err := n.db.MarkReminderSent(o.ID, false); err != nil {
				log.Printf("Ошибка отметки 2ч напоминания для заказа #%d: %v", o.ID, err)
			}
		}
	}

	return nil
}
