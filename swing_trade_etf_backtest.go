// This program will be used to backtest a swing trading strategy utilizing
// leveraged ETFs such as TQQQ and SQQQ. For this strategy, position 
// management will be completely determined by price action within specified
// multiples of the ETF's Average True Range. 

// The program will accept two parameters: a start date and an end date. The
// program will then output the results of the backtest if we were to 
// implement the strategy between the two dates.
// USAGE: go run swing_trade_etf_backtest.go 01-01-2000 01-01-2005

// Key Assumptions:
// - TQQQ and SQQQ reflect exactly 3x the daily percentage change in QQQ
package main

import (
	"encoding/csv"
	"fmt"
	// "io/ioutil"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	// symbol, month, day, year, month, day, year
	YAHOO_FINANCE_API_URL string = "http://real-chart.finance.yahoo.com/table.csv?s=%s&d=%s&e=%s&f=%s&g=d&a=%s&b=%s&c=%s&ignore=.csv"
	SUMMARY_FILE          string = "/Users/albert/Desktop/stocks/output/%s_summary.txt"
	// DETAILED_SUMMARY_FILE string = "/Users/albert/Desktop/stocks/output/%s_detailed_summary.txt"
	ETF                   string = "QQQ"
	NUM_YEARS_DATA        int    = 15
	LONG_TYPE             string = "LONG"
	SHORT_TYPE            string = "SHORT"
	MIN_TYPE              string = "MIN"
	MAX_TYPE              string = "MAX"
	TIME_LAYOUT           string = "2006-01-02"

	// ATR Configuration and Multiples
	ATR_WINDOW               int     = 50
	ATR_MULT_CUT_POSITION    float64 = 1.0
	ATR_MULT_EXIT_POSITION   float64 = 1.5
	ATR_MULT_CHANGE_POSITION float64 = 2.0
	ATR_MULT_ADD_POSITION    float64 = 2.5

	// Portfolio Configuration
	INITIAL_CAPITAL   float64 = 100000.0
	LEVERAGE_MULTIPLE float64 = 1.0
	LONG_PARTIAL_PERCENTAGE float64 = 0.5
	SHORT_MAX_PERCENTAGE float64 = 0.5
)

type Portfolio struct {
	StartDate string
	EndDate string
	CurrentDate string
	InitialValue float64
	CurrentValue float64
	CurrentPosition interface{}
	ClosedPositions []Position
	Transactions []Transaction
}

func (p *Portfolio) EnterInitialPosition(startDate string, data StockData) {

}

// updates the portfolio's current position with the current day's data
// since we're simulating only EOD trades if closing prices exceed ATR multiples,
// we take the current closing price no matter what it is
func (p *Portfolio) UpdatePortfolio(currentDate string, currClose float64) {
	if p.CurrentPosition != nil {
		currPosition := p.CurrentPosition.(Position)
		prevClose := currPosition.CurrentPrice
		currPosition.CurrentDate = currentDate
		currPosition.CurrentPrice = currClose
		p.CurrentPosition = currPosition

		// update portfolio value
		percentChange := (currClose / prevClose) - 1
		p.CurrentValue *= (1 + (percentChange * LEVERAGE_MULTIPLE))
		fmt.Printf("$%.2f\n", p.CurrentValue)
	}
	// TODO: add logging
	// date, short or long, percentage gain, new value
}

