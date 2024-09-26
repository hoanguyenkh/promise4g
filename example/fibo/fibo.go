package main

import (
	"context"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"time"

	"github.com/hoanguyenkh/promise4g"
)

// Function to calculate Fibonacci number
func fibo(n int) int {
	if n <= 1 {
		return n
	}
	return fibo(n-1) + fibo(n-2)
}

func main() {

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			panic(err)
		}
	}()
	ctx := context.Background()
	for {
		// List of Fibonacci numbers to calculate
		numbers := []int{3, 4, 5, 4, 3}
		var promises []*promise4g.Promise[int]
		start := time.Now()

		// Create promises for Fibonacci numbers
		for _, n := range numbers {
			tmp := promise4g.New(func(resolve func(int), reject func(error)) {
				time.Sleep(100 * time.Millisecond)
				resolve(fibo(n))
			})
			promises = append(promises, tmp)
		}
		elapsed := time.Since(start)
		fmt.Println("t1:", elapsed)

		// Wait for all promises to complete
		allPromise := promise4g.All(ctx, promises...)
		results, err := allPromise.Await(ctx)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Println("Fibonacci Results:", results)
		}
		elapsed = time.Since(start)
		fmt.Println("t2:", elapsed)
		time.Sleep(1 * time.Second)
	}
}
