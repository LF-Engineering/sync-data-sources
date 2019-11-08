package main

import (
	"fmt"
	"time"
)

func main() {
	dtStart := time.Now()
	dtEnd := time.Now()
	fmt.Printf("Time: %v\n", dtEnd.Sub(dtStart))
}
