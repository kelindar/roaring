package main

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/RoaringBitmap/roaring"
	rb "github.com/kelindar/roaring"
)

const (
	// Table formatting constants
	tableFormat = "%-16s %-6s %-12s %-12s %-12s\n"
	headerSep   = "------------"
)

var (
	sizes = []int{1e3, 1e7}
)

func main() {
	prefix := flag.String("bench", "", "Run only benchmarks with this prefix (e.g. 'set', 'and', 'range')")
	flag.Parse()

	runner := &BenchRunner{
		prefix: *prefix,
	}

	runner.printHeader()
	runner.runOps()
	runner.runMath()
	runner.runRange()
}

type BenchRunner struct {
	prefix string
}

func (br *BenchRunner) printHeader() {
	fmt.Printf(tableFormat, "name", "size", "time/op", "ops/s", "speedup")
	fmt.Printf(tableFormat, "----------------", "------", headerSep, headerSep, headerSep)
}

func (br *BenchRunner) shouldRun(name string) bool {
	if br.prefix == "" {
		return true
	}
	return strings.HasPrefix(name, br.prefix)
}

func (br *BenchRunner) runOps() {
	operations := []struct {
		name  string
		ourFn func(rb *rb.Bitmap, v uint32)
		refFn func(rb *roaring.Bitmap, v uint32)
	}{
		{"set", func(rb *rb.Bitmap, v uint32) { rb.Set(v) }, func(rb *roaring.Bitmap, v uint32) { rb.Add(v) }},
		{"has", func(rb *rb.Bitmap, v uint32) { rb.Contains(v) }, func(rb *roaring.Bitmap, v uint32) { rb.Contains(v) }},
		{"del", func(rb *rb.Bitmap, v uint32) { rb.Remove(v) }, func(rb *roaring.Bitmap, v uint32) { rb.Remove(v) }},
	}

	shapes := []struct {
		name string
		gen  func(size int) []uint32
	}{
		{"seq", dataSeq},
		{"rnd", dataRand},
		{"sps", dataSparse},
		{"dns", dataDense},
	}

	for _, op := range operations {
		if !br.shouldRun(op.name) {
			continue
		}

		for _, size := range sizes {
			for _, shape := range shapes {
				data := shape.gen(size)
				our, ref := br.randomBitmaps(data)

				// Measure reference performance
				refOps := br.benchOp(data, func(v uint32) { op.refFn(ref, v) })

				// Measure our performance
				ourOps := br.benchOp(data, func(v uint32) { op.ourFn(our, v) })

				// Calculate metrics
				nsPerOp := 1e9 / ourOps
				speedup := ourOps / refOps

				fmt.Printf(tableFormat,
					fmt.Sprintf("%s (%s)", op.name, shape.name), br.formatSize(size),
					br.formatTime(nsPerOp), fmt.Sprintf("%.1fM", ourOps/1e6), br.formatSpeedup(speedup))
			}
		}
	}
}

func (br *BenchRunner) runMath() {
	operations := []struct {
		name  string
		ourFn func(dst, src *rb.Bitmap)
		refFn func(dst, src *roaring.Bitmap)
	}{
		{"and", func(dst, src *rb.Bitmap) { dst.And(src) }, func(dst, src *roaring.Bitmap) { dst.And(src) }},
		{"or", func(dst, src *rb.Bitmap) { dst.Or(src) }, func(dst, src *roaring.Bitmap) { dst.Or(src) }},
		{"xor", func(dst, src *rb.Bitmap) { dst.Xor(src) }, func(dst, src *roaring.Bitmap) { dst.Xor(src) }},
		{"andnot", func(dst, src *rb.Bitmap) { dst.AndNot(src) }, func(dst, src *roaring.Bitmap) { dst.AndNot(src) }},
	}

	shapes := []struct {
		name string
		gen  func(size int) []uint32
	}{
		{"seq", dataSeq},
		{"rnd", dataRand},
		{"sps", dataSparse},
		{"dns", dataDense},
	}

	for _, op := range operations {
		if !br.shouldRun(op.name) {
			continue
		}

		for _, size := range sizes {
			for _, shape := range shapes {
				data := shape.gen(size)
				our, ref := br.randomBitmaps(data)
				our.Optimize()
				ref.RunOptimize()

				// Measure reference performance
				refOps := br.benchMathOpRef(ref, op.refFn)

				// Measure our performance
				ourOps := br.benchMathOpOur(our, op.ourFn)

				// Calculate metrics
				nsPerOp := 1e9 / ourOps
				speedup := ourOps / refOps

				fmt.Printf(tableFormat,
					fmt.Sprintf("%s (%s)", op.name, shape.name), br.formatSize(size),
					br.formatTime(nsPerOp), fmt.Sprintf("%.1fM", ourOps/1e6), br.formatSpeedup(speedup))
			}
		}
	}
}

