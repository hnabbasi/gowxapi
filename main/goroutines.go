package main

import (
	"fmt"
	"sync"
	"time"
)

var wg = sync.WaitGroup{}

var mtx = sync.RWMutex{}

func main() {
	wg.Add(3)

	// mtx.Lock()
	go countPrimesRunner(10)

	// mtx.Lock()
	go countPrimesRunner(100)
	// mtx.Lock()
	go countPrimesRunner(1_000)

	wg.Wait()

	countPrimes(10, "main")
	countPrimes(100, "main")
	countPrimes(1000, "main")
}

func countPrimesRunner(n int) {
	countPrimes(n, "countPrimesRunner")
	wg.Done()
	// mtx.Unlock()
}

func countPrimes(n int, c string) {
	start := time.Now()
	count := 0
	for i := 0; i <= n; i++ {
		if isPrime(i) {
			count++
		}
	}
	fmt.Printf("%v: %s. For %v, found %v prime numbers.\n", c, (time.Since(start)), n, count)
}

func isPrime(n int) bool {
	if n <= 1 {
		return false
	}
	for i := 2; i <= n/2; i++ {
		if n%i == 0 {
			return false
		}
	}
	return true
}

func getCurrent() {
	fmt.Println("1. Current conditions")
	wg.Done()
	mtx.Unlock()
}
func getHourly() {
	fmt.Println("2. Hourly forecast")
	wg.Done()
	mtx.Unlock()
}
func getWeekly() {
	fmt.Println("3. Weekly forecast")
	wg.Done()
	mtx.Unlock()
}
