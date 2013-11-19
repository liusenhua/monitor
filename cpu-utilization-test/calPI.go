package main

import (
    "fmt"
	"flag"
    "runtime"
    "time"
)

var ncpu = flag.Int("ncpu", 0, "number of CPUs to use")

var n int64 = 10000000000
var h float64 = 1.0 / float64(n)

func f(a float64) float64 {
    return 4.0 / (1.0 + a*a)
}

func chunk(start, end int64, c chan float64) {
    var sum float64 = 0.0
    for i := start; i < end; i++ {
        x := h * (float64(i) + 0.5)
        sum += f(x)
    }
    c <- sum * h
}

func main() {
	flag.Parse()

	// Tell go how many cpu core it will be used 
    np := *ncpu
	if np == 0 {
		np = runtime.NumCPU() 
	}
    runtime.GOMAXPROCS(np)

    start := time.Now()

    var pi float64
    c := make(chan float64, np)

    for i := 0; i < np; i++ {
        go chunk(int64(i)*n/int64(np), (int64(i)+1)*n/int64(np), c)
    }

    for i := 0; i < np; i++ {
        pi += <-c
    }

    fmt.Println("Pi: ", pi)

    end := time.Now()

    fmt.Printf("spend time: %vs\n", end.Sub(start).Seconds())
}