// checks if the current closing price moves us past an ATR multiple. if it does,
// we adjust the current position accordingly.
// returns true if the position was adjusted
func (p *Portfolio) AdjustPosition(currentDate string, currClose, currATR float64) bool {
	// pass along current extremes
	if p.CurrentPosition != nil {
		currPosition := p.CurrentPosition.(Position)
		currExtreme := currPosition.ReferencedExtreme
		if currPosition.Type == LONG_TYPE {
			if currExtreme.Type == MIN_TYPE {
				if currClose < currExtreme.Value {
					// change position and update extreme
				} else {
					if currClose > currExtreme.getATRThreshold(ATR_MULT_ADD_POSITION) {
						// ensure we are 100% long
					} else if currClose > currExtreme.getATRThreshold(ATR_MULT_CHANGE_POSITION) {
						// ensure we are 100% long
					}
				}
			} else if currExtreme.Type == MAX_TYPE {
				if currClose >= currExtreme.Value {
					currExtreme.Value = currClose
					currExtreme.ATR = currATR
				} else {
					// check ATR multiples
				}
			} else {
				panic("ILLEGAL TYPE")
			}
		} else if currPosition.Type == SHORT_TYPE {
			if currExtreme.Type == MIN_TYPE {
				if currClose <= currExtreme.Value {
					currExtreme.Value = currClose
					currExtreme.ATR = currATR
				} else {
					// check ATR multiples
				}
			} else if currExtreme.Type == MAX_TYPE {

			} else {
				panic("ILLEGAL TYPE")
			}
		} else {
			panic("ILLEGAL TYPE")
		}
	}

	return true
}

func (p *Portfolio) ToString() string {
	// TODO: print out current position and last transaction
	return fmt.Sprintf("%s - Current Capital: $%.2f; Current Position", p.CurrentDate, p.CurrentValue)
}

type Position struct {
	Symbol string
	Type string
	InvestedCapital float64
	PercentageInvested float64
	EntryDate string
	EntryPrice float64
	ReferencedExtreme Extreme
	CurrentDate string
	CurrentPrice float64
}

type Extreme struct {
	Type string
	Value float64
	ATR float64
}

func (e *Extreme) getATRThreshold(multiple float64) float64 {
	if e.Type == MAX_TYPE {
		return e.Value - (e.ATR * multiple)
	} else if e.Type == MIN_TYPE {
		return e.Value + (e.ATR * multiple)
	} else {
		panic("ILLEGAL TYPE")
	}
}

type Transaction struct {
	Date string

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
	ATR      float64
}

func (b *StockBar) ToString() string {
	return fmt.Sprintf("%s - Open: $%.2f; Close: $%.2f; High: $%.2f; Low: $%.2f; ATR%d: %.2f", b.Date, b.Open, b.Close, b.High, b.Low, ATR_WINDOW, b.ATR)
}

func main() {
	args := os.Args[1:]
	portfolio := Portfolio{args[0], args[1], args[0], INITIAL_CAPITAL, INITIAL_CAPITAL, nil, make([]Position, 0), make([]Transaction, 0)}
	ETFData := getStockData(ETF, NUM_YEARS_DATA)
	simulate(portfolio, ETFData)
	// for _, bar := range ETFData.Data {
	// 	fmt.Println(bar.ToString())
	// }
	fmt.Println(portfolio.ToString())
}

	// EntryDate string
	// EntryPrice float64
	// ReferencedExtreme Extreme
	// CurrentDate string
	// CurrentPrice float64

func simulate(portfolio Portfolio, etfData StockData) {
	startDate, _ := time.Parse(TIME_LAYOUT, portfolio.StartDate)
	endDate, _ := time.Parse(TIME_LAYOUT, portfolio.EndDate)
	for _, bar := range etfData.Data {
		currBarDate, _ := time.Parse(TIME_LAYOUT, bar.Date)
		if (currBarDate.After(startDate) || currBarDate.Equal(startDate)) && (currBarDate.Before(endDate) || currBarDate.Equal(endDate)) {
			// create initial position
			if portfolio.CurrentPosition == nil {
				portfolio.CurrentPosition = Position{ETF, LONG_TYPE, portfolio.InitialValue, 1, currBarDate.String(), bar.Close, Extreme{}, currBarDate.String(), bar.Close}
			} else {
				portfolio.UpdatePortfolio(currBarDate.String(), bar.Close)
			}
		}
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
