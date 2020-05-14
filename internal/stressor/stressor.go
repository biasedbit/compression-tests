package stressor

import (
	"github.com/biasedbit/comp-test/internal/compression"
	"sync"
	"time"

	"go.uber.org/atomic"
)

const backoff = 10 * time.Millisecond

func New(cfg Config) *Stressor {
	return &Stressor{
		cfg:       cfg,
		shutdown:  make(chan struct{}),
		prodwg:    sync.WaitGroup{},
		conswg:    sync.WaitGroup{},
		count:     atomic.NewUint64(0),
		volumeCmp: atomic.NewUint64(0),
		volumeDec: atomic.NewUint64(0),
		pipe:      make(chan []byte, 1000),
	}
}

type Stressor struct {
	cfg       Config
	shutdown  chan struct{}
	prodwg    sync.WaitGroup
	conswg    sync.WaitGroup
	count     *atomic.Uint64
	volumeCmp *atomic.Uint64
	volumeDec *atomic.Uint64
	pipe      chan []byte
	st        time.Time
	end       time.Time
}

func (s *Stressor) Start() {
	s.st = time.Now()
	s.end = time.Time{}
	s.count.Store(0)
	s.volumeCmp.Store(0)
	s.volumeDec.Store(0)
	s.conswg.Add(s.cfg.Consumers)
	for c := 0; c < s.cfg.Consumers; c++ {
		go s.consume()
	}
	s.prodwg.Add(s.cfg.Producers)
	for p := 0; p < s.cfg.Producers; p++ {
		go s.produce()
	}
}

func (s *Stressor) Stop() {
	close(s.shutdown)
	s.prodwg.Wait()
	close(s.pipe)
	s.conswg.Wait()
	s.end = time.Now()
}

func (s *Stressor) Duration() time.Duration {
	end := s.end
	if (end == time.Time{}) {
		end = time.Now()
	}
	return end.Sub(s.st)
}

func (s *Stressor) Count() uint64 {
	return s.count.Load()
}

func (s *Stressor) CountThroughput() float64 {
	return float64(s.count.Load()) / s.Duration().Seconds()
}

func (s *Stressor) VolumeCmpThroughput() float64 {
	return float64(s.volumeCmp.Load()) / s.Duration().Seconds()
}

func (s *Stressor) VolumeDecThroughput() float64 {
	return float64(s.volumeDec.Load()) / s.Duration().Seconds()
}

func (s *Stressor) Ratio() float64 {
	d, c := s.volumeDec.Load(), s.volumeCmp.Load()
	return float64(d) / float64(c)
}

func (s *Stressor) produce() {
	defer s.prodwg.Done()

	push := func() {
		data, _ := compression.RandomString(s.cfg.size())
		compressed, _ := compression.Compress(s.cfg.codec(), []byte(data))
		for {
			select {
			case <-s.shutdown:
				return
			case s.pipe <- compressed:
			default:
				<-time.After(backoff)
			}
		}
	}

	for {
		select {
		case <-s.shutdown:
			return
		case <-time.After(s.cfg.Interval):
			push()
		}
	}
}

func (s *Stressor) consume() {
	defer s.conswg.Done()
	for data := range s.pipe {
		decompressed, err := compression.Decompress(s.cfg.codec(), data)
		if err != nil {
			panic(err)
		}
		s.count.Inc()
		s.volumeCmp.Add(uint64(len(data)))
		s.volumeDec.Add(uint64(len(decompressed)))
	}
}
