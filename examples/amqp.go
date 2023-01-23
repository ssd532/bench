package main

import (
	"fmt"
	"time"

	"github.com/ssd532/bench/v2"
	"github.com/ssd532/bench/v2/requester"
)

func main() {
	r := &requester.AMQPRequesterFactory{
		URL:         "amqp://user:bitnami@localhost:5672",
		PayloadSize: 10000,
		Queue:       "benchqueue",
		Exchange:    "benchmark",
	}

	benchmark := bench.NewBenchmark(r, 10000, 3, 60*time.Second, 0)
	summary, err := benchmark.Run()
	if err != nil {
		panic(err)
	}

	fmt.Println(summary)
	summary.GenerateLatencyDistribution(nil, "rmq-amqp-payload-10k-persec-10k-nconn-3-dur-1min-ram-1g.txt")
}
