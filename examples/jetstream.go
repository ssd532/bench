package main

import (
	"fmt"
	"time"

	"github.com/ssd532/bench/v2"
	"github.com/ssd532/bench/v2/requester"
)

func main() {
	r := &requester.JetStreamRequesterFactory{
		URL:                  "localhost:4222",
		PayloadSize:          1000,
		Stream:               "benchmark",
		AsyncPublish:         true,
		MaxPublishAckPending: 1000, // not useful if async publish false
	}

	benchmark := bench.NewBenchmark(r, 100000, 1, 30*time.Second, 0)
	summary, err := benchmark.Run()
	if err != nil {
		panic(err)
	}

	fmt.Println(summary)
	summary.GenerateLatencyDistribution(nil, "jetstream.txt")
}
