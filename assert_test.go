package roaring

import (
	"math/rand/v2"
	"testing"

	"github.com/kelindar/bitmap"
	"github.com/stretchr/testify/assert"
)

func newArr(data ...uint32) *container {
	return newContainer(typeArray, data...)
}

func newBmp(data ...uint32) *container {
	return newContainer(typeBitmap, data...)
}

func newRun(data ...uint32) *container {
	return newContainer(typeRun, data...)
}

func newContainer(typ ctype, data ...uint32) *container {
	c := &container{
		Type: typ,
		Data: make([]uint16, 0, len(data)),
	}

	for _, v := range data {
		switch c.Type {
		case typeArray:
			c.arrSet(uint16(v))
		case typeBitmap:
			c.bmpSet(uint16(v))
		case typeRun:
			c.runSet(uint16(v))
		}
	}
	return c
}

// ---------------------------------------- Test Helpers ----------------------------------------

// testPair creates both our bitmap and reference bitmap with same data
func testPair(data []uint32) (*Bitmap, *bitmap.Bitmap) {
	our := New()
	var ref bitmap.Bitmap
	for _, v := range data {
		our.Set(v)
		ref.Set(v)
	}
	return our, &ref
}

// testPairRandom creates bitmaps with 50% of values set randomly
func testPairRandom(data []uint32) (*Bitmap, *bitmap.Bitmap) {
	our := New()
	var ref bitmap.Bitmap
	for _, v := range data {
		if rand.IntN(2) == 0 {
			our.Set(v)
			ref.Set(v)
		}
	}
	return our, &ref
}

// assertEqualBitmaps compares our bitmap with reference bitmap
func assertEqualBitmaps(t *testing.T, our *Bitmap, ref *bitmap.Bitmap) {
	assert.Equal(t, ref.Count(), our.Count(), "Count mismatch")

	// Compare all values
	var ourValues, refValues []uint32
	our.Range(func(x uint32) { ourValues = append(ourValues, x) })
	ref.Range(func(x uint32) { refValues = append(refValues, x) })
	assert.Equal(t, refValues, ourValues, "Range mismatch")

	// Check individual contains calls
	ref.Range(func(x uint32) {
		assert.True(t, our.Contains(x), "Contains mismatch for %d", x)
	})
}

// changeType creates bitmap that forces specific container types
func changeType(ctype ctype) (*Bitmap, []uint32) {
	our := New()
	var values []uint32

	switch ctype {
	case typeArray:
		// Few sparse values to stay as array
		values = []uint32{1, 5, 10, 100, 500, 1000}
		for _, v := range values {
			our.Set(v)
		}
	case typeBitmap:
		// Many sparse values to become bitmap
		for i := 0; i < 5000; i++ {
			v := uint32(i * 3) // Sparse to prevent run optimization
			our.Set(v)
			values = append(values, v)
		}
	case typeRun:
		// Consecutive values + optimize to become run
		for i := 1000; i <= 2000; i++ {
			v := uint32(i)
			our.Set(v)
			values = append(values, v)
		}
		our.Optimize()
	}
	return our, values
}

// ---------------------------------------- Data Generators ----------------------------------------

type dataGen = func() ([]uint32, string)

// genSeq creates consecutive integers starting from offset
func genSeq(size int, offset uint32) dataGen {
	return func() ([]uint32, string) {
		data := make([]uint32, size)
		for i := 0; i < size; i++ {
			data[i] = offset + uint32(i)
		}
		return data, "seq"
	}
}

// genRand creates random integers within a range
func genRand(size int, maxVal uint32) dataGen {
	return func() ([]uint32, string) {
		data := make([]uint32, size)
		for i := 0; i < size; i++ {
			data[i] = uint32(rand.IntN(int(maxVal)))
		}
		return data, "rnd"
	}
}

// genSparse creates sparse integers (large gaps)
func genSparse(size int) dataGen {
	return func() ([]uint32, string) {
		data := make([]uint32, size)
		for i := 0; i < size; i++ {
			data[i] = uint32(i * 1000)
		}
		return data, "sps"
	}
}

// genDense creates dense integers in small range
func genDense(size int) dataGen {
	return func() ([]uint32, string) {
		data := make([]uint32, size)
		for i := 0; i < size; i++ {
			data[i] = uint32(rand.IntN(size / 10))
		}
		return data, "dns"
	}
}

// genBoundary creates boundary/edge case values
func genBoundary() dataGen {
	return func() ([]uint32, string) {
		data := []uint32{0, 65535, 65536, 131071, 131072, 4294967295}
		return data, "bnd"
	}
}

// genMixed creates values across multiple containers
func genMixed() dataGen {
	return func() ([]uint32, string) {
		var data []uint32
		// Container 0: array values
		data = append(data, 1, 5, 10, 100, 500, 1000)
		// Container 1: bitmap values
		for i := 0; i < 1000; i++ {
			data = append(data, uint32(65536+i*3))
		}
		// Container 2: run values
		for i := 131072; i <= 131172; i++ {
			data = append(data, uint32(i))
		}
		return data, "mix"
	}
}
