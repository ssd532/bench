package main

import (
	"fmt"
	"time"

	"github.com/ssd532/bench/v2"
	"github.com/ssd532/bench/v2/requester"
)

func main() {
	r := &requester.KafkaRequesterFactory{
		URLs:        []string{"localhost:9092"},
		PayloadSize: 1000,
		Topic:       "benchmark",
		DoConsume:   false,
		IsAsync:     false,
	}

	benchmark := bench.NewBenchmark(r, 1000000, 3, 600*time.Second, 0)
	summary, err := benchmark.Run()
	if err != nil {
		panic(err)
	}

	fmt.Println(summary)
	summary.GenerateLatencyDistribution(nil, "out/kafka-noconsume-payload-1k-persec-1m-nconn-3-dur-10m.txt")
}
