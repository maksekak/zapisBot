package main

import (
	"fmt"
	"strings"

	//"strconv"

	"time"

	"github.com/xuri/excelize/v2"
)

func tomorrowDate(p int) string {
	now := time.Now()
	tomorrow := now.AddDate(0, 0, p)

	day := tomorrow.Day()
	month := int(tomorrow.Month())
	year := tomorrow.Year()

	// Берём последние 2 цифры года
	shortYear := year % 100

	// Форматируем с ведущими нулями для дня и месяца
	return fmt.Sprintf("%02d.%02d.%02d", day, month, shortYear)
}

func freeDays(f *excelize.File, dayChange int) map[string][]string {
	date := tomorrowDate(dayChange)

	start := false
	freeTime := make([]string, 0)
	mapOfDates := make(map[string][]string)
	line, err := f.GetRows("Sheet1")
	if err != nil {
		fmt.Println("ошибка чтения строки", err)
	}

	for i, r := range line {
		if strings.Contains(r[0], date) {
			start = true
		}
		if start {
			for j := range 8 {
				if line[i+1][j+1] == "" {
					freeTime = append(freeTime, line[i][j+1])
				}
			}
			mapOfDates[r[0]] = freeTime
			freeTime = nil
			if i == 150 {
				break
			}
		}
	}

	return mapOfDates
}

func nearDate(f *excelize.File, dayChange int) (s string, sm []string) {
	date := tomorrowDate(dayChange)
	for key, vel := range freeDays(f, dayChange) {
		if key == date {
			s, sm = key, vel
		}
	}
	return
}
func reFind(f *excelize.File, userStorage map[int64]userStatus, id int64) bool {
	mu.Lock()
	defer mu.Unlock()

	line, err := f.GetRows("Sheet1")
	if err != nil {
		fmt.Printf("Ошибка чтения строки: %v\n", err)

	}
	userStruct, exists := userStorage[id]
	if !exists {
		fmt.Printf("Пользователь с id=%d не найден\n", id)

	}

	dataUs := []string{
		userStruct.userDate,
		userStruct.userTime,
	}
	for i, r := range line {
		if strings.Contains(r[0], dataUs[0]) {
			for j := range 8 {
				if strings.Contains(line[i][j+1], dataUs[1]) {
					if line[i+1][j+1] == "" {
						return true
					} else {
						return false
					}
				}
			}
		}
	}

	return false
}
func newName(f *excelize.File, userStorage map[int64]userStatus, id int64) {
	mu.Lock()
	defer mu.Unlock()

	line, err := f.GetRows("Sheet1")
	if err != nil {
		fmt.Printf("Ошибка чтения строки: %v\n", err)
		return
	}

	userStruct, exists := userStorage[id]
	if !exists {
		fmt.Printf("Пользователь с id=%d не найден\n", id)
		return
	}

	dataUs := []string{
		userStruct.userDate,
		userStruct.userTime,
		userStruct.userName,
		userStruct.userSurname,
		userStruct.userPhone,
		userStruct.userOrder,
	}

	for i, r := range line {
		if strings.Contains(r[0], dataUs[0]) {
			for j := range 8 {
				if strings.Contains(line[i][j+1], dataUs[1]) {
					for k := range 4 {
						col, row := j, i
						cellRef, err := excelize.CoordinatesToCellName(col+2, row+2+k)
						if err != nil {
							fmt.Printf("Не удалось преобразовать координаты: %v\n", err)
							return
						}
						f.SetCellValue("Sheet1", cellRef, dataUs[2+k])
					}
				}
			}
		}
	}

	if err := f.Save(); err != nil {
		fmt.Printf("Ошибка сохранения файла: %v\n", err)
	}
}
