package database

import (
	"time"
)

// Client — клиент сервиса записи.
type Client struct {
	ID         int64     `json:"id"`
	FullName   string    `json:"full_name"`
	Phone      string    `json:"phone"`
	TelegramID int64     `json:"telegram_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// Service — услуга (категория + тип изделия + параметры).
type Service struct {
	ID          int64              `json:"id"`
	Category    string             `json:"category"`      // "пескоструй" | "покраска"
	ItemType    string             `json:"item_type"`     // диски | детали авто | мото-рамы | суппорты | металлоконструкции
	Parameters  string             `json:"parameters"`    // JSON строка с параметрами
	Color       string             `json:"color"`         // RAL
	Coating     string             `json:"coating"`       // матовый | глянец
	CreatedAt   time.Time          `json:"created_at"`
}

// PriceList — прайс-лист: тип изделия + параметры + покрытие -> цена.
type PriceList struct {
	ID         int64     `json:"id"`
	ItemType   string    `json:"item_type"`
	Parameters string    `json:"parameters"`
	Coating    string    `json:"coating"`
	Price      float64   `json:"price"`
}

// Schedule — расписание работы.
type Schedule struct {
	ID         int64     `json:"id"`
	Date       string    `json:"date"`          // YYYY-MM-DD
	TimeSlot   string    `json:"time_slot"`     // HH:00
	IsOccupied bool      `json:"is_occupied"`
	IsWorkday  bool      `json:"is_workday"`
}

// Order — запись клиента.
type Order struct {
	ID          int64     `json:"id"`
	ClientID    int64     `json:"client_id"`
	ServiceID   int64     `json:"service_id"`
	Date        string    `json:"date"`
	TimeSlot    string    `json:"time_slot"`
	Status      string    `json:"status"` // confirmed | cancelled | completed
	Price       float64   `json:"price"`
	Reminded24h bool      `json:"reminded_24h"`
	Reminded2h  bool      `json:"reminded_2h"`
	CreatedAt   time.Time `json:"created_at"`
}

// OrderDetail — расширенная информация о заказе (с данными услуги и клиента).
type OrderDetail struct {
	Order
	ClientName       string `json:"client_name"`
	ClientPhone      string `json:"client_phone"`
	ClientTelegramID int64  `json:"client_telegram_id"`
	Category         string `json:"category"`
	ItemType         string `json:"item_type"`
	Parameters       string `json:"parameters"`
	Color            string `json:"color"`
	Coating          string `json:"coating"`
}

// Admin — администратор системы.
type Admin struct {
	TelegramID int64  `json:"telegram_id"`
	FullName   string `json:"full_name"`
	Role       string `json:"role"`
}