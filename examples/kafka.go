package main

import (
	"fmt"
	"time"

	"github.com/kishansairam9/bench/v2"
	"github.com/kishansairam9/bench/v2/requester"
)

func main() {
	r := &requester.KafkaRequesterFactory{
		URLs:        []string{"localhost:9092"},
		PayloadSize: 1000,
		Topic:       "benchmark",
	}

	benchmark := bench.NewBenchmark(r, 20000, 25, 30*time.Second, 0)
	summary, err := benchmark.Run()
	if err != nil {
		panic(err)
	}

	fmt.Println(summary)
	summary.GenerateLatencyDistribution(nil, "kafka.txt")
}
