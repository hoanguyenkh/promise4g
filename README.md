# Promise4G

Promise4G is a Go library that provides a promise-like abstraction for handling asynchronous computations.

## Installation

To install Promise4G, use `go get`:

```sh
go get github.com/hoanguyenkh/promise4g
```

## Usage

Here's a simple example to demonstrate how to use Promise4G:
```go
package main

import (
	"context"
	"fmt"
	"github.com/hoanguyenkh/promise4g"
	"time"
)

func main() {
	ctx := context.Background()

	p1 := promise4g.New(ctx, func(resolve func(string), reject func(error)) {
		time.Sleep(100 * time.Millisecond)
		resolve("one")
	})

	p2 := promise4g.New(ctx, func(resolve func(string), reject func(error)) {
		time.Sleep(200 * time.Millisecond)
		resolve("two")
	})

	p3 := promise4g.New(ctx, func(resolve func(string), reject func(error)) {
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

```

## Referer
 1) https://github.com/chebyrash/promise