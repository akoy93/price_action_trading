package main

import (
	"encoding/csv"
	"fmt"
	// "io/ioutil"
	"math"
	"net/http"
	// "os"
	"strconv"
	"time"
)

const (
	// symbol, month, day, year, month, day, year
	YAHOO_FINANCE_API_URL string = "http://real-chart.finance.yahoo.com/table.csv?s=%s&d=%s&e=%s&f=%s&g=d&a=%s&b=%s&c=%s&ignore=.csv"
	TRANSACTIONS_FILE     string = "/Users/albert/Desktop/stocks/output/%s_transactions.txt"
	SUMMARY_FILE          string = "/Users/albert/Desktop/stocks/output/%s_summary.txt"
	ETF                   string = "QQQ"
	NUM_YEARS_DATA        int    = 15

	// ATR Configuration and Multiples
	ATR_WINDOW               int     = 50
	ATR_MULT_CUT_POSITION    float64 = 1.0
	ATR_MULT_EXIT_POSITION   float64 = 1.5
	ATR_MULT_CHANGE_POSITION float64 = 2.0
	ATR_CUT_PERCENTAGE       float64 = 0.5

	// Portfolio Configuration
	INITIAL_CAPITAL   float64 = 100000.0
	LEVERAGE_MULTIPLE float64 = 3.0
)

// type Portolio struct {
// 	OpenPositions map[string]Position
// }

// type Position struct {
// 	Symbol string
// 	EntryDate string

// }

// type Transaction struct {
// 	Date string

// }

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
	ATR      float64
}

func (b *StockBar) ToString() string {
	return fmt.Sprintf("%s - Open: $%.2f; Close: $%.2f; High: $%.2f; Low: $%.2f; ATR%d: %.2f", b.Date, b.Open, b.Close, b.High, b.Low, ATR_WINDOW, b.ATR)
}

func main() {
	ETFData := getStockData(ETF, NUM_YEARS_DATA)
	for _, bar := range ETFData.Data {
		fmt.Println(bar.ToString())
	}
}

func getStockData(symbol string, numYears int) StockData {
	t := time.Now()
	month := fmt.Sprintf("%02d", t.Month()-1)
	itoa := strconv.Itoa
	return getStockDataHelper(symbol, month, itoa(t.Day()), itoa(t.Year()), month, itoa(t.Day()), itoa(t.Year()-numYears))
}

func getStockDataHelper(symbol, endMonth, endDay, endYear, startMonth, startDay, startYear string) StockData {
	url := fmt.Sprintf(YAHOO_FINANCE_API_URL, symbol, endMonth, endDay, endYear, startMonth, startDay, startYear)
	resp, httpErr := http.Get(url)
	if httpErr != nil {
		fmt.Printf("Unable to retrieve data for %s. Retrying...", symbol)
		return getStockDataHelper(symbol, endMonth, endDay, endYear, startMonth, startDay, startYear)
	}
	defer resp.Body.Close()

	reader := csv.NewReader(resp.Body)
	rawCSVdata, csvErr := reader.ReadAll()

	if csvErr != nil {
		panic(fmt.Sprintf("ERROR: Unable to parse response for %s.", symbol))
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

	// compute ATR
	var tempATRList []float64
	for i := 1; i < len(allBars); i++ {
		allBars[i].ATR = getUpdatedATR(&tempATRList, getTradingRange(allBars[i-1], allBars[i]))
	}

	return StockData{allBars, symbol}
}

func getTradingRange(prevBar, currBar StockBar) float64 {
	// high and low of today
	max := math.Abs(currBar.High - currBar.Low)
	// today's high and yesterday's close
	currHigh := math.Abs(currBar.High - prevBar.Close)
	if currHigh > max {
		max = currHigh
	}
	// today's low and yesterday's close
	currLow := math.Abs(currBar.Low - prevBar.Close)
	if currLow > max {
		max = currLow
	}

	return max
}

func getUpdatedATR(list *[]float64, newValue float64) float64 {
	if len(*list) < ATR_WINDOW {
		*list = append(*list, newValue)
		return -1.0
	} else {
		*list = append((*list)[1:], newValue)
		sum := 0.0
		for _, val := range *list {
			sum += val
		}
		return sum / float64(ATR_WINDOW)
	}
}
