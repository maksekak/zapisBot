package services

import (
	"fmt"
	"time"

	"zapisBot/internal/config"
	"zapisBot/internal/database"
)

// OrderService — бизнес-логика работы с заказами.
type OrderService struct {
	cfg *config.Config
	db  *database.DB
}

// NewOrderService создаёт новый сервис заказов.
func NewOrderService(cfg *config.Config, db *database.DB) *OrderService {
	return &OrderService{cfg: cfg, db: db}
}

// CanCancel проверяет, можно ли отменить/перенести запись (не позднее чем за 24 часа).
func (s *OrderService) CanCancel(orderDate, orderTime string) (bool, string) {
	// Парсим дату и время заказа в локальной таймзоне
	layout := "2006-01-02 15:04"
	orderDateTime, err := time.ParseInLocation(layout, orderDate+" "+orderTime, time.Local)
	if err != nil {
		// Пробуем без времени
		orderDateTime, err = time.ParseInLocation("2006-01-02", orderDate, time.Local)
		if err != nil {
			return false, "Некорректная дата заказа"
		}
	}

	now := time.Now()
	diff := orderDateTime.Sub(now)

	// Если до визита меньше 24 часов — отмена/перенос невозможны
	if diff < 24*time.Hour {
		return false, "Отмена или перенос записи возможны не позднее чем за 24 часа до визита"
	}

	return true, ""
}

// CheckPhoneLimit проверяет лимит активных записей на один телефон (макс 2).
func (s *OrderService) CheckPhoneLimit(phone string) (bool, string) {
	count, err := s.db.CountActiveOrdersByPhone(phone)
	if err != nil {
		return false, fmt.Sprintf("Ошибка проверки лимита: %v", err)
	}
	if count >= 2 {
		return false, "У вас уже есть 2 активные записи. Невозможно создать третью. Отмените одну из существующих записей перед новой бронью."
	}
	return true, ""
}

// IsSlotAvailable проверяет доступность слота.
func (s *OrderService) IsSlotAvailable(date, timeSlot string) (bool, error) {
	// Проверяем рабочий день
	isWorkday, err := s.db.IsWorkday(date)
	if err != nil {
		return false, err
	}
	if !isWorkday {
		return false, nil
	}

	// Проверяем occupied
	occupied, err := s.db.IsSlotOccupied(date, timeSlot)
	if err != nil {
		return false, err
	}
	if occupied {
		return false, nil
	}

	return true, nil
}