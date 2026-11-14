package services

import (
	"fmt"
	"html"
	"log"

	"zapisBot/internal/config"
	"zapisBot/internal/database"
)

// Calculator — сервис расчёта стоимости.
// Формула: база по типу изделия + надбавка за параметры + надбавка за покрытие.
// Все значения конфигурируются через .env (не зашиты в код).
type Calculator struct {
	cfg *config.Config
	db  *database.DB
}

// NewCalculator создаёт новый калькулятор.
func NewCalculator(cfg *config.Config, db *database.DB) *Calculator {
	return &Calculator{cfg: cfg, db: db}
}

// CalcResult — результат расчёта стоимости.
type CalcResult struct {
	BasePrice          float64 `json:"base_price"`
	ParameterSurcharge float64 `json:"parameter_surcharge"`
	CoatingSurcharge   float64 `json:"coating_surcharge"`
	Total              float64 `json:"total"`
	HasPrice           bool    `json:"has_price"`
	FromPriceList      bool    `json:"from_price_list"`
	Message            string  `json:"message"`
}

// Calculate вычисляет предварительную стоимость заказа.
func (c *Calculator) Calculate(itemType, parameters, coating string) CalcResult {
	// Базовая цена из конфига
	basePrice, hasBase := c.cfg.BasePrices[itemType]

	// Надбавка за покрытие
	coatingSurcharge := c.cfg.CoatingSurcharge[coating]

	// Надбавка за параметры (для дисков — по радиусу, для прочих — фиксированная)
	parameterSurcharge := c.parameterSurcharge(itemType, parameters)

	// Пытаемся найти точную цену в прайс-листе БД
	var dbPrice float64
	var hasDBPrice bool
	if c.db != nil {
		var err error
		dbPrice, hasDBPrice, err = c.db.GetPrice(itemType, parameters, coating)
		if err != nil {
			log.Printf("Ошибка чтения прайс-листа БД (%s, %s, %s): %v", itemType, parameters, coating, err)
		}
	}

	result := CalcResult{
		BasePrice:          basePrice,
		ParameterSurcharge: parameterSurcharge,
		CoatingSurcharge:   coatingSurcharge,
		Total:              basePrice + parameterSurcharge + coatingSurcharge,
		HasPrice:           hasBase || hasDBPrice,
	}

	if hasDBPrice {
		result.Total = dbPrice
		result.HasPrice = true
		result.FromPriceList = true
	}

	if !result.HasPrice {
		result.Message = "Стоимость уточняется у менеджера"
	} else {
		result.Message = fmt.Sprintf("Предварительная стоимость: %.0f ₽", result.Total)
	}

	return result
}

// parameterSurcharge возвращает надбавку за параметры.
func (c *Calculator) parameterSurcharge(itemType, parameters string) float64 {
	// Для дисков параметры — это радиус (R13-R24), берём надбавку по радиусу
	if itemType == "диски" {
		for r := 13; r <= 24; r++ {
			radiusKey := fmt.Sprintf("R%d", r)
			if parameters == radiusKey || parameters == fmt.Sprintf("%d", r) {
				return c.cfg.RadiusSurcharge[r]
			}
		}
	}
	// Для остальных типов изделий — фиксированная надбавка (например, за габариты)
	return 0
}

// FormatResult форматирует результат расчёта для пользователя.
func (c *Calculator) FormatResult(result CalcResult, category, itemType, parameters, coating, color string) string {
	esc := html.EscapeString
	if !result.HasPrice {
		return fmt.Sprintf(
			"💰 <b>Калькулятор стоимости</b>\n\n"+
				"Категория: %s\nТип изделия: %s\nПараметры: %s\nЦвет: %s\nПокрытие: %s\n\n"+
				"⚠️ %s",
			esc(category), esc(itemType), esc(parameters), esc(color), esc(coating), esc(result.Message),
		)
	}
	if result.FromPriceList {
		return fmt.Sprintf(
			"💰 <b>Калькулятор стоимости</b>\n\n"+
				"Категория: %s\nТип изделия: %s\nПараметры: %s\nЦвет: %s\nПокрытие: %s\n\n"+
				"Тариф по прайс-листу: <b>%.0f ₽</b>",
			esc(category), esc(itemType), esc(parameters), esc(color), esc(coating), result.Total,
		)
	}
	return fmt.Sprintf(
		"💰 <b>Калькулятор стоимости</b>\n\n"+
			"Категория: %s\nТип изделия: %s\nПараметры: %s\nЦвет: %s\nПокрытие: %s\n\n"+
			"База: %.0f ₽\nНадбавка за параметры: %.0f ₽\nНадбавка за покрытие: %.0f ₽\n"+
			"<b>Итого: %.0f ₽</b>",
		esc(category), esc(itemType), esc(parameters), esc(color), esc(coating),
		result.BasePrice, result.ParameterSurcharge, result.CoatingSurcharge, result.Total,
	)
}
