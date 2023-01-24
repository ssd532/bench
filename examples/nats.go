package main

import (
	"fmt"
	"time"

	"github.com/ssd532/bench/v2"
	"github.com/ssd532/bench/v2/requester"
)

func main() {
	r := &requester.NATSRequesterFactory{
		URL:         "nats://localhost:4222",
		PayloadSize: 1000,
		Subject:     "benchmark",
	}

	benchmark := bench.NewBenchmark(r, 100000, 1, 30*time.Second, 0)
	summary, err := benchmark.Run()
	if err != nil {
		panic(err)
	}

	fmt.Println(summary)
	summary.GenerateLatencyDistribution(nil, "nats.txt")
}
