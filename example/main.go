package main

import (
	"context"
	"fmt"
	"github.com/hoanguyenkh/promise4g"
	"time"
)

func main() {
	ctx := context.Background()

	p1 := promise4g.New(func(resolve func(string), reject func(error)) {
		time.Sleep(100 * time.Millisecond)
		resolve("one")
	})

	p2 := promise4g.New(func(resolve func(string), reject func(error)) {
		time.Sleep(200 * time.Millisecond)
		resolve("two")
	})

	p3 := promise4g.New(func(resolve func(string), reject func(error)) {
		time.Sleep(300 * time.Millisecond)
		resolve("three")
	})

	allPromise := promise4g.All(ctx, p1, p2, p3)
	results, err := allPromise.Await(ctx)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Results:", results)
	}
}
