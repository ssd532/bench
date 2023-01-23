package main

import (
	"fmt"
	"time"

	"github.com/ssd532/bench/v2"
	"github.com/ssd532/bench/v2/requester"
)

func main() {
	r := &requester.RMQStreamRequesterFactory{
		URLs:        []string{"rabbitmq-stream://user:bitnami@localhost:5552"},
		PayloadSize: 1000,
		Stream:      "benchmark",
		DoConsume:   false,
	}

	benchmark := bench.NewBenchmark(r, 1000000, 3, 600*time.Second, 0)
	summary, err := benchmark.Run()
	if err != nil {
		panic(err)
	}

	fmt.Println(summary)
	summary.GenerateLatencyDistribution(nil, "out/rmqstream-noconsume-payload-1k-persec-1m-nconn-3-dur-10m.txt")
}
