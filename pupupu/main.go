/*
package main

import "fmt"

	func um(x int, y int) int {
		return x * y
	}

	func main() {
		fmt.Println(um(5, 6))

}

package main

import "fmt"

	type reper struct {
		name      string
		followers int
		swaga     bool
	}

	func main() {
		var TumniPrinc reper = reper{"Princ", 90, true}
		var tewiq reper = reper{"tewiq", 40, true}
		fmt.Println(TumniPrinc)
		fmt.Println(tewiq)

}

package main

import "fmt"

	func sortir(mas []int) {
		size := len(mas)
		for i := 0; i < size-1; i++ {
			for j := 0; j < size-i-1; j++ {
				if mas[j] > mas[j+1] {
					mas[j], mas[j+1] = mas[j+1], mas[j]
				}
			}
		}
	}

	func main() {
		var mas = []int{3, 4, 6, 2, 1, 7, 10}
		fmt.Println("начальный массив", mas)
		sortir(mas)
		fmt.Println("отсортированный массив", mas)

}

package main

import "fmt"

	func sravnim(mas1 []int, mas2 []int) {
		var size1 int = len(mas1)
		var size2 int = len(mas2)
		var prover1 bool = false
		var prover2 bool = true
		if size1 == size2 {
			prover1 = true

		}

		for i := 0; i < size1; i++ {
			if mas1[i] != mas2[i] {
				prover2 = false
			}

		}
		if prover1 == true && prover2 == true {
			fmt.Println("массивы одинаковые")

		} else {
			fmt.Println("массивы не одинаковые")
		}

}

	func main() {
		var mas1 = []int{3, 453, 6, 2, 6, 7, 8, 2}
		var mas2 = []int{3, 453, 6, 2, 6, 7, 8, 2}
		sravnim(mas1, mas2)

}

package main

import "fmt"

const (

	i = 8
	j
	k

)

	func main() {
		fmt.Println(i, j, k)
	}

package main

import "fmt"

	func main() {
		var h int
		fmt.Scan(h)
		fmt.Println(h)
	}

package main

import "fmt"

	func percec(mas1 []int,mas2 []int){
		var masp =[]int{}
		for i:=
	}

func main() {

}

package main

import (

	"fmt"
	"math/rand"

)

func main() {

	fmt.Println(rand.Intn(1000))
	fmt.Println(rand.Intn(1000))

}

package main

import "fmt"

	func main(){
		a := “mfgah134517095aldrfgvh8h”

}

package main

import (

	"fmt"
	"sort"

)

	func cafe(mas [7]int) {
		var (
			kupon int
			den   int
			p     int
			x     int
		)

		for i, num := range mas {

			if num > 100 {
				kupon++
				den = i + 1
			}
			p += num

		}
		fmt.Print(kupon, " ")
		sum := mas[den:]
		sort.Ints(sum)
		kupon = len(sum) - kupon
		sump := sum[kupon:]
		for _, num := range sump {
			x += num
		}
		p -= x
		fmt.Println(kupon)
		fmt.Println(p)

}

	func main() {
		masiv := [7]int{35, 40, 101, 59, 163, 43, 51}
		cafe(masiv)
	}
*/
package main

import (
	"fmt"
	"github.com/xuri/excelize/v2"
	"strconv"
	"strings"
)

func main() {
	f, err := excelize.OpenFile("test.xlsx")
	if err != nil {
		fmt.Println("oшибка открытия файла", err)
	}
	defer f.Close()
	p := 1
	d, m := tomorrowDate(p)
	fmt.Println(recCounts(f, d, m))
	rec, l := recCounts(f, d, m)
	fmt.Println(l)
	if rec >= 8 {
		p += 1
	} else {

		line, err := f.GetRows("Sheet1")
		if err != nil {
			fmt.Println("ошибка чтения строки", err)

		}
		arr := [10]int{}

		for i := l; r == "0"; i++ {
			j := 1
			r := line[i][1]

			if strings.Contains(r, "Время.ч") || r == "" || r == "0" {
				continue
			} else {
				//fmt.Println(i-1, r[1])
				s, _ := strconv.Atoi(r)
				arr[j] = s

			}

			//arr[i] = s
			//fmt.Println(arr)
			j += 1
		}
		fmt.Println(arr)
	}
}
