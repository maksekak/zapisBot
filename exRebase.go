package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

var ()

func waitUntilNextDay() time.Duration {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	return next.Sub(now)
}

func runDailyUpdater(fr *excelize.File) {
	for {
		sleepTime := waitUntilNextDay()
		fmt.Printf("Ожидание до следующих суток: %v\n", sleepTime)
		time.Sleep(sleepTime)
		delPast(fr)

	}
}

func delPast(fr *excelize.File) {
	mu.Lock()
	defer mu.Unlock()
	tDate := tomorrowDate(0)
	fmt.Println(tDate)
	var indexRows []int
	line, err := fr.GetRows("Sheet1")
	if err != nil {
		fmt.Println(err)
	}
	for i, r := range line {
		indexRows = append(indexRows, i)
		if strings.Contains(r[0], tDate) {
			break
		}
	}
	for i := len(indexRows) - 1; i >= 0; i-- {
		fr.RemoveRow("Sheet1", i)
	}
}
func addFut(fr *excelize.File) {
	line, err := fr.GetRows("Sheet1")
	lenn := len(line)
	if err != nil {
		fmt.Println(err)
	}
	p := 22 //len(indexRows) / 5
	time := []string{"9", "10", "11", "14", "15", "16", "17", "18"}
	kostil := []string{"", "", "", "", "", "", "", "", "|"}
	var dateToAdd []string
	for i := range p {
		dateToAdd = append(dateToAdd, tomorrowDate(i+1))
	}

	var result []string
	for i, item := range dateToAdd {
		// Добавляем текущий элемент
		result = append(result, item)
		// Если это НЕ последний элемент — добавляем 4 пробелов как отдельные элементы
		if i < len(dateToAdd)-1 {
			for j := 0; j < 4; j++ {
				result = append(result, "") // каждый пробел — отдельный элемент
			}
		}
	}

	cords, _ := excelize.CoordinatesToCellName(1, lenn+1)
	fmt.Println(cords)
	fr.SetSheetCol("Sheet1", cords, &result)
	fmt.Print(result)
	for i := lenn + 1; i <= len(result); i++ {
		cordss, _ := excelize.CoordinatesToCellName(2, i)
		fmt.Println(cordss)
		fr.SetSheetRow("Sheet1", cordss, &kostil)
		//if i == 0 || i%5 == 0 {

		fr.SetSheetRow("Sheet1", cordss, &time)
		//}

	}
	fmt.Println(lenn, len(result))
	fr.Save()
}
