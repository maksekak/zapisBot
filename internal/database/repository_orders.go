package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// CreateOrder создаёт заказ в транзакции: бронирует слот, создаёт услугу и заказ.
func (d *DB) CreateOrder(clientID int64, category, itemType, parameters, color, coating, date, timeSlot string, price float64) (*Order, error) {
	tx, err := d.Begin()
	if err != nil {
		return nil, fmt.Errorf("начало транзакции: %w", err)
	}
	defer tx.Rollback()

	var occupied, isWorkday bool
	err = tx.QueryRow(
		"SELECT is_occupied, is_workday FROM schedule WHERE date = $1 AND time_slot = $2 FOR UPDATE",
		date, timeSlot,
	).Scan(&occupied, &isWorkday)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("слот на %s %s не найден", date, timeSlot)
		}
		return nil, fmt.Errorf("проверка слота: %w", err)
	}
	if !isWorkday {
		return nil, fmt.Errorf("выбранный день является выходным")
	}
	if occupied {
		return nil, fmt.Errorf("слот уже занят")
	}

	if _, err = tx.Exec(
		"UPDATE schedule SET is_occupied = TRUE WHERE date = $1 AND time_slot = $2",
		date, timeSlot,
	); err != nil {
		return nil, fmt.Errorf("бронирование слота: %w", err)
	}

	paramsBytes, err := json.Marshal(map[string]string{
		"parameters": parameters,
		"color":      color,
	})
	if err != nil {
		return nil, fmt.Errorf("сериализация параметров: %w", err)
	}

	var serviceID int64
	err = tx.QueryRow(
		`INSERT INTO services (category, item_type, parameters, color, coating)
		 VALUES ($1, $2, $3::jsonb, $4, $5)
		 RETURNING id`,
		category, itemType, string(paramsBytes), color, coating,
	).Scan(&serviceID)
	if err != nil {
		return nil, fmt.Errorf("создание услуги: %w", err)
	}

	var order Order
	var dateVal, slotVal time.Time
	var svcID sql.NullInt64
	var priceVal sql.NullFloat64
	err = tx.QueryRow(
		`INSERT INTO orders (client_id, service_id, date, time_slot, status, price)
		 VALUES ($1, $2, $3, $4, 'confirmed', $5)
		 RETURNING id, client_id, service_id, date, time_slot, status, price, created_at`,
		clientID, serviceID, date, timeSlot, price,
	).Scan(&order.ID, &order.ClientID, &svcID, &dateVal, &slotVal, &order.Status, &priceVal, &order.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("создание заказа: %w", err)
	}
	if svcID.Valid {
		order.ServiceID = svcID.Int64
	}
	if priceVal.Valid {
		order.Price = priceVal.Float64
	}
	order.Date = dateVal.Format("2006-01-02")
	order.TimeSlot = slotVal.Format("15:04")

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("коммит заказа: %w", err)
	}

	log.Printf("Создан заказ #%d для клиента %d на %s %s", order.ID, clientID, order.Date, order.TimeSlot)
	return &order, nil
}

