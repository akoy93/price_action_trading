package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	// symbol, month, day, year, month, day, year
	YAHOO_FINANCE_API_URL      string  = "http://real-chart.finance.yahoo.com/table.csv?s=%s&d=%s&e=%s&f=%s&g=d&a=%s&b=%s&c=%s&ignore=.csv"
	STOCK_FILE                 string  = "/Users/albert/Desktop/stocks/stocks.txt"
	OUTPUT_FILE                string  = "/Users/albert/Desktop/stocks/output/%s_output.txt"
	OUTPUT_SYMBOLS_FILE        string  = "/Users/albert/Desktop/stocks/output/%s_symbols.txt"
	NUM_YEARS_DATA             int     = 1
	START_PIVOT_WIDTH          int     = 3
	PIVOT_WIDTH                int     = 5
	HORIZONTAL_SLOPE_THRESHOLD float64 = 0.005
	TREND_LINE                 string  = "Trend Line"
	TREND_CHANNEL_LINE         string  = "Trend Channel Line"
	SUPPORT                    string  = "Support"
	NUM_INTERSECTIONS_REQUIRED int     = 2
)

type Line struct {
	X1 int
	Y1 float64
	X2 int
	Y2 float64
}

type Intersection struct {
	Line  Line
	Price float64
	Date  string
	Type  string
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

func (l *Line) Slope() float64 {
	return (l.Y2 - l.Y1) / float64(l.X2-l.X1)
}

func (l *Line) Crosses(x int, high, low float64) (float64, bool) {
	projection := l.GetProjection(x)
	return projection, projection <= high && projection >= low
}

func (l *Line) GetProjection(x int) float64 {
	return l.Y1 + (l.Slope() * float64(x-l.X1))
}

func (l *Line) ToString(stock *StockData) string {
	str := ""
	str += fmt.Sprintf("%s - %s - %d\n", stock.Data[l.X1].Date, fmt.Sprintf("$%.2f", l.Y1), l.X1)
	str += fmt.Sprintf("%s - %s - %d\n", stock.Data[l.X2].Date, fmt.Sprintf("$%.2f", l.Y2), l.X2)
	return str
}

func (l *Line) NoPivotsBelow(stock *StockData, pivots []int) bool {
	for _, pivot := range pivots {
		projection := l.GetProjection(pivot)
		if stock.Data[pivot].Low < projection {
			return false
		}
	}
	return true
}

func (l *Line) NoPivotsAbove(stock *StockData, pivots []int) bool {
	for _, pivot := range pivots {
		projection := l.GetProjection(pivot)
		if stock.Data[pivot].High > projection {
			return false
		}
	}
	return true
}

func main() {
	t := time.Now()
	month := fmt.Sprintf("%02d", t.Month()-1)
	itoa := strconv.Itoa

	var c chan interface{} = make(chan interface{}, 1)

	file, err := os.Open(STOCK_FILE)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	numLines := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		symbol := scanner.Text()
		numLines++
		go getStockData(c, symbol, month, itoa(t.Day()), itoa(t.Year()), month, itoa(t.Day()), itoa(t.Year()-NUM_YEARS_DATA))
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	output := ""
	outputSymbols := ""
	for i := 1; i <= numLines; i++ {
		data := <-c
		if data != nil {
			stock := data.(StockData)
			fmt.Printf("(%d/%d) Evaluating %s...\n", i, numLines, stock.Symbol)
			trendChannelLines, trendLines, horizontalLines := getLines(&stock, false)
			tclInts, tlInts, hInts := getAllIntersections(&stock, trendChannelLines, trendLines, horizontalLines)
			setup, ok := getBestSetup(tclInts, tlInts, hInts)
			if ok {
				output += fmt.Sprintf("=============== %s ===============\n", stock.Symbol)
				output += "++++++++++++ Best Setup ++++++++++++\n"
				outputSymbols += stock.Symbol + "\n"
				for _, intersection := range setup {
					output += fmt.Sprintf("----- %s -----\n", intersection.Type)
					output += intersection.Line.ToString(&stock)
					output += fmt.Sprintf("Crosses $%.2f on %s\n", intersection.Price, intersection.Date)
				}

				// print all data
				output += "++++++++++++ All Lines ++++++++++++\n"
				lines := [][]Intersection{tclInts, tlInts, hInts}
				for _, set := range lines {
					for _, intersection := range set {
						output += fmt.Sprintf("----- %s -----\n", intersection.Type)
						output += intersection.Line.ToString(&stock)
						output += fmt.Sprintf("Crosses $%.2f on %s\n", intersection.Price, intersection.Date)
					}
				}
			}
		} else {
			fmt.Printf("(%d/%d) Evaluating...\n", i, numLines)
		}
	}

	// write to file
	outputBytes := []byte(output)
	outputSymbolsBytes := []byte(outputSymbols)

	outputErr := ioutil.WriteFile(fmt.Sprintf(OUTPUT_FILE, t.Format("01-02-2006")), outputBytes, 0644)
	outputSymbolsErr := ioutil.WriteFile(fmt.Sprintf(OUTPUT_SYMBOLS_FILE, t.Format("01-02-2006")), outputSymbolsBytes, 0644)
	if outputErr != nil || outputSymbolsErr != nil {
		fmt.Println("ERROR writing to file!")
	} else {
		fmt.Println("DONE!")
	}
}

func getBestSetup(trendChannelLineIntersections, trendLineIntersections, horizontalLineIntersections []Intersection) ([]Intersection, bool) {
	var bestPair []Intersection
	bestRange := math.MaxFloat64

	for _, tcl := range trendChannelLineIntersections {
		for _, tl := range trendLineIntersections {
			currRange, currPair := getPairRange(tcl, tl)
			if currRange < bestRange {
				bestRange = currRange
				bestPair = currPair
			}
		}
	}

	setup := append(bestPair, horizontalLineIntersections...)

	return setup, len(setup)-len(horizontalLineIntersections) >= NUM_INTERSECTIONS_REQUIRED
}

func getPairRange(tclIntersection, tlIntersection Intersection) (float64, []Intersection) {
	return math.Abs(tclIntersection.Price - tlIntersection.Price), []Intersection{tclIntersection, tlIntersection}
}

func getAllIntersections(stock *StockData, trendChannelLines, trendLines, horizontalLines []Line) ([]Intersection, []Intersection, []Intersection) {
	trendChannelLineIntersections := getIntersections(stock, TREND_CHANNEL_LINE, trendChannelLines)
	trendLineIntersections := getIntersections(stock, TREND_LINE, trendLines)
	horizontalLineIntersections := getIntersections(stock, SUPPORT, horizontalLines)

	return trendChannelLineIntersections, trendLineIntersections, horizontalLineIntersections
}

func getIntersections(stock *StockData, lineType string, lines []Line) []Intersection {
	var intersections []Intersection

	lastBarIndex := len(stock.Data) - 1
	for _, line := range lines {
		price, crosses := line.Crosses(lastBarIndex, stock.Data[lastBarIndex].High, stock.Data[lastBarIndex].Low)
		if crosses {
			intersection := Intersection{line, price, stock.Data[lastBarIndex].Date, lineType}
			intersections = append(intersections, intersection)
		}
	}

	return intersections
}

// draw lines for low pivots:
// iteratively go through pivots
// have an anchor pivot (use start pivots as anchor pivots)
// create a line if a next pivot creates a line with a slope less than what's been seen so far

// returns trendLines, trendChannelLines, horizontalLines
func getLines(stock *StockData, getOverLines bool) ([]Line, []Line, []Line) {
	startPivots := getStartPivots(stock, getOverLines)
	endPivots := getPivots(stock, getOverLines, PIVOT_WIDTH)
	return getLinesFromPivots(stock, startPivots, endPivots, getOverLines)
}

func getLinesFromPivots(stock *StockData, startPivots []int, pivots []int, getHighLines bool) ([]Line, []Line, []Line) {
	var lines []Line
	currentPivotIndex := 0

	for _, startPivot := range startPivots {
		var prevLine interface{}
		prevLine = nil
		for j := currentPivotIndex; j < len(pivots); j++ {
			pivot := pivots[j]
			if pivot > startPivot {
				if pivots[currentPivotIndex] <= startPivot {
					currentPivotIndex = j
				}

				// interface conversion
				var prevLineConverted Line
				if prevLine != nil {
					prevLineConverted = prevLine.(Line)
				}

				// draw lines
				if getHighLines {
					currLine := Line{startPivot, stock.Data[startPivot].High, pivot, stock.Data[pivot].High}
					if currLine.NoPivotsAbove(stock, pivots[j:]) && (prevLine == nil || currLine.Slope() >= prevLineConverted.Slope()) {
						prevLine = currLine
						lines = append(lines, currLine)
					}
				} else {
					currLine := Line{startPivot, stock.Data[startPivot].Low, pivot, stock.Data[pivot].Low}
					if currLine.NoPivotsBelow(stock, pivots[j:]) && (prevLine == nil || currLine.Slope() <= prevLineConverted.Slope()) {
						prevLine = currLine
						lines = append(lines, currLine)
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
			if line.Slope() > HORIZONTAL_SLOPE_THRESHOLD {
				trendChannelLines = append(trendChannelLines, line)
			} else if line.Slope() < -HORIZONTAL_SLOPE_THRESHOLD {
				trendLines = append(trendLines, line)
			} else {
				horizontalLines = append(horizontalLines, line)
			}
		} else {
			if line.Slope() > HORIZONTAL_SLOPE_THRESHOLD {
				trendLines = append(trendLines, line)
			} else if line.Slope() < -HORIZONTAL_SLOPE_THRESHOLD {
				trendChannelLines = append(trendChannelLines, line)
			} else {
				horizontalLines = append(horizontalLines, line)
			}
		}
	}

	return trendChannelLines, trendLines, horizontalLines
}

func getStartPivots(stock *StockData, getHighPivots bool) []int {
	return getPivots(stock, getHighPivots, START_PIVOT_WIDTH)
}

func getPivots(stock *StockData, getHighPivots bool, width int) []int {
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

func getStockData(c chan interface{}, symbol, endMonth, endDay, endYear, startMonth, startDay, startYear string) {
	url := fmt.Sprintf(YAHOO_FINANCE_API_URL, symbol, endMonth, endDay, endYear, startMonth, startDay, startYear)
	resp, httpErr := http.Get(url)
	if httpErr != nil {
		go getStockData(c, symbol, endMonth, endDay, endYear, startMonth, startDay, startYear)
		return
	}
	defer resp.Body.Close()

	reader := csv.NewReader(resp.Body)
	rawCSVdata, csvErr := reader.ReadAll()

	if csvErr != nil {
		c <- nil
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
