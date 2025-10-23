package main

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

var ()

func delPast() {
	var fr, err = excelize.OpenFile("testRebase.xlsx")
	if err != nil {
		fmt.Println("oшибка открытия файла", err)
	}
	defer fr.Close()
	tDate := tomorrowDate(-1)
	fmt.Println(tDate)
	line, err := fr.GetRows("Sheet1")
	for i, r := range line {
		for x := range 9 {
			cellRef, err := excelize.CoordinatesToCellName(x+1, 1+i)
			if err != nil {
				fmt.Printf("Не удалось преобразовать координаты: %v\n", err)
				return
			}
			fr.SetCellValue("Sheet1", cellRef, nil)

		}
		if strings.Contains(r[0], tDate) {
			break
		}
	}
	fr.Save()
}
