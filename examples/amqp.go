package main

import (
	"fmt"
	"time"

	"github.com/kishansairam9/bench/v2"
	"github.com/kishansairam9/bench/v2/requester"
)

func main() {
	r := &requester.AMQPRequesterFactory{
		URL:         "amqp://localhost:5672",
		PayloadSize: 1000,
		Queue:       "benchqueue",
		Exchange:    "benchmark",
	}

	benchmark := bench.NewBenchmark(r, 20000, 1, 30*time.Second, 0)
	summary, err := benchmark.Run()
	if err != nil {
		panic(err)
	}

	fmt.Println(summary)
	summary.GenerateLatencyDistribution(nil, "amqp.txt")
}
