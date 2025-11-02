package main

import (
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"
)

func waitUntilNextDay() time.Duration {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	return next.Sub(now)
}
func runDailyUpdater(f *excelize.File) {
	for {
		sleepTime := waitUntilNextDay()
		fmt.Printf("Ожидание до следующих суток: %v\n", sleepTime)
		date := tomorrowDate(-1)
		for key, user := range userStorage {
			if user.userDate == date {
				cancelRec(f, key, userStorage)
			}
		}
		time.Sleep(sleepTime)
		delPast(f)
		addFut(f)

	}
}
func delPast(f *excelize.File) {
	mu.Lock()
	defer mu.Unlock()
	for i := 5; i >= 0; i-- {
		f.RemoveRow("Sheet1", i)

	}
}
func addFut(f *excelize.File) {
	mu.Lock()
	defer mu.Unlock()
	line, err := f.GetRows("Sheet1")
	if err != nil {
		fmt.Println(err)
	}
	lenOfTable := len(line) - 5
	fmt.Println(line[lenOfTable][0])
	l := line[lenOfTable][0]
	dateToAdd, err := getTomorrowShortDate(l)
	if err != nil {
		fmt.Println(err)
	}
	time := []string{"9", "10", "11", "14", "15", "16", "17", "18"}
	kostil := []string{"|", "|", "|", "|", "|"}
	rowToAdd := append([]string{dateToAdd}, time...)
	cords, _ := excelize.CoordinatesToCellName(1, lenOfTable+6)
	cords2, _ := excelize.CoordinatesToCellName(10, lenOfTable+6)
	f.SetSheetRow("Sheet1", cords, &rowToAdd)
	f.SetSheetCol("Sheet1", cords2, &kostil)
	f.Save()
}
func getTomorrowShortDate(dateStr string) (string, error) {
	// Шаг 1. Парсим входную строку в time.Time
	// Шаблон "02.01.06" соответствует формату "ДД.ММ.ГГ"
	parsedTime, err := time.Parse("02.01.06", dateStr)
	if err != nil {
		return "", fmt.Errorf("неверный формат даты '%s': %w", dateStr, err)
	}
	// Шаг 2. Добавляем 24 часа (1 день)
	tomorrow := parsedTime.Add(24 * time.Hour)
	// Шаг 3. Форматируем обратно в строку в формате "01.01.25"
	return tomorrow.Format("02.01.06"), nil
}
