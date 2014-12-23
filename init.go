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
	NUM_YEARS_DATA        int    = 1
	START_PIVOT_WIDTH     int    = 3
	PIVOT_WIDTH           int    = 5
)

type Line struct {
	X1, Y1, X2, Y2 float64
	slope() float64
}

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

func slope(l* Line) float64 {
	return (l.Y2 - l.Y1) / (l.X2 - l.X1)
}

func main() {
	t := time.Now()
	month := fmt.Sprintf("%02d", t.Month()-1)
	itoa := strconv.Itoa

	var c chan StockData = make(chan StockData, 15)

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

	for i := 0; i < 15; i++ {
		stock := <-c
		fmt.Println(stock.Symbol)
		for _, pivotIndex := range getStartPivots(stock, false) {
			fmt.Println(stock.Data[pivotIndex].Date)
		}
	}

	var input string
	fmt.Scanln(&input)
}

func getLines(stock StockData, startPivots []int, pivots []int, getHighLines bool) []Line, []Line, []Line {
	var lines []Line
	currentPivotIndex := 0

	for i, startPivot := range startPivots {
		var prevLine Line
		prevLine = nil
		for j := currentPivotIndex; j < len(pivots); j++ {
			pivot := pivots[j]
			if pivot > startPivot && pivots[currentPivotIndex] <= startPivot {
				currentPivotIndex = i
			}
			if pivot > startPivot {
				if getHighLines {
					currLine := Line(startPivot, stock.Data[startPivot].High, pivot, stock.Data[pivot].High)
					if prevLine == nil || currLine.slope() >= prevLine.slope() {
						prevLine = currLine
						lines = append(lines, prevLine)
					}
				} else {
					currLine := Line(startPivot, stock.Data[startPivot].Low, pivot, stock.Data[pivot].Low)
					if prevLine == nil || currLine.slope() <= prevLine.slope() {
						prevLine = currLine
						lines = append(lines, prevLine)
					}
				}
			}
		}
	}

	var trendLines []Line
	var trendChannelLines []Line
	var horizontalLines []Line

	for _, line := range lines {
		if getHighLines {
			if line.slope() > 0 {
				trendChannelLines = append(trendChannelLines, line)
			} else if line.slope() < 0 {
				trendLines = append(trendLines, line)
			} else {
				horizontalLines = append(horizontalLines, line)
			}
		} else {
			if line.slope() > 0 {
				trendLines = append(trendLines, line)
			} else if line.slope() < 0 {
				trendChannelLines = append(trendChannelLines, line)
			} else {
				horizontalLines = append(horizontalLines, line)
			}
		}
	}

	return trendLines, trendChannelLines, horizontalLines
}

// draw lines for low pivots:
// iteratively go through pivots
// have an anchor pivot (use start pivots as anchor pivots)
// create a line if a next pivot creates a line with a slope less than what's been seen so far

func getStartPivots(stock StockData, getHighPivots bool) []int {
	return getPivots(stock, getHighPivots, START_PIVOT_WIDTH)
}

func getPivots(stock StockData, getHighPivots bool, width int) []int {
	var pivots []int
	data := stock.Data

	for i := width; i < len(data)-width; i++ {
		isPivot := true
		for j := i - width; j <= i+width; j++ {
			if getHighPivots {
				if data[j].High > data[i].High {
					isPivot = false
					break
				}
			} else {
				if data[j].Low < data[i].Low {
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
		return
	}
	defer resp.Body.Close()

	reader := csv.NewReader(resp.Body)
	rawCSVdata, csvErr := reader.ReadAll()

	if csvErr != nil {
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