// GetOrdersByClient возвращает заказы клиента (активные и архивные).
func (d *DB) GetOrdersByClient(clientID int64) ([]Order, error) {
	rows, err := d.Query(
		`SELECT id, client_id, service_id, date, time_slot, status, price, created_at
		 FROM orders WHERE client_id = $1
		 ORDER BY date DESC, time_slot DESC, created_at DESC`,
		clientID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		var dateVal, slotVal time.Time
		var svcID sql.NullInt64
		var priceVal sql.NullFloat64
		if err := rows.Scan(&o.ID, &o.ClientID, &svcID, &dateVal, &slotVal, &o.Status, &priceVal, &o.CreatedAt); err != nil {
			return nil, err
		}
		if svcID.Valid {
			o.ServiceID = svcID.Int64
		}
		if priceVal.Valid {
			o.Price = priceVal.Float64
		}
		o.Date = dateVal.Format("2006-01-02")
		o.TimeSlot = slotVal.Format("15:04")
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

// CancelOrder отменяет заказ (меняет статус на cancelled и освобождает слот).
func (d *DB) CancelOrder(orderID int64) error {
	tx, err := d.Begin()
	if err != nil {
		return fmt.Errorf("начало транзакции: %w", err)
	}
	defer tx.Rollback()

	var dateVal, slotVal time.Time
	err = tx.QueryRow(
		"SELECT date, time_slot FROM orders WHERE id = $1 AND status = 'confirmed' FOR UPDATE",
		orderID,
	).Scan(&dateVal, &slotVal)
	if err != nil {
		return fmt.Errorf("заказ не найден или уже отменён: %w", err)
	}
	dateStr := dateVal.Format("2006-01-02")
	slotStr := slotVal.Format("15:04")

	if _, err = tx.Exec("UPDATE orders SET status = 'cancelled' WHERE id = $1", orderID); err != nil {
		return fmt.Errorf("отмена заказа: %w", err)
	}
	if _, err = tx.Exec("UPDATE schedule SET is_occupied = FALSE WHERE date = $1 AND time_slot = $2", dateStr, slotStr); err != nil {
		return fmt.Errorf("освобождение слота: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("коммит отмены: %w", err)
	}

	log.Printf("Заказ #%d отменён, слот %s %s освобождён", orderID, dateStr, slotStr)
	return nil
}

// RescheduleOrder переносит подтверждённый заказ на новую дату и время в одной транзакции.
func (d *DB) RescheduleOrder(orderID int64, newDate, newTimeSlot string) (*Order, error) {
	tx, err := d.Begin()
	if err != nil {
		return nil, fmt.Errorf("начало транзакции: %w", err)
	}
	defer tx.Rollback()

	var oldDateVal, oldSlotVal time.Time
	err = tx.QueryRow(
		"SELECT date, time_slot FROM orders WHERE id = $1 AND status = 'confirmed' FOR UPDATE",
		orderID,
	).Scan(&oldDateVal, &oldSlotVal)
	if err != nil {
		return nil, fmt.Errorf("заказ не найден или уже отменён: %w", err)
	}
	oldDate := oldDateVal.Format("2006-01-02")
	oldSlot := oldSlotVal.Format("15:04")
	if oldDate == newDate && oldSlot == newTimeSlot {
		return nil, fmt.Errorf("новая дата и время совпадают с текущими")
	}

	var occupied, isWorkday bool
	err = tx.QueryRow(
		"SELECT is_occupied, is_workday FROM schedule WHERE date = $1 AND time_slot = $2 FOR UPDATE",
		newDate, newTimeSlot,
	).Scan(&occupied, &isWorkday)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("выбранный слот не найден")
		}
		return nil, fmt.Errorf("проверка нового слота: %w", err)
	}
	if !isWorkday {
		return nil, fmt.Errorf("выбранный день является выходным")
	}
	if occupied {
		return nil, fmt.Errorf("выбранный слот уже занят")
	}

	if _, err = tx.Exec("UPDATE schedule SET is_occupied = FALSE WHERE date = $1 AND time_slot = $2", oldDate, oldSlot); err != nil {
		return nil, fmt.Errorf("освобождение старого слота: %w", err)
	}
	if _, err = tx.Exec("UPDATE schedule SET is_occupied = TRUE WHERE date = $1 AND time_slot = $2", newDate, newTimeSlot); err != nil {
		return nil, fmt.Errorf("бронирование нового слота: %w", err)
	}

	var order Order
	var dateVal, slotVal time.Time
	var svcID sql.NullInt64
	var priceVal sql.NullFloat64
	err = tx.QueryRow(
		`UPDATE orders
		 SET date = $1, time_slot = $2, reminded_24h = FALSE, reminded_2h = FALSE
		 WHERE id = $3
		 RETURNING id, client_id, service_id, date, time_slot, status, price, created_at`,
		newDate, newTimeSlot, orderID,
	).Scan(&order.ID, &order.ClientID, &svcID, &dateVal, &slotVal, &order.Status, &priceVal, &order.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("обновление заказа: %w", err)
	}
	if svcID.Valid {
		order.ServiceID = svcID.Int64
	}
	if priceVal.Valid {
		order.Price = priceVal.Float64
	}
	order.Date = dateVal.Format("2006-01-02")
	order.TimeSlot = slotVal.Format("15:04")

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("коммит переноса: %w", err)
	}

	log.Printf("Заказ #%d перенесён с %s %s на %s %s", orderID, oldDate, oldSlot, order.Date, order.TimeSlot)
	return &order, nil
}
