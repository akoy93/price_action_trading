package main

import (
	"fmt"
)

const (
	ATR_WINDOW int = 50
)

func main() {
	var list []float64
	for i := 0.0; i < 100.0; i += 1.0 {
		fmt.Println(getUpdatedATR(&list, i))
	}
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
