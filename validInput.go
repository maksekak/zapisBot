package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// IsDateFormat проверяет, что строка соответствует формату ДД.ММ.ГГГГ
func IsDateFormat(s string) bool {
	// Регулярное выражение: 2 цифры, точка, 2 цифры, точка, 4 цифры
	pattern := `^\d{2}\.\d{2}\.\d{2}$`
	matched, _ := regexp.MatchString(pattern, s)
	return matched
}
func HasValidNumber(s string) bool {
	// Регулярное выражение для формата hh:mm
	re := regexp.MustCompile(`^([01]\d|2[0-3]):([0-5]\d)$`)
	return re.MatchString(s)
}
func IsValidName(firstName, lastName string) bool {
	// Проверка на пустоту
	if firstName == "" || lastName == "" {
		return false
	}

	// Минимальная длина
	if len(firstName) < 2 || len(lastName) < 2 {
		return false
	}

	// Проверка символов (только буквы и пробелы)
	for _, r := range firstName {
		if !unicode.IsLetter(r) && r != ' ' {
			return false
		}
	}
	for _, r := range lastName {
		if !unicode.IsLetter(r) && r != ' ' {
			return false
		}
	}

	// Проверка пробелов
	trimmedFirst := strings.TrimSpace(firstName)
	trimmedLast := strings.TrimSpace(lastName)
	if trimmedFirst != firstName || trimmedLast != lastName {
		return false // пробелы в начале/конце
	}
	if strings.Contains(firstName, "  ") || strings.Contains(lastName, "  ") {
		return false // несколько пробелов подряд
	}

	return true
}

var phoneRegex = regexp.MustCompile(
	`^(\+7|8)(\s*\(\d{3}\)\s*|\s*\d{3}\s*)?[\d\s\-]{7,15}$`,
)

func IsValidPhoneRegex(phone string) bool {
	return phoneRegex.MatchString(phone)
}
func validErr(chatID int64, currentData []string) {
	mu.Lock()
	delete(inputData, chatID)
	currentData = nil
	mu.Unlock()
	sendMenu(chatID)
	readyToRec[chatID] = false
}

// ParseHourFromTime преобразует время в число часов (9 для "9:00", "09:00", "9", "09")
func ParseHourFromTime(timeStr string) (int, error) {
	if timeStr == "" {
		return 0, fmt.Errorf("пустая строка")
	}

	timeStr = strings.TrimSpace(timeStr)

	// Случай: только часы (9 или 09)
	if !strings.Contains(timeStr, ":") {
		hour, err := strconv.Atoi(timeStr)
		if err != nil {
			return 0, fmt.Errorf("некорректное число часов: %s", timeStr)
		}
		if hour < 0 || hour > 23 {
			return 0, fmt.Errorf("часы вне диапазона 0–23: %d", hour)
		}
		return hour, nil
	}

	// Случай: hh:mm или h:mm
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("неверный формат времени: %s", timeStr)
	}

	hourStr, minStr := parts[0], parts[1]

	// Проверяем минуты
	minutes, err := strconv.Atoi(minStr)
	if err != nil || minutes < 0 || minutes > 59 {
		return 0, fmt.Errorf("некорректные минуты: %s", minStr)
	}

	// Если минуты не 00 — ошибка (по условию хотим только полные часы)
	if minutes != 0 {
		return 0, fmt.Errorf("минуты должны быть 00: %s", timeStr)
	}

	// Преобразуем часы
	hour, err := strconv.Atoi(hourStr)
	if err != nil || hour < 0 || hour > 23 {
		return 0, fmt.Errorf("некорректные часы: %s", hourStr)
	}

	return hour, nil
}
