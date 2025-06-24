package roaring

import (
	"fmt"
	"math/rand/v2"
	"testing"
	"time"
)

func BenchmarkOps(b *testing.B) {
	benchAll(b, "set", func(rb *Bitmap, v uint32) {
		rb.Set(v)
	}, empty(), full())
	benchAll(b, "has", func(rb *Bitmap, v uint32) {
		rb.Contains(v)
	}, empty(), full())
	benchAll(b, "del", func(rb *Bitmap, v uint32) {
		rb.Remove(v)
	}, empty(), full())
}

// ---------------------------------------- Benchmarking ----------------------------------------

func benchAll(b *testing.B, name string, fn func(rb *Bitmap, v uint32), states ...fnState) {
	for _, size := range []int{1000, 1000000} {
		for _, shape := range []fnShape{dataSeq(size, 0), dataRand(size, uint32(size)), dataSparse(size), dataDense(size)} {
			for _, state := range states {
				bench(b, fmt.Sprintf("%s-%d", name, size), shape, state, fn)
			}
		}
	}
}

// bench runs a benchmark for a given generator and function
func bench(b *testing.B, name string, gen fnShape, setup fnState, fn func(rb *Bitmap, v uint32)) {
	data, shape := gen()
	bitmap, state := setup(data)
	b.Run(fmt.Sprintf("%s-%s-%s", name, shape, state), func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		start := time.Now()
		count := 0
		for i := 0; i < b.N; i++ {
			for _, v := range data {
				fn(bitmap, v)
				count++
			}
		}

		b.ReportMetric(float64(count/1e6)/time.Since(start).Seconds(), "M/s")
	})
}

// ---------------------------------------- Generators ----------------------------------------

type fnState = func(data []uint32) (*Bitmap, string)

func empty() fnState {
	return func(data []uint32) (*Bitmap, string) {
		return New(), "new"
	}
}

func full() func(data []uint32) (*Bitmap, string) {
	return func(data []uint32) (*Bitmap, string) {
		rb := New()
		for _, v := range data {
			rb.Set(v)
		}
		return rb, "ful"
	}
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
