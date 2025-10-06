package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

func tomorrowDate(p int) (d, m int) {
	now := time.Now()
	tomorrow := now.AddDate(0, 0, p)
	month := int(tomorrow.Month())
	day := tomorrow.Day()

	return day, month
}

func recCounts(f *excelize.File, cDay, cMonth int) (busyDays, ls int) {

	line, err := f.GetRows("Sheet1")
	if err != nil {
		fmt.Println("ошибка чтения строки", err)
	}
	busyDays = 0
	ls = 0
	tDate := fmt.Sprintf("%d.%d", cDay, cMonth)
	for i, r := range line {
		if strings.Contains(r[0], tDate) {
			busyDays += 1
			ls = i
		}

	}
	return busyDays, ls
}

/*
	func findFreeTime(f *excelize.File){
		line, err := f.GetRows("Sheet1")
		if err != nil {
			fmt.Println("ошибка чтения строки", err)
		}
		for _, r := range line {
			if strings.Contains(r[1], tDate) {
				busyDays += 1
			}

		}
	}
*/
func findFreeTime(arr []int) []int {
	// Определяем целевой диапазон
	const lo, hi = 9, 18

	// Создаем карта для присутствующих чисел в диапазоне (за исключением 12 и 13)
	present := make(map[int]bool)
	for _, v := range arr {
		if v == 12 || v == 13 {
			// Игнорируем 12 и 13, как если их нет
			continue
		}
		if v >= lo && v <= hi {
			present[v] = true
		}
	}

	// Собираем пропущенные числа в диапазоне, кроме 12 и 13
	var missing []int
	for x := lo; x <= hi; x++ {
		if x == 12 || x == 13 {
			continue
		}
		if !present[x] {
			missing = append(missing, x)
		}
	}
	return missing
}
