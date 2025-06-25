package roaring

import (
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/RoaringBitmap/roaring"
)

func BenchmarkOps(b *testing.B) {
	benchAll(b, "set", func(rb *Bitmap, v uint32) {
		rb.Set(v)
	}, func(rb *roaring.Bitmap, v uint32) {
		rb.Add(v)
	})
	benchAll(b, "has", func(rb *Bitmap, v uint32) {
		rb.Contains(v)
	}, func(rb *roaring.Bitmap, v uint32) {
		rb.Contains(v)
	})
	benchAll(b, "del", func(rb *Bitmap, v uint32) {
		rb.Remove(v)
	}, func(rb *roaring.Bitmap, v uint32) {
		rb.Remove(v)
	})
}

func BenchmarkRange(b *testing.B) {
	for _, size := range []int{1000, 1000000} {
		for _, shape := range []fnShape{dataSeq(size, 0), dataRand(size, uint32(size)), dataSparse(size), dataDense(size)} {
			benchRange(b, fmt.Sprintf("rng-%d", size), shape)
		}
	}
}

func BenchmarkAnd(b *testing.B) {
	for _, size := range []int{1000, 1000000} {
		for _, shape := range []fnShape{dataSeq(size, 0), dataRand(size, uint32(size)), dataSparse(size), dataDense(size)} {
			benchAnd(b, fmt.Sprintf("and-%d", size), shape)
		}
	}
}

func BenchmarkClone(b *testing.B) {
	data, _ := dataRand(1e6, 1e6)()
	rb, _ := random(data)
	rb.Optimize()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		clone := rb.Clone(nil)
		_ = clone
	}
}

// ---------------------------------------- Benchmarking ----------------------------------------

// benchRange runs a benchmark for the Range operation
func benchRange(b *testing.B, name string, gen fnShape) {
	data, shape := gen()
	our, ref := random(data)

	b.Run(fmt.Sprintf("%s-%s", name, shape), func(b *testing.B) {
		// Measure reference implementation speed using Iterate
		start := time.Now()
		refIterations := 0
		for time.Since(start) < time.Second {
			ref.Iterate(func(uint32) bool { return true })
			refIterations++
		}
		refTime := time.Since(start)
		f0 := float64(refIterations) / refTime.Seconds()

		// Measure our implementation speed
		b.ResetTimer()
		b.ReportAllocs()
		start = time.Now()
		ourIterations := 0
		for time.Since(start) < time.Second {
			our.Range(func(uint32) {})
			ourIterations++
		}
		ourTime := time.Since(start)
		f1 := float64(ourIterations) / ourTime.Seconds()

		b.ReportMetric(1e9/(f1*float64(our.Count())), "ns/op") // Per element
		b.ReportMetric(f1*float64(our.Count())/1e6, "M/s")     // Elements per second
		b.ReportMetric(f1/f0*100, "%")                         // Speedup
	})
}

func benchAll(b *testing.B, name string, fn func(rb *Bitmap, v uint32), fnRef func(rb *roaring.Bitmap, v uint32)) {
	for _, size := range []int{1000, 1000000} {
		for _, shape := range []fnShape{dataSeq(size, 0), dataRand(size, uint32(size)), dataSparse(size), dataDense(size)} {
			bench(b, fmt.Sprintf("%s-%d", name, size), shape, fn, fnRef)
		}
	}
}

// bench runs a benchmark for a given generator and function
func bench(b *testing.B, name string, gen fnShape, fnOur func(rb *Bitmap, v uint32), fnRef func(rb *roaring.Bitmap, v uint32)) {
	data, shape := gen()
	our, ref := random(data)
	b.Run(fmt.Sprintf("%s-%s", name, shape), func(b *testing.B) {
		f0 := loopFor(time.Second, data, func(v uint32) {
			fnRef(ref, v)
		})

		b.ResetTimer()
		b.ReportAllocs()
		f1 := loopFor(time.Second, data, func(v uint32) {
			fnOur(our, v)
		})

		b.ReportMetric(1e9/f1, "ns/op")
		b.ReportMetric(f1/1e6, "M/s")  // Througput
		b.ReportMetric(f1/f0*100, "%") // Speedup
	})
}

func loopFor(interval time.Duration, data []uint32, fn func(v uint32)) float64 {
	start, ops := time.Now(), float64(0)
	for time.Since(start) < interval {
		for _, v := range data {
			fn(v)
			ops++
		}
	}
	return float64(ops) / time.Since(start).Seconds()
}

// ---------------------------------------- Generators ----------------------------------------

// random creates a bitmap with 50% of the values set
func random(data []uint32) (*Bitmap, *roaring.Bitmap) {
	out := New()
	ref := roaring.NewBitmap()
	for _, v := range data {
		if rand.IntN(2) == 0 {
			out.Set(v)
			ref.Add(v)
		}
	}
	return out, ref
}

type fnShape = func() ([]uint32, string)

// dataSeq creates consecutive integers starting from offset
func dataSeq(size int, offset uint32) fnShape {
	return func() ([]uint32, string) {
		data := make([]uint32, size)
		for i := 0; i < size; i++ {
			data[i] = offset + uint32(i)
		}
		return data, "seq"
	}
}

// dataRand creates random integers within a range
func dataRand(size int, maxVal uint32) fnShape {
	return func() ([]uint32, string) {
		data := make([]uint32, size)
		for i := 0; i < size; i++ {
			data[i] = uint32(rand.IntN(int(maxVal)))
		}
		return data, "rnd"
	}
}

// dataSparse creates sparse integers (large gaps between values)
func dataSparse(size int) fnShape {
	return func() ([]uint32, string) {
		data := make([]uint32, size)
		for i := 0; i < size; i++ {
			data[i] = uint32(i * 1000)
		}
		return data, "sps"
	}
}

// dataDense creates dense integers in a small range
func dataDense(size int) fnShape {
	return func() ([]uint32, string) {
		data := make([]uint32, size)
		for i := 0; i < size; i++ {
			data[i] = uint32(rand.IntN(size / 10))
		}
		return data, "dns"
	}
}

// benchAnd runs a benchmark for the And operation
func benchAnd(b *testing.B, name string, gen fnShape) {
	data, shape := gen()
	our1, ref1 := random(data)
	our2, ref2 := random(data)

	b.Run(fmt.Sprintf("%s-%s", name, shape), func(b *testing.B) {
		// Measure reference implementation speed
		start := time.Now()
		refIterations := 0
		for time.Since(start) < time.Second {
			refClone1 := ref1.Clone()
			refClone2 := ref2.Clone()
			refClone1.And(refClone2)
			refIterations++
		}
		refTime := time.Since(start)
		f0 := float64(refIterations) / refTime.Seconds()

		// Measure our implementation speed
		b.ResetTimer()
		b.ReportAllocs()
		start = time.Now()

		ourIterations := 0
		for time.Since(start) < time.Second {
			ourClone1 := our1.Clone(nil)
			ourClone2 := our2.Clone(nil)
			ourClone1.And(ourClone2)
			ourIterations++
		}
		ourTime := time.Since(start)
		f1 := float64(ourIterations) / ourTime.Seconds()

		b.N = ourIterations
		b.ReportMetric(f1/1e6, "M/s")  // Operations per second (in millions)
		b.ReportMetric(f1/f0*100, "%") // Speedup ratio
	})
}
