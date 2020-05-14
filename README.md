comp-test
=========

Proof of concept to test alternative fixes for github.com/segmentio/kafka-go/zstd's leaky impl.

Uses strings made up of UUIDs as data to compress (not great for compression).

Usage:
```
% go run cmd/stressor/main.go --help
Usage: main

Flags:
  --help             Show context-sensitive help.
  --codec="zstd"     Codec implementation to use. Values:
                       zstd (default, leaks goroutines),
                       zstdold (DataDog, C impl),
                       zstd-fixed (same as zstd, but uses finalizers to avoid leak),
                       zstd-chanpool (swaps sync.Pool with channel-based pool)
  --producers=100    Number of producers.
  --consumers=100    Number of consumers.
  --interval=10ms    Data generation interval.
  --min_size=2048    Min size of data to compress.
  --max_size=2048    Max size of data to compress. When max > min, size is randomly picked for each block.
```

## Results

**Benchmark machine:**
- MacBook Pro (15-inch, 2018)
- 2.9 GHz 6-Core Intel Core i9
- 32 GB 2400 MHz DDR4

**Program arguments:**
* `--producers=100`
* `--consumers=100`
* `--interval=10ms`
* `--min_size=2048`
* `--max_size=2048`

(i.e. defaults)

#### zstd
Leaks goroutines, performance degrades over time.

* Throughput vol: ~1GB/s
* Compression ratio: 1.60-1.70
* Goroutines: 1600++
* Memory: 4.5GB++

#### zstd-fixed
Basically the same impl as zstd, except it pools `*reader` (not `*zstdlib.Decoder`) and users finalizers to close unreachable decoders.
Performance holds steady over time.

* Throughput vol: ~1GB/s
* Compression ratio: 1.60-1.70
* Goroutines: 1600-1700
* Memory: 4.5-6.5GB

#### zstd-chanpool
Similar impl to zstd, but uses channel-based pool instead of sync.Pool.
This pool never downsizes and holds strong references to decoders, so they're always reachable (i.e. no leaks).

* Throughput vol: ~900MB/s
* Compression ratio: 1.60-1.70
* Goroutines: 1600-1700
* Memory: 4.5-6.5GB

#### zstd-old
Previous version of github.com/segmentio/kafka-go/zstd, uses DataDog's wrapper for C zstdlib.
By far the fastest, with lowest footprint.

* Throughput vol: 1.4-1.5GB/s
* Compression ratio: 1.86
* Goroutines: 205
* Memory: <10MB
