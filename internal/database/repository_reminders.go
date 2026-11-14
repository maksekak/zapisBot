package database

import (
	"database/sql"
	"time"
)

func scanOrderDetails(rows *sql.Rows, withReminders bool) ([]OrderDetail, error) {
	var list []OrderDetail
	for rows.Next() {
		var od OrderDetail
		var dateVal, slotVal time.Time
		var svcID sql.NullInt64
		var priceVal sql.NullFloat64

		var err error
		if withReminders {
			err = rows.Scan(
				&od.ID, &od.ClientID, &svcID, &dateVal, &slotVal, &od.Status, &priceVal,
				&od.Reminded24h, &od.Reminded2h, &od.CreatedAt,
				&od.ClientName, &od.ClientPhone, &od.ClientTelegramID,
				&od.Category, &od.ItemType, &od.Parameters, &od.Color, &od.Coating,
			)
		} else {
			err = rows.Scan(
				&od.ID, &od.ClientID, &svcID, &dateVal, &slotVal, &od.Status, &priceVal, &od.CreatedAt,
				&od.ClientName, &od.ClientPhone, &od.ClientTelegramID,
				&od.Category, &od.ItemType, &od.Parameters, &od.Color, &od.Coating,
			)
		}
		if err != nil {
			return nil, err
		}
		if svcID.Valid {
			od.ServiceID = svcID.Int64
		}
		if priceVal.Valid {
			od.Price = priceVal.Float64
		}
		od.Date = dateVal.Format("2006-01-02")
		od.TimeSlot = slotVal.Format("15:04")
		list = append(list, od)
	}
	return list, rows.Err()
}

// GetOrderDetailsByClient возвращает заказы клиента с деталями услуги.
func (d *DB) GetOrderDetailsByClient(clientID int64) ([]OrderDetail, error) {
	rows, err := d.Query(
		`SELECT o.id, o.client_id, o.service_id, o.date, o.time_slot, o.status, o.price, o.created_at,
		        COALESCE(c.full_name, ''), COALESCE(c.phone, ''), COALESCE(c.telegram_id, 0),
		        COALESCE(s.category, ''), COALESCE(s.item_type, ''),
		        COALESCE(s.parameters->>'parameters', ''), COALESCE(s.color, ''), COALESCE(s.coating, '')
		 FROM orders o
		 LEFT JOIN clients c ON o.client_id = c.id
		 LEFT JOIN services s ON o.service_id = s.id
		 WHERE o.client_id = $1
		 ORDER BY o.date DESC, o.time_slot DESC, o.created_at DESC`,
		clientID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOrderDetails(rows, false)
}

// GetOrderDetail возвращает детальную информацию по одному заказу.
func (d *DB) GetOrderDetail(orderID int64) (*OrderDetail, error) {
	rows, err := d.Query(
		`SELECT o.id, o.client_id, o.service_id, o.date, o.time_slot, o.status, o.price, o.created_at,
		        COALESCE(c.full_name, ''), COALESCE(c.phone, ''), COALESCE(c.telegram_id, 0),
		        COALESCE(s.category, ''), COALESCE(s.item_type, ''),
		        COALESCE(s.parameters->>'parameters', ''), COALESCE(s.color, ''), COALESCE(s.coating, '')
		 FROM orders o
		 LEFT JOIN clients c ON o.client_id = c.id
		 LEFT JOIN services s ON o.service_id = s.id
		 WHERE o.id = $1`,
		orderID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list, err := scanOrderDetails(rows, false)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return &list[0], nil
}

// GetOrdersForDate возвращает все подтверждённые заказы на дату.
func (d *DB) GetOrdersForDate(date string) ([]Order, error) {
	rows, err := d.Query(
		`SELECT o.id, o.client_id, o.service_id, o.date, o.time_slot, o.status, o.price, o.created_at
		 FROM orders o
		 WHERE o.date = $1 AND o.status = 'confirmed'
		 ORDER BY o.time_slot`,
		date,
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

// GetOrdersDetailForDate возвращает все подтверждённые заказы на дату с данными клиента и услуги.
func (d *DB) GetOrdersDetailForDate(date string) ([]OrderDetail, error) {
	rows, err := d.Query(
		`SELECT o.id, o.client_id, o.service_id, o.date, o.time_slot, o.status, o.price, o.created_at,
		        COALESCE(c.full_name, ''), COALESCE(c.phone, ''), COALESCE(c.telegram_id, 0),
		        COALESCE(s.category, ''), COALESCE(s.item_type, ''),
		        COALESCE(s.parameters->>'parameters', ''), COALESCE(s.color, ''), COALESCE(s.coating, '')
		 FROM orders o
		 LEFT JOIN clients c ON o.client_id = c.id
		 LEFT JOIN services s ON o.service_id = s.id
		 WHERE o.date = $1 AND o.status = 'confirmed'
		 ORDER BY o.time_slot`,
		date,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOrderDetails(rows, false)
}

// GetPendingReminders возвращает ближайшие подтверждённые заказы, по которым ещё не отправлены все напоминания.
func (d *DB) GetPendingReminders(fromDate, toDate string) ([]OrderDetail, error) {
	rows, err := d.Query(
		`SELECT o.id, o.client_id, o.service_id, o.date, o.time_slot, o.status, o.price,
		        o.reminded_24h, o.reminded_2h, o.created_at,
		        COALESCE(c.full_name, ''), COALESCE(c.phone, ''), COALESCE(c.telegram_id, 0),
		        COALESCE(s.category, ''), COALESCE(s.item_type, ''),
		        COALESCE(s.parameters->>'parameters', ''), COALESCE(s.color, ''), COALESCE(s.coating, '')
		 FROM orders o
		 LEFT JOIN clients c ON o.client_id = c.id
		 LEFT JOIN services s ON o.service_id = s.id
		 WHERE o.status = 'confirmed'
		   AND o.date >= $1::date AND o.date <= $2::date
		   AND (o.reminded_24h = FALSE OR o.reminded_2h = FALSE)
		 ORDER BY o.date, o.time_slot`,
		fromDate, toDate,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOrderDetails(rows, true)
}

// MarkReminderSent отмечает отправленное напоминание (is24h == true -> reminded_24h, иначе reminded_2h).
func (d *DB) MarkReminderSent(orderID int64, is24h bool) error {
	if is24h {
		_, err := d.Exec("UPDATE orders SET reminded_24h = TRUE WHERE id = $1", orderID)
		return err
	}
	_, err := d.Exec("UPDATE orders SET reminded_2h = TRUE WHERE id = $1", orderID)
	return err
}
