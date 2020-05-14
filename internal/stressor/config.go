package stressor

import (
	"github.com/biasedbit/comp-test/internal/zstdchanpool"
	"github.com/biasedbit/comp-test/internal/zstdfixed"
	"github.com/biasedbit/comp-test/internal/oldzstd"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/gzip"
	"github.com/segmentio/kafka-go/lz4"
	"github.com/segmentio/kafka-go/snappy"
	"github.com/segmentio/kafka-go/zstd"
	"math/rand"
	"time"
)

type Config struct {
	Codec     string        `flag name:"codec" default:"zstd" help:"Codec implementation to use. Values: zstd (default, leaks goroutines), zstdold (DataDog, C impl), zstd-fixed (same as zstd, but uses finalizers to avoid leak), zstd-chanpool (swaps sync.Pool with channel-based pool)"`
	Producers int           `flag name:"producers" default:"100" help:"Number of producers."`
	Consumers int           `flag name:"consumers" default:"100" help:"Number of consumers."`
	Interval  time.Duration `flag name:"interval" default:"10ms" help:"Data generation interval."`
	MinSize   int           `flat name:"min_size" default:"2048" help:"Min size of data to compress."`
	MaxSize   int           `flat name:"max_size" default:"2048" help:"Max size of data to compress. When max > min, size is randomly picked for each block."`

	c kafka.CompressionCodec
}

func (c Config) codec() kafka.CompressionCodec {
	if c.c == nil {
		switch c.Codec {
		case "snappy":
			c.c = snappy.NewCompressionCodec()
		case "gzip":
			c.c = gzip.NewCompressionCodec()
		case "lz4":
			c.c = lz4.NewCompressionCodec()
		case "zstd":
			// github.com/segmentio/kafka-go/zstd; leaks
			c.c = zstd.NewCompressionCodec()
		case "zstd-old":
			// github.com/DataDog/zstd based CompressionCodec
			c.c = oldzstd.NewCompressionCodec()
		case "zstd-fixed":
			// similar to github.com/segmentio/kafka-go/zstd but uses finalizers and pools reader (instead of decoder)
			c.c = zstdfixed.NewCompressionCodec()
		case "zstd-chanpool":
			// copy of github.com/segmentio/kafka-go/zstd that uses channel-based pool instead of sync.Pool
			c.c = zstdchanpool.NewCompressionCodec()
		default:
			panic("unknown codec: " + c.Codec)
		}
	}

	return c.c
}

func (c Config) size() int {
	if c.MaxSize <= 0 {
		return 128
	}

	// When there's spread, randomly calculate a number in between.
	if c.MaxSize > c.MinSize {
		return rand.Intn(c.MaxSize-c.MinSize) + c.MinSize
	}

	return c.MaxSize
}
