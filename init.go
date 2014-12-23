package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	// symbol, month, day, year, month, day, year
	YAHOO_FINANCE_API_URL string = "http://real-chart.finance.yahoo.com/table.csv?s=%s&d=%s&e=%s&f=%s&g=d&a=%s&b=%s&c=%s&ignore=.csv"
	STOCK_FILE            string = "/Users/albert/Desktop/stocks/stocks_sample.txt"
	NUM_YEARS_DATA        int    = 3
	START_PIVOT_WIDTH     int    = 3
	PIVOT_WIDTH           int    = 5
)

type StockData struct {
	Data   []StockBar
	Symbol string
}

type StockBar struct {
	Date     string
	Open     float64
	High     float64
	Low      float64
	Close    float64
	Volume   int
	AdjClose float64
}

func main() {
	t := time.Now()
	month := fmt.Sprintf("%02d", t.Month()-1)
	itoa := strconv.Itoa

	var c chan StockData = make(chan StockData, 1)

	file, err := os.Open(STOCK_FILE)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		symbol := scanner.Text()
		go getStockData(c, symbol, month, itoa(t.Day()), itoa(t.Year()), month, itoa(t.Day()), itoa(t.Year()-NUM_YEARS_DATA))
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	for i := 0; i < 6000; i++ {
		data := <-c
		// if data != nil {
			fmt.Println(data.Symbol)
			for i := range getPivots(data.Data, false, PIVOT_WIDTH) {
				fmt.Println(data.Data[i].Date)
			}
		// }
	}

	var input string
	fmt.Scanln(&input)
}

func getPivots(data []StockBar, getHighPivots bool, width int) []int {
	var pivots []int

	for i, value := range data[width:-width] {
		isPivot := true
		for j := i - width; j <= i+width; j++ {
			if getHighPivots {
				if data[j].High > value.High {
					isPivot = false
					break
				}
			} else {
				if data[j].Low < value.Low {
					isPivot = false
					break
				}
			}
		}
		if isPivot {
			pivots = append(pivots, i)
		}
	}

	return pivots
}

func getStockData(c chan StockData, symbol, endMonth, endDay, endYear, startMonth, startDay, startYear string) {
	url := fmt.Sprintf(YAHOO_FINANCE_API_URL, symbol, endMonth, endDay, endYear, startMonth, startDay, startYear)
	resp, httpErr := http.Get(url)
	if httpErr != nil {
		// c <- nil
		return
	}
	defer resp.Body.Close()

	reader := csv.NewReader(resp.Body)
	rawCSVdata, csvErr := reader.ReadAll()

	if csvErr != nil {
		// c <- nil
		return
	}

	// move raw CSV data to structs
	var oneBar StockBar
	var allBars []StockBar

	for _, row := range rawCSVdata[1:] {
		oneBar.Date = row[0]
		oneBar.Open, _ = strconv.ParseFloat(row[1], 64)
		oneBar.High, _ = strconv.ParseFloat(row[2], 64)
		oneBar.Low, _ = strconv.ParseFloat(row[3], 64)
		oneBar.Close, _ = strconv.ParseFloat(row[4], 64)
		oneBar.Volume, _ = strconv.Atoi(row[5])
		oneBar.AdjClose, _ = strconv.ParseFloat(row[6], 64)
		allBars = append([]StockBar{oneBar}, allBars...)
	}

	var data StockData
	data.Data = allBars
	data.Symbol = symbol

	c <- data
}
