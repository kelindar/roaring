package main

import (
	"flag"
	"fmt"
	"math/rand/v2"

	"github.com/RoaringBitmap/roaring"
	rb "github.com/kelindar/roaring"
	"github.com/kelindar/roaring/tinybench"
)

var (
	sizes = []int{1e3, 1e7}
)

func main() {
	prefix := flag.String("bench", "", "Run only benchmarks with this prefix (e.g. 'set', 'and', 'range')")
	flag.Parse()

	runner := tinybench.New("bench.json").Filter(*prefix).Start()

	runOps(runner)
	runMath(runner)
	runRange(runner)

	runner.Finish()
}

func runOps(runner *tinybench.Runner) {
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
				runner.Run(name,
					func() { op.ourFn(our, data[rand.IntN(len(data))]) },
					func() { op.refFn(ref, data[rand.IntN(len(data))]) })
			}
		}
	}
}

func runMath(runner *tinybench.Runner) {
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
				runner.Run(name,
					func() {
						dst := our.Clone(nil)
						op.ourFn(dst, ourSrc)
					},
					func() {
						dst := ref.Clone()
						op.refFn(dst, refSrc)
					})
			}
		}
	}
}

func runRange(runner *tinybench.Runner) {
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

			runner.Run(name,
				func() { our.Range(func(uint32) {}) },
				func() { ref.Iterate(func(uint32) bool { return true }) })
		}
	}
}

func formatSize(size int) string {
	if size >= 1e6 {
		return fmt.Sprintf("%.1fM", float64(size)/1e6)
	}
	return fmt.Sprintf("%.1fK", float64(size)/1e3)
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
