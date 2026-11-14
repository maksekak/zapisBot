package database

import (
	"database/sql"
	"fmt"
	"time"
)

// ClientByPhone ищет клиента по телефону.
func (d *DB) ClientByPhone(phone string) (*Client, error) {
	var c Client
	var telegramID sql.NullInt64
	err := d.QueryRow(
		"SELECT id, full_name, phone, telegram_id, created_at FROM clients WHERE phone = $1",
		phone,
	).Scan(&c.ID, &c.FullName, &c.Phone, &telegramID, &c.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if telegramID.Valid {
		c.TelegramID = telegramID.Int64
	}
	return &c, nil
}

// ClientByTelegramID ищет клиента по Telegram ID.
func (d *DB) ClientByTelegramID(telegramID int64) (*Client, error) {
	var c Client
	var tgID sql.NullInt64
	err := d.QueryRow(
		"SELECT id, full_name, phone, telegram_id, created_at FROM clients WHERE telegram_id = $1",
		telegramID,
	).Scan(&c.ID, &c.FullName, &c.Phone, &tgID, &c.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if tgID.Valid {
		c.TelegramID = tgID.Int64
	}
	return &c, nil
}

// CreateClient создаёт нового клиента.
func (d *DB) CreateClient(fullName, phone string, telegramID int64) (*Client, error) {
	var c Client
	var tgID sql.NullInt64
	err := d.QueryRow(
		`INSERT INTO clients (full_name, phone, telegram_id)
		 VALUES ($1, $2, $3)
		 RETURNING id, full_name, phone, telegram_id, created_at`,
		fullName, phone, telegramID,
	).Scan(&c.ID, &c.FullName, &c.Phone, &tgID, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("создание клиента: %w", err)
	}
	if tgID.Valid {
		c.TelegramID = tgID.Int64
	}
	return &c, nil
}

// UpsertClient создаёт клиента или обновляет существующего (по phone или telegram_id).
func (d *DB) UpsertClient(fullName, phone string, telegramID int64) (*Client, error) {
	existingByPhone, err := d.ClientByPhone(phone)
	if err != nil {
		return nil, err
	}
	if existingByPhone != nil {
		var c Client
		var tgID sql.NullInt64
		err := d.QueryRow(
			`UPDATE clients SET full_name = $1, telegram_id = $2
			 WHERE id = $3
			 RETURNING id, full_name, phone, telegram_id, created_at`,
			fullName, telegramID, existingByPhone.ID,
		).Scan(&c.ID, &c.FullName, &c.Phone, &tgID, &c.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("обновление клиента по телефону: %w", err)
		}
		if tgID.Valid {
			c.TelegramID = tgID.Int64
		}
		return &c, nil
	}

	existingByTG, err := d.ClientByTelegramID(telegramID)
	if err != nil {
		return nil, err
	}
	if existingByTG != nil {
		var c Client
		var tgID sql.NullInt64
		err := d.QueryRow(
			`UPDATE clients SET full_name = $1, phone = $2
			 WHERE id = $3
			 RETURNING id, full_name, phone, telegram_id, created_at`,
			fullName, phone, existingByTG.ID,
		).Scan(&c.ID, &c.FullName, &c.Phone, &tgID, &c.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("обновление клиента по telegram_id: %w", err)
		}
		if tgID.Valid {
			c.TelegramID = tgID.Int64
		}
		return &c, nil
	}

	return d.CreateClient(fullName, phone, telegramID)
}

// UpdateClientName обновляет имя клиента.
func (d *DB) UpdateClientName(clientID int64, fullName string) error {
	_, err := d.Exec("UPDATE clients SET full_name = $1 WHERE id = $2", fullName, clientID)
	return err
}

// CountActiveOrdersByPhone возвращает количество активных заказов клиента по телефону.
func (d *DB) CountActiveOrdersByPhone(phone string) (int, error) {
	var count int
	err := d.QueryRow(
		`SELECT COUNT(*) FROM orders o
		 JOIN clients c ON o.client_id = c.id
		 WHERE c.phone = $1 AND o.status = 'confirmed'`,
		phone,
	).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// IsSlotOccupied проверяет, занят ли слот.
func (d *DB) IsSlotOccupied(date, timeSlot string) (bool, error) {
	var occupied bool
	err := d.QueryRow(
		"SELECT is_occupied FROM schedule WHERE date = $1 AND time_slot = $2",
		date, timeSlot,
	).Scan(&occupied)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return occupied, nil
}

// IsWorkday проверяет, является ли дата рабочим днём.
func (d *DB) IsWorkday(date string) (bool, error) {
	var isWorkday bool
	err := d.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM schedule WHERE date = $1 AND is_workday = TRUE)",
		date,
	).Scan(&isWorkday)
	if err != nil {
		return false, err
	}
	return isWorkday, nil
}

// GetFreeSlots возвращает список свободных слотов на дату в формате HH:MM.
func (d *DB) GetFreeSlots(date string) ([]string, error) {
	rows, err := d.Query(
		`SELECT time_slot FROM schedule
		 WHERE date = $1 AND is_workday = TRUE AND is_occupied = FALSE
		 ORDER BY time_slot`,
		date,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slots []string
	for rows.Next() {
		var t time.Time
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		slots = append(slots, t.Format("15:04"))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return slots, nil
}

// GetAvailableDates возвращает даты на ближайшие days дней, где есть хотя бы один свободный рабочий слот.
func (d *DB) GetAvailableDates(startDate string, days int) (map[string]int, error) {
	rows, err := d.Query(
		`SELECT date, COUNT(*) FROM schedule
		 WHERE date >= $1::date AND date < ($1::date + $2 * INTERVAL '1 day')
		   AND is_workday = TRUE AND is_occupied = FALSE
		 GROUP BY date
		 ORDER BY date`,
		startDate, days,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var dt time.Time
		var cnt int
		if err := rows.Scan(&dt, &cnt); err != nil {
			return nil, err
		}
		result[dt.Format("2006-01-02")] = cnt
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// GetPrice возвращает цену из прайс-листа по комбинации параметров.
func (d *DB) GetPrice(itemType, parameters, coating string) (float64, bool, error) {
	var price float64
	err := d.QueryRow(
		`SELECT price FROM price_list
		 WHERE item_type = $1 AND parameters = $2 AND coating = $3`,
		itemType, parameters, coating,
	).Scan(&price)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, false, nil // цены нет
		}
		return 0, false, err
	}
	return price, true, nil
}

// GetScheduleGrid возвращает сетку расписания на дату.
func (d *DB) GetScheduleGrid(date string) ([]Schedule, error) {
	rows, err := d.Query(
		`SELECT date, time_slot, is_occupied, is_workday FROM schedule
		 WHERE date = $1 ORDER BY time_slot`,
		date,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedule []Schedule
	for rows.Next() {
		var s Schedule
		var dateTime time.Time
		var timeSlot time.Time
		if err := rows.Scan(&dateTime, &timeSlot, &s.IsOccupied, &s.IsWorkday); err != nil {
			return nil, err
		}
		s.Date = dateTime.Format("2006-01-02")
		s.TimeSlot = timeSlot.Format("15:04")
		schedule = append(schedule, s)
	}
	return schedule, nil
}

// UpsertPriceList добавляет или обновляет запись в прайс-листе.
func (d *DB) UpsertPriceList(itemType, parameters, coating string, price float64) error {
	_, err := d.Exec(
		`INSERT INTO price_list (item_type, parameters, coating, price)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (item_type, parameters, coating)
		 DO UPDATE SET price = EXCLUDED.price`,
		itemType, parameters, coating, price,
	)
	return err
}

// GetAllPrices возвращает все записи прайс-листа.
func (d *DB) GetAllPrices() ([]PriceList, error) {
	rows, err := d.Query("SELECT id, item_type, parameters, coating, price FROM price_list ORDER BY item_type, parameters, coating")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prices []PriceList
	for rows.Next() {
		var p PriceList
		var params, coating sql.NullString
		if err := rows.Scan(&p.ID, &p.ItemType, &params, &coating, &p.Price); err != nil {
			return nil, err
		}
		p.Parameters = params.String
		p.Coating = coating.String
		prices = append(prices, p)
	}
	return prices, rows.Err()
}

// DeletePriceList удаляет запись из прайс-листа.
func (d *DB) DeletePriceList(id int64) error {
	_, err := d.Exec("DELETE FROM price_list WHERE id = $1", id)
	return err
}

// SetWorkday устанавливает флаг рабочего/выходного дня для всех слотов указанной даты.
func (d *DB) SetWorkday(date string, isWorkday bool) error {
	return d.SetWorkdayWithSlots(date, isWorkday, []int{9, 10, 11, 14, 15, 16, 17, 18})
}

// SetWorkdayWithSlots обновляет статус рабочего дня для существующих слотов даты или создаёт их при отсутствии.
func (d *DB) SetWorkdayWithSlots(date string, isWorkday bool, timeSlots []int) error {
	res, err := d.Exec("UPDATE schedule SET is_workday = $2 WHERE date = $1", date, isWorkday)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err == nil && affected > 0 {
		return nil
	}
	for _, hour := range timeSlots {
		slot := fmt.Sprintf("%02d:00", hour)
		if err := d.SetWorkdaySlot(date, slot, false, isWorkday); err != nil {
			return err
		}
	}
	return nil
}

// SetWorkdaySlot создаёт или обновляет конкретный временной слот для даты.
func (d *DB) SetWorkdaySlot(date, timeSlot string, isOccupied, isWorkday bool) error {
	_, err := d.Exec(
		`INSERT INTO schedule (date, time_slot, is_occupied, is_workday)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (date, time_slot) DO UPDATE SET is_occupied = EXCLUDED.is_occupied, is_workday = EXCLUDED.is_workday`,
		date, timeSlot, isOccupied, isWorkday,
	)
	return err
}