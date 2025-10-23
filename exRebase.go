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
	fmt.Print(len(indexRows))
	fr.Save()
}
