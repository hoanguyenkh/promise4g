package main

import (
	"context"
	"fmt"
	"time"

	"github.com/hoanguyenkh/promise4g"
)

type httpResponse1 struct {
	Message   string `json:"message"`
	RequestId string `json:"requestId"`
}

type httpResponse2 struct {
	Username  string `json:"username"`
	RequestId string `json:"requestId"`
}

func fakeHttp1(url string) (httpResponse1, error) {
	fmt.Println("fakeHttp1", url)
	time.Sleep(100 * time.Millisecond)
	return httpResponse1{
		Message:   "hello world",
		RequestId: "requestId 1",
	}, nil
}

func fakeHttp2(url string) (httpResponse2, error) {
	fmt.Println("fakeHttp2", url)
	time.Sleep(200 * time.Millisecond)
	return httpResponse2{
		Username:  "username",
		RequestId: "requestId 2",
	}, nil
}

func main() {
	ctx := context.Background()
	t1 := time.Now()
	p1 := promise4g.AsyncTask(func() (any, error) {
		return fakeHttp1("fakeHttp1")
	})

	p2 := promise4g.AsyncTask(func() (any, error) {
		return fakeHttp2("fakeHttp2")
	})
	p := promise4g.All(ctx, p1, p2)
	results, err := p.Await(ctx)
	if err != nil {
		panic(err)
	}

	res1 := results[0].(httpResponse1)
	res2 := results[1].(httpResponse2)
	fmt.Println(res1.RequestId, res1.Message)
	fmt.Println(res2.RequestId, res2.Username)

	fmt.Println("total time: ", time.Since(t1))

}
