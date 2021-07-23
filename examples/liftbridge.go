package main

import (
	"fmt"
	"time"

	"github.com/kishansairam9/bench/v2"
	"github.com/kishansairam9/bench/v2/requester"
)

func main() {
	r := &requester.LiftbridgeRequesterFactory{
		URLs:         []string{"localhost:9292"},
		PayloadSize:  1000,
		Stream:       "benchmark",
		AsyncPublish: true,
	}

	benchmark := bench.NewBenchmark(r, 100000, 1, 30*time.Second, 0)
	summary, err := benchmark.Run()
	if err != nil {
		panic(err)
	}

	fmt.Println(summary)
	summary.GenerateLatencyDistribution(nil, "liftbridge.txt")
}