func (br *BenchRunner) runRange() {
	if !br.shouldRun("range") {
		return
	}

	shapes := []struct {
		name string
		gen  func(size int) []uint32
	}{
		{"seq", dataSeq},
		{"rnd", dataRand},
		{"sps", dataSparse},
		{"dns", dataDense},
	}

	for _, size := range sizes {
		for _, shape := range shapes {
			data := shape.gen(size)
			our, ref := br.randomBitmaps(data)

			// Measure reference performance using Iterate
			refOps := br.benchRange(func() { ref.Iterate(func(uint32) bool { return true }) })

			// Measure our performance using Range
			ourOps := br.benchRange(func() { our.Range(func(uint32) {}) })

			// Calculate metrics
			nsPerOp := 1e9 / (ourOps * float64(our.Count()))
			speedup := ourOps / refOps

			fmt.Printf(tableFormat,
				fmt.Sprintf("range (%s)", shape.name), br.formatSize(size),
				br.formatTime(nsPerOp), fmt.Sprintf("%.1fM", ourOps*float64(our.Count())/1e6), br.formatSpeedup(speedup))
		}
	}
}

// Helper functions for benchmarking

func (br *BenchRunner) formatSpeedup(speedup float64) string {
	percentage := speedup * 100
	if percentage >= 100 {
		return fmt.Sprintf("✅ %6.1f%%", percentage)
	}
	return fmt.Sprintf("❌ %6.1f%%", percentage)
}

func (br *BenchRunner) formatSize(size int) string {
	if size >= 1000000 {
		return fmt.Sprintf("%.0fM", float64(size)/1000000)
	}
	if size >= 1000 {
		return fmt.Sprintf("%.0fK", float64(size)/1000)
	}
	return fmt.Sprintf("%d", size)
}

func (br *BenchRunner) formatTime(nsPerOp float64) string {
	if nsPerOp >= 1000000 {
		return fmt.Sprintf("%.1fms", nsPerOp/1000000)
	}
	return fmt.Sprintf("%.1fns", nsPerOp)
}

func (br *BenchRunner) benchOp(data []uint32, fn func(uint32)) float64 {
	start := time.Now()
	ops := 0
	for time.Since(start) < time.Second {
		for _, v := range data {
			fn(v)
			ops++
		}
	}
	return float64(ops) / time.Since(start).Seconds()
}

func (br *BenchRunner) benchMathOpOur(bm *rb.Bitmap, fn func(dst, src *rb.Bitmap)) float64 {
	start := time.Now()
	ops := 0
	for time.Since(start) < time.Second {
		clone := bm.Clone(nil)
		fn(clone, bm)
		ops++
	}
	return float64(ops) / time.Since(start).Seconds()
}

func (br *BenchRunner) benchMathOpRef(bm *roaring.Bitmap, fn func(dst, src *roaring.Bitmap)) float64 {
	start := time.Now()
	ops := 0
	for time.Since(start) < time.Second {
		clone := bm.Clone()
		fn(clone, bm)
		ops++
	}
	return float64(ops) / time.Since(start).Seconds()
}

func (br *BenchRunner) benchRange(fn func()) float64 {
	start := time.Now()
	ops := 0
	for time.Since(start) < time.Second {
		fn()
		ops++
	}
	return float64(ops) / time.Since(start).Seconds()
}

// Data generators

func dataSeq(size int) []uint32 {
	data := make([]uint32, size)
	for i := 0; i < size; i++ {
		data[i] = uint32(i)
	}
	return data
}

func dataRand(size int) []uint32 {
	data := make([]uint32, size)
	maxVal := uint32(size)
	for i := 0; i < size; i++ {
		data[i] = uint32(rand.IntN(int(maxVal)))
	}
	return data
}

func dataSparse(size int) []uint32 {
	data := make([]uint32, size)
	for i := 0; i < size; i++ {
		data[i] = uint32(i * 1000)
	}
	return data
}

func dataDense(size int) []uint32 {
	data := make([]uint32, size)
	for i := 0; i < size; i++ {
		data[i] = uint32(rand.IntN(size / 10))
	}
	return data
}

// randomBitmaps creates bitmaps with 50% of the values set
func (br *BenchRunner) randomBitmaps(data []uint32) (*rb.Bitmap, *roaring.Bitmap) {
	our := rb.New()
	ref := roaring.NewBitmap()
	for _, v := range data {
		if rand.IntN(2) == 0 {
			our.Set(v)
			ref.Add(v)
		}
	}
	return our, ref
}
