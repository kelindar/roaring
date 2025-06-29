package main

import (
	"bytes"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/RoaringBitmap/roaring"
	"github.com/kelindar/bench"
	rb "github.com/kelindar/roaring"
)

var (
	sizes = []int{1e3, 1e6}
)

func main() {
	bench.Run(func(runner *bench.B) {
		runOps(runner)
		runMath(runner)
		runRange(runner)
		runCodec(runner)
	}, bench.WithReference(),
		bench.WithDuration(10*time.Millisecond),
		bench.WithSamples(100),
	)
}

func runOps(b *bench.B) {
	operations := []struct {
		name  string
		ourFn func(*rb.Bitmap, uint32)
		refFn func(*roaring.Bitmap, uint32)
	}{
		{"set", (*rb.Bitmap).Set, (*roaring.Bitmap).Add},
		{"has", func(bm *rb.Bitmap, v uint32) { bm.Contains(v) }, func(bm *roaring.Bitmap, v uint32) { bm.Contains(v) }},
		{"del", (*rb.Bitmap).Remove, (*roaring.Bitmap).Remove},
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
		for _, size := range sizes {
			for _, shape := range shapes {
				data := shape.gen(size)
				our, ref := randomBitmaps(data)

				name := fmt.Sprintf("%s %s (%s) ", op.name, formatSize(size), shape.name)
				b.Run(name,
					func(i int) { op.ourFn(our, data[i%len(data)]) },
					func(i int) { op.refFn(ref, data[i%len(data)]) })
			}
		}
	}
}

func runMath(b *bench.B) {
	operations := []struct {
		name  string
		ourFn func(*rb.Bitmap, *rb.Bitmap)
		refFn func(*roaring.Bitmap, *roaring.Bitmap)
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
		for _, size := range sizes {
			for _, shape := range shapes {
				data := shape.gen(size)
				our, ref := randomBitmaps(data)
				ourSrc, refSrc := randomBitmaps(data)
				our.Optimize()
				ref.RunOptimize()
				ourSrc.Optimize()
				refSrc.RunOptimize()

				name := fmt.Sprintf("%s %s (%s) ", op.name, formatSize(size), shape.name)
				b.Run(name,
					func(_ int) {
						dst := our.Clone(nil)
						op.ourFn(dst, ourSrc)
					},
					func(_ int) {
						dst := ref.Clone()
						op.refFn(dst, refSrc)
					})
			}
		}
	}
}

func runRange(b *bench.B) {
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
			our, ref := randomBitmaps(data)

			name := fmt.Sprintf("range %s (%s) ", formatSize(size), shape.name)

			b.Run(name,
				func(op int) {
					our.Range(func(uint32) bool { return true })
				},
				func(op int) {
					ref.Iterate(func(uint32) bool { return true })
				})
		}
	}
}

func formatSize(size int) string {
	if size >= 1e6 {
		return fmt.Sprintf("%.0fM", float64(size)/1e6)
	}
	return fmt.Sprintf("%.0fK", float64(size)/1e3)
}

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
func randomBitmaps(data []uint32) (*rb.Bitmap, *roaring.Bitmap) {
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

// Benchmark codec (WriteTo + ReadFrom) for 100K bitmaps with different shapes
func runCodec(b *bench.B) {
	const size = 100_000
	shapes := []struct {
		name string
		gen  func(size int) []uint32
	}{
		{"seq", dataSeq},
		{"rnd", dataRand},
		{"sps", dataSparse},
		{"dns", dataDense},
	}

	for _, shape := range shapes {
		data := shape.gen(size)
		bm := rb.New()
		for _, v := range data {
			bm.Set(v)
		}

		b.Run("write "+shape.name, func(_ int) {
			var buf bytes.Buffer
			_, _ = bm.WriteTo(&buf)
		})

		encoded := bm.ToBytes()
		b.Run("read "+shape.name, func(_ int) {
			bm2 := rb.New()
			_, _ = bm2.ReadFrom(bytes.NewReader(encoded))
		})
	}
}
