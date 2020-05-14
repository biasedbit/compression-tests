package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	"github.com/dustin/go-humanize"

	"github.com/biasedbit/comp-test/internal/stressor"
)

func main() {
	cfg := stressor.Config{}
	kong.Parse(&cfg)

	fmt.Printf("%d producers vs %d consumers using %s...\n",
		cfg.Producers, cfg.Producers, cfg.Codec)

	s := stressor.New(cfg)
	s.Start()
	lastThroughput := 0.0
	lastConsumed := uint64(0)
	go func() {
		interval := 5.0
		for {
			consumed := s.Count()
			recentThroughput := printStatus(s, consumed, lastConsumed, lastThroughput, interval)
			lastThroughput = recentThroughput
			lastConsumed = consumed
			<-time.After(time.Duration(interval) * time.Second)
		}
	}()

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	sigint := make(chan os.Signal)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
	fmt.Println("waiting for ctrl-c...")
	<-sigint
	s.Stop()
}

func printStatus(
	s *stressor.Stressor,
	consumed uint64,
	lastConsumed uint64,
	lastThroughput float64,
	interval float64,
) float64 {
	m := runtime.MemStats{}
	runtime.ReadMemStats(&m)
	recentThroughput := float64(consumed-lastConsumed) / interval

	fmt.Printf(
		"time: %.2fs, consumed: %d, throughput: %.0f msg/s (vol: %s/s, ratio: %.2f), recent: %.0f msg/s (%+.2f%%), goroutines: %d, mem: %s\n",
		s.Duration().Seconds(), s.Count(), s.CountThroughput(),
		humanize.Bytes(uint64(s.VolumeDecThroughput())),
		s.Ratio(),
		recentThroughput,
		((recentThroughput/lastThroughput)-1)*100,
		runtime.NumGoroutine(),
		humanize.Bytes(m.HeapAlloc))
	return recentThroughput
}
