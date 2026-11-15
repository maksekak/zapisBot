package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

// Config — конфигурация бота. Загружается из переменных окружения (.env).
// Все цены и настройки вынесены в конфиг, чтобы заказчик мог менять их без изменения кода.
type Config struct {
	TelegramToken string
	AdminIDs      []int64
	DatabaseURL   string
	BotName       string

	// Калькулятор: базовая стоимость по типу изделия (в рублях)
	BasePrices map[string]float64

	// Надбавка за покрытие (в рублях)
	CoatingSurcharge map[string]float64

	// Надбавка за радиус диска (в рублях)
	RadiusSurcharge map[int]float64

	// Рабочее время
	WorkHourStart int
	WorkHourEnd   int

	// Временные слоты (часы)
	TimeSlots []int

	// Уведомления
	NotificationEnabled bool

	// Прокси для Telegram API (socks5:// или http://)
	TelegramProxy string
}

// Load загружает конфигурацию из переменных окружения.
func Load() *Config {
	cfg := &Config{
		TelegramToken: getEnv("TELEGRAM_TOKEN", ""),
		BotName:       getEnv("BOT_NAME", "Pulver Farbe"),
		DatabaseURL:   getEnv("DATABASE_URL", ""),
		AdminIDs:      parseInt64List(getEnv("ADMIN_IDS", "")),
		BasePrices:    parseFloatMap(getEnv("BASE_PRICES", "")),
		CoatingSurcharge: map[string]float64{
			"matte":   getEnvFloat("COATING_MATTE_SURCHARGE", 0),
			"glossy":  getEnvFloat("COATING_GLOSSY_SURCHARGE", 0),
			"матовый": getEnvFloat("COATING_MATTE_SURCHARGE", 0),
			"глянец":  getEnvFloat("COATING_GLOSSY_SURCHARGE", 0),
		},
		RadiusSurcharge: map[int]float64{
			13: getEnvFloat("RADIUS_13_SURCHARGE", 0),
			14: getEnvFloat("RADIUS_14_SURCHARGE", 0),
			15: getEnvFloat("RADIUS_15_SURCHARGE", 0),
			16: getEnvFloat("RADIUS_16_SURCHARGE", 0),
			17: getEnvFloat("RADIUS_17_SURCHARGE", 0),
			18: getEnvFloat("RADIUS_18_SURCHARGE", 0),
			19: getEnvFloat("RADIUS_19_SURCHARGE", 0),
			20: getEnvFloat("RADIUS_20_SURCHARGE", 0),
			21: getEnvFloat("RADIUS_21_SURCHARGE", 0),
			22: getEnvFloat("RADIUS_22_SURCHARGE", 0),
			23: getEnvFloat("RADIUS_23_SURCHARGE", 0),
			24: getEnvFloat("RADIUS_24_SURCHARGE", 0),
		},
		WorkHourStart:      getEnvInt("WORK_HOUR_START", 9),
		WorkHourEnd:        getEnvInt("WORK_HOUR_END", 18),
		TimeSlots:          parseIntList(getEnv("TIME_SLOTS", "9,10,11,14,15,16,17,18")),
		NotificationEnabled: getEnvBool("NOTIFICATION_ENABLED", true),
	TelegramProxy:       getEnv("TELEGRAM_PROXY", ""),
	}

	if cfg.TelegramToken == "" {
		log.Fatal("TELEGRAM_TOKEN не задан в переменных окружения")
	}
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL не задан в переменных окружения")
	}

	return cfg
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	v := getEnv(key, "")
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func getEnvFloat(key string, def float64) float64 {
	v := getEnv(key, "")
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}

func getEnvBool(key string, def bool) bool {
	v := strings.ToLower(getEnv(key, ""))
	if v == "" {
		return def
	}
	return v == "true" || v == "1" || v == "yes"
}

func parseInt64List(s string) []int64 {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	ids := make([]int64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			log.Printf("Некорректный ADMIN_ID: %s, пропуск", p)
			continue
		}
		ids = append(ids, id)
	}
	return ids
}

func parseIntList(s string) []int {
	if s == "" {
		return []int{9, 10, 11, 14, 15, 16, 17, 18}
	}
	parts := strings.Split(s, ",")
	nums := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		nums = append(nums, n)
	}
	if len(nums) == 0 {
		return []int{9, 10, 11, 14, 15, 16, 17, 18}
	}
	return nums
}

func defaultBasePrices() map[string]float64 {
	return map[string]float64{
		"диски":              1500,
		"детали авто":        2000,
		"мото-рамы":          2500,
		"суппорты":           1800,
		"металлоконструкции": 5000,
	}
}

func parseFloatMap(s string) map[string]float64 {
	result := make(map[string]float64)
	if s == "" {
		return defaultBasePrices()
	}
	// Формат: "type1:price1,type2:price2,..."
	parts := strings.Split(s, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		kv := strings.SplitN(p, ":", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val, err := strconv.ParseFloat(strings.TrimSpace(kv[1]), 64)
		if err != nil {
			continue
		}
		result[key] = val
	}
	if len(result) == 0 {
		return defaultBasePrices()
	}
	return result
}

// IsAdmin проверяет, является ли Telegram ID администратором.
func (c *Config) IsAdmin(telegramID int64) bool {
	for _, id := range c.AdminIDs {
		if id == telegramID {
			return true
		}
	}
	return false
}