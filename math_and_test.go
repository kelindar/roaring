// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package roaring

import (
	"math/rand/v2"
	"sort"
	"testing"

	"github.com/kelindar/bitmap"
	"github.com/stretchr/testify/assert"
)

func bitmapWith(c *container) (*Bitmap, []uint16) {
	v := New()
	v.ctrAdd(0, 0, c)
	return v, valuesOf(v)
}

func valuesOf(v *Bitmap) []uint16 {
	out := []uint16{}
	v.Range(func(x uint32) bool {
		out = append(out, uint16(x))
		return true
	})
	return out
}

func values32(v *Bitmap) []uint32 {
	out := []uint32{}
	v.Range(func(x uint32) bool {
		out = append(out, x)
		return true
	})
	return out
}

func bitmapOf(data ...uint32) *Bitmap {
	v := New()
	for _, x := range data {
		v.Set(x)
	}
	return v
}

func containerTailCleared(v *Bitmap) bool {
	tail := v.containers[len(v.containers):cap(v.containers)]
	for _, c := range tail {
		if c.Data != nil || c.Size != 0 || c.Shared || c.Call != 0 || c.Type != typeArray {
			return false
		}
	}
	return true
}

func newArr(data ...uint32) *container {
	return newContainer(typeArray, data...)
}

func newRun(data ...uint32) *container {
	return newContainer(typeRun, data...)
}

func newBmp(data ...uint32) *container {
	return newContainer(typeBitmap, data...)
}

// newBmpPermutations creates a Bitmap with all 16 4-bit permutations
func newBmpPermutations() *container {
	rb := newBmp()
	for perm := 0; perm < 16; perm++ {
		offset := perm * 4
		for bit := 0; bit < 4; bit++ {
			if (perm>>bit)&1 == 1 {
				rb.bmpSet(uint16(offset + bit))
			}
		}
	}
	return rb
}

func newContainer(typ ctype, data ...uint32) *container {
	c := &container{
		Type: typeArray,
		Data: make([]uint16, 0, len(data)),
	}

	for _, v := range data {
		c.arrSet(uint16(v))
	}

	switch typ {
	case typeBitmap:
		c.arrToBmp()
	case typeRun:
		arrToRun(c) // force
	}
	return c
}

// arrToRun attempts to convert array to run in a single pass
func arrToRun(c *container) {
	c.Type = typeRun
	if len(c.Data) == 0 {
		return
	}

	runsData := make([]uint16, 0, len(c.Data)/2)
	i0 := c.Data[0]
	i1 := c.Data[0]

	for i := 1; i < len(c.Data); i++ {
		if c.Data[i] == i1+1 {
			i1 = c.Data[i]
		} else {
			runsData = append(runsData, i0, i1)
			i0 = c.Data[i]
			i1 = c.Data[i]
		}
	}

	c.Data = append(runsData, i0, i1)
}

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

// changeType creates bitmap that forces specific container types
func changeType(ctype ctype) (*Bitmap, []uint32) {
	our := New()
	var values []uint32

	switch ctype {
	case typeArray:
		values = []uint32{1, 5, 10, 100, 500, 1000}
		for _, v := range values {
			our.Set(v)
		}
	case typeBitmap:
		for i := 0; i < 5000; i++ {
			v := uint32(i * 3)
			our.Set(v)
			values = append(values, v)
		}
	case typeRun:
		for i := 1000; i <= 2000; i++ {
			v := uint32(i)
			our.Set(v)
			values = append(values, v)
		}
		our.Optimize()
	}
	return our, values
}

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
		data = append(data, 1, 5, 10, 100, 500, 1000)
		for i := 0; i < 1000; i++ {
			data = append(data, uint32(65536+i*3))
		}
		for i := 131072; i <= 131172; i++ {
			data = append(data, uint32(i))
		}
		return data, "mix"
	}
}

func TestAnd(t *testing.T) {
	tc := []struct {
		name   string
		c1     *container
		c2     *container
		result []uint16
	}{
		{"empty", newArr(), newArr(), []uint16{}},
		{"arr ∧ arr", newArr(1, 2, 3), newArr(1, 2, 3), []uint16{1, 2, 3}},
		{"arr ∧ bmp", newArr(1, 2, 3), newBmp(1, 2, 3), []uint16{1, 2, 3}},
		{"arr ∧ run", newArr(1, 2, 3), newRun(1, 2, 3), []uint16{1, 2, 3}},
		{"bmp ∧ arr", newBmp(1, 2, 3), newArr(1, 2, 3), []uint16{1, 2, 3}},
		{"bmp ∧ bmp", newBmp(1, 2, 3), newBmp(1, 2, 3), []uint16{1, 2, 3}},
		{"bmp ∧ run", newBmp(1, 2, 3), newRun(1, 2, 3), []uint16{1, 2, 3}},
		{"run ∧ arr", newRun(1, 2, 3), newArr(1, 2, 3), []uint16{1, 2, 3}},
		{"run ∧ bmp", newRun(1, 2, 3), newBmp(1, 2, 3), []uint16{1, 2, 3}},
		{"run ∧ run", newRun(1, 2, 3), newRun(1, 2, 3), []uint16{1, 2, 3}},

		// Partial intersections
		{"arr ∧ arr partial", newArr(1, 2, 3, 4), newArr(2, 3, 5, 6), []uint16{2, 3}},
		{"arr ∧ bmp partial", newArr(1, 2, 3, 4), newBmp(2, 3, 5, 6), []uint16{2, 3}},
		{"arr ∧ run partial", newArr(1, 2, 3, 4), newRun(2, 3, 5, 6), []uint16{2, 3}},
		{"bmp ∧ arr partial", newBmp(1, 2, 3, 4), newArr(2, 3, 5, 6), []uint16{2, 3}},
		{"bmp ∧ bmp partial", newBmp(1, 2, 3, 4), newBmp(2, 3, 5, 6), []uint16{2, 3}},
		{"bmp ∧ run partial", newBmp(1, 2, 3, 4), newRun(2, 3, 5, 6), []uint16{2, 3}},
		{"run ∧ arr partial", newRun(1, 2, 3, 4), newArr(2, 3, 5, 6), []uint16{2, 3}},
		{"run ∧ bmp partial", newRun(1, 2, 3, 4), newBmp(2, 3, 5, 6), []uint16{2, 3}},
		{"run ∧ run partial", newRun(1, 2, 3, 4), newRun(2, 3, 5, 6), []uint16{2, 3}},

		// No intersections
		{"arr ∧ arr empty", newArr(1, 2, 3), newArr(4, 5, 6), []uint16{}},
		{"arr ∧ bmp empty", newArr(1, 2, 3), newBmp(4, 5, 6), []uint16{}},
		{"arr ∧ run empty", newArr(1, 2, 3), newRun(4, 5, 6), []uint16{}},
		{"bmp ∧ arr empty", newBmp(1, 2, 3), newArr(4, 5, 6), []uint16{}},
		{"bmp ∧ bmp empty", newBmp(1, 2, 3), newBmp(4, 5, 6), []uint16{}},
		{"bmp ∧ run empty", newBmp(1, 2, 3), newRun(4, 5, 6), []uint16{}},
		{"run ∧ arr empty", newRun(1, 2, 3), newArr(4, 5, 6), []uint16{}},
		{"run ∧ bmp empty", newRun(1, 2, 3), newBmp(4, 5, 6), []uint16{}},
		{"run ∧ run empty", newRun(1, 2, 3), newRun(4, 5, 6), []uint16{}},

		// Single element intersections
		{"arr ∧ arr single", newArr(1, 2, 3), newArr(2, 4, 5), []uint16{2}},
		{"arr ∧ bmp single", newArr(1, 2, 3), newBmp(2, 4, 5), []uint16{2}},
		{"arr ∧ run single", newArr(1, 2, 3), newRun(2, 4, 5), []uint16{2}},
		{"bmp ∧ arr single", newBmp(1, 2, 3), newArr(2, 4, 5), []uint16{2}},
		{"bmp ∧ bmp single", newBmp(1, 2, 3), newBmp(2, 4, 5), []uint16{2}},
		{"bmp ∧ run single", newBmp(1, 2, 3), newRun(2, 4, 5), []uint16{2}},
		{"run ∧ arr single", newRun(1, 2, 3), newArr(2, 4, 5), []uint16{2}},
		{"run ∧ bmp single", newRun(1, 2, 3), newBmp(2, 4, 5), []uint16{2}},
		{"run ∧ run single", newRun(1, 2, 3), newRun(2, 4, 5), []uint16{2}},

		// Boundary values
		{"arr ∧ arr boundary", newArr(0, 1, 65535), newArr(0, 65535), []uint16{0, 65535}},
		{"arr ∧ bmp boundary", newArr(0, 1, 65535), newBmp(0, 65535), []uint16{0, 65535}},
		{"arr ∧ run boundary", newArr(0, 1, 65535), newRun(0, 65535), []uint16{0, 65535}},
		{"bmp ∧ arr boundary", newBmp(0, 1, 65535), newArr(0, 65535), []uint16{0, 65535}},
		{"bmp ∧ bmp boundary", newBmp(0, 1, 65535), newBmp(0, 65535), []uint16{0, 65535}},
		{"bmp ∧ run boundary", newBmp(0, 1, 65535), newRun(0, 65535), []uint16{0, 65535}},
		{"run ∧ arr boundary", newRun(0, 1, 65535), newArr(0, 65535), []uint16{0, 65535}},
		{"run ∧ bmp boundary", newRun(0, 1, 65535), newBmp(0, 65535), []uint16{0, 65535}},
		{"run ∧ run boundary", newRun(0, 1, 65535), newRun(0, 65535), []uint16{0, 65535}},

		// One side empty
		{"arr ∧ empty", newArr(1, 2, 3), newArr(), []uint16{}},
		{"bmp ∧ empty", newBmp(1, 2, 3), newArr(), []uint16{}},
		{"run ∧ empty", newRun(1, 2, 3), newArr(), []uint16{}},
		{"empty ∧ arr", newArr(), newArr(1, 2, 3), []uint16{}},
		{"empty ∧ bmp", newArr(), newBmp(1, 2, 3), []uint16{}},
		{"empty ∧ run", newArr(), newRun(1, 2, 3), []uint16{}},

		// Large ranges with runs
		{"run ∧ run ranges", newRun(1, 2, 3, 4, 5, 10, 11, 12), newRun(3, 4, 5, 6, 7, 11, 12, 13), []uint16{3, 4, 5, 11, 12}},
		{"arr ∧ run ranges", newArr(1, 2, 3, 4, 5, 10, 11, 12), newRun(3, 4, 5, 6, 7, 11, 12, 13), []uint16{3, 4, 5, 11, 12}},
		{"bmp ∧ run ranges", newBmp(1, 2, 3, 4, 5, 10, 11, 12), newRun(3, 4, 5, 6, 7, 11, 12, 13), []uint16{3, 4, 5, 11, 12}},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			a, _ := bitmapWith(tt.c1)
			b, bv := bitmapWith(tt.c2)

			a.And(b)

			// Assert the result is correct
			assert.Equal(t, tt.result, valuesOf(a))
			assert.Equal(t, bv, valuesOf(b))
		})
	}
}

func TestAndCompaction(t *testing.T) {
	a := New()
	a.ctrAdd(0, 0, newArr(1))
	a.ctrAdd(1, 1, newBmp(1, 2, 3))

	b := New()
	b.ctrAdd(1, 0, newArr(2, 3))

	a.And(b)

	assert.Equal(t, 2, a.Count())
	assert.False(t, a.Contains(1))
	assert.False(t, a.Contains(1<<16|1))
	assert.True(t, a.Contains(1<<16|2))
	assert.True(t, a.Contains(1<<16|3))
}

func TestAndNot(t *testing.T) {
	tc := []struct {
		name   string
		c1     *container
		c2     *container
		result []uint16
	}{
		{"empty", newArr(), newArr(), []uint16{}},
		{"arr ¬ arr", newArr(1, 2, 3), newArr(1, 2, 3), []uint16{}}, // remove all = empty
		{"arr ¬ bmp", newArr(1, 2, 3), newBmp(1, 2, 3), []uint16{}},
		{"arr ¬ run", newArr(1, 2, 3), newRun(1, 2, 3), []uint16{}},
		{"bmp ¬ arr", newBmp(1, 2, 3), newArr(1, 2, 3), []uint16{}},
		{"bmp ¬ bmp", newBmp(1, 2, 3), newBmp(1, 2, 3), []uint16{}},
		{"bmp ¬ run", newBmp(1, 2, 3), newRun(1, 2, 3), []uint16{}},
		{"run ¬ arr", newRun(1, 2, 3), newArr(1, 2, 3), []uint16{}},
		{"run ¬ bmp", newRun(1, 2, 3), newBmp(1, 2, 3), []uint16{}},
		{"run ¬ run", newRun(1, 2, 3), newRun(1, 2, 3), []uint16{}},

		// Disjoint sets (remove none = identity)
		{"arr ¬ arr disjoint", newArr(1, 2, 3), newArr(4, 5, 6), []uint16{1, 2, 3}},
		{"arr ¬ bmp disjoint", newArr(1, 2, 3), newBmp(4, 5, 6), []uint16{1, 2, 3}},
		{"arr ¬ run disjoint", newArr(1, 2, 3), newRun(4, 5, 6), []uint16{1, 2, 3}},
		{"bmp ¬ arr disjoint", newBmp(1, 2, 3), newArr(4, 5, 6), []uint16{1, 2, 3}},
		{"bmp ¬ bmp disjoint", newBmp(1, 2, 3), newBmp(4, 5, 6), []uint16{1, 2, 3}},
		{"bmp ¬ run disjoint", newBmp(1, 2, 3), newRun(4, 5, 6), []uint16{1, 2, 3}},
		{"run ¬ arr disjoint", newRun(1, 2, 3), newArr(4, 5, 6), []uint16{1, 2, 3}},
		{"run ¬ bmp disjoint", newRun(1, 2, 3), newBmp(4, 5, 6), []uint16{1, 2, 3}},
		{"run ¬ run disjoint", newRun(1, 2, 3), newRun(4, 5, 6), []uint16{1, 2, 3}},

		// Partial differences
		{"arr ¬ arr partial", newArr(1, 2, 3, 4), newArr(3, 4, 5, 6), []uint16{1, 2}},
		{"arr ¬ bmp partial", newArr(1, 2, 3, 4), newBmp(3, 4, 5, 6), []uint16{1, 2}},
		{"arr ¬ run partial", newArr(1, 2, 3, 4), newRun(3, 4, 5, 6), []uint16{1, 2}},
		{"bmp ¬ arr partial", newBmp(1, 2, 3, 4), newArr(3, 4, 5, 6), []uint16{1, 2}},
		{"bmp ¬ bmp partial", newBmp(1, 2, 3, 4), newBmp(3, 4, 5, 6), []uint16{1, 2}},
		{"bmp ¬ run partial", newBmp(1, 2, 3, 4), newRun(3, 4, 5, 6), []uint16{1, 2}},
		{"run ¬ arr partial", newRun(1, 2, 3, 4), newArr(3, 4, 5, 6), []uint16{1, 2}},
		{"run ¬ bmp partial", newRun(1, 2, 3, 4), newBmp(3, 4, 5, 6), []uint16{1, 2}},
		{"run ¬ run partial", newRun(1, 2, 3, 4), newRun(3, 4, 5, 6), []uint16{1, 2}},

		// Single element removals
		{"arr ¬ arr single", newArr(1, 2, 3), newArr(2), []uint16{1, 3}},
		{"arr ¬ bmp single", newArr(1, 2, 3), newBmp(2), []uint16{1, 3}},
		{"arr ¬ run single", newArr(1, 2, 3), newRun(2), []uint16{1, 3}},
		{"bmp ¬ arr single", newBmp(1, 2, 3), newArr(2), []uint16{1, 3}},
		{"bmp ¬ bmp single", newBmp(1, 2, 3), newBmp(2), []uint16{1, 3}},
		{"bmp ¬ run single", newBmp(1, 2, 3), newRun(2), []uint16{1, 3}},
		{"run ¬ arr single", newRun(1, 2, 3), newArr(2), []uint16{1, 3}},
		{"run ¬ bmp single", newRun(1, 2, 3), newBmp(2), []uint16{1, 3}},
		{"run ¬ run single", newRun(1, 2, 3), newRun(2), []uint16{1, 3}},

		// Boundary values
		{"arr ¬ arr boundary", newArr(0, 1, 65535), newArr(0, 65535), []uint16{1}},
		{"arr ¬ bmp boundary", newArr(0, 1, 65535), newBmp(0, 65535), []uint16{1}},
		{"arr ¬ run boundary", newArr(0, 1, 65535), newRun(0, 65535), []uint16{1}},
		{"bmp ¬ arr boundary", newBmp(0, 1, 65535), newArr(0, 65535), []uint16{1}},
		{"bmp ¬ bmp boundary", newBmp(0, 1, 65535), newBmp(0, 65535), []uint16{1}},
		{"bmp ¬ run boundary", newBmp(0, 1, 65535), newRun(0, 65535), []uint16{1}},
		{"run ¬ arr boundary", newRun(0, 1, 65535), newArr(0, 65535), []uint16{1}},
		{"run ¬ bmp boundary", newRun(0, 1, 65535), newBmp(0, 65535), []uint16{1}},
		{"run ¬ run boundary", newRun(0, 1, 65535), newRun(0, 65535), []uint16{1}},

		// Empty removals (remove nothing = identity)
		{"arr ¬ empty", newArr(1, 2, 3), newArr(), []uint16{1, 2, 3}},
		{"bmp ¬ empty", newBmp(1, 2, 3), newArr(), []uint16{1, 2, 3}},
		{"run ¬ empty", newRun(1, 2, 3), newArr(), []uint16{1, 2, 3}},

		// Remove from empty
		{"empty ¬ arr", newArr(), newArr(1, 2, 3), []uint16{}},
		{"empty ¬ bmp", newArr(), newBmp(1, 2, 3), []uint16{}},
		{"empty ¬ run", newArr(), newRun(1, 2, 3), []uint16{}},

		// Complex patterns
		{"arr ¬ run complex", newArr(1, 2, 3, 4, 5, 6, 7), newRun(2, 4, 6), []uint16{1, 3, 5, 7}},
		{"bmp ¬ run complex", newBmp(1, 2, 3, 4, 5, 6, 7), newRun(2, 4, 6), []uint16{1, 3, 5, 7}},
		{"run ¬ arr complex", newRun(1, 2, 3, 4, 5, 6, 7), newArr(2, 4, 6), []uint16{1, 3, 5, 7}},
		{"run ¬ run complex", newRun(1, 2, 3, 4, 5, 10, 11, 12), newRun(2, 4, 11), []uint16{1, 3, 5, 10, 12}},

		// Subset removals
		{"arr ¬ arr subset", newArr(1, 2, 3, 4, 5), newArr(2, 4), []uint16{1, 3, 5}},
		{"bmp ¬ arr subset", newBmp(1, 2, 3, 4, 5), newArr(2, 4), []uint16{1, 3, 5}},
		{"run ¬ arr subset", newRun(1, 2, 3, 4, 5), newArr(2, 4), []uint16{1, 3, 5}},

		// Superset removals (remove more than exists)
		{"arr ¬ arr superset", newArr(2, 4), newArr(1, 2, 3, 4, 5), []uint16{}},
		{"bmp ¬ arr superset", newBmp(2, 4), newArr(1, 2, 3, 4, 5), []uint16{}},
		{"run ¬ arr superset", newRun(2, 4), newArr(1, 2, 3, 4, 5), []uint16{}},

		// Edge cases with consecutive ranges
		{"run ¬ run range split", newRun(1, 2, 3, 4, 5, 6, 7, 8), newRun(3, 4, 5, 6), []uint16{1, 2, 7, 8}},
		{"run ¬ arr range split", newRun(1, 2, 3, 4, 5, 6, 7, 8), newArr(3, 4, 5, 6), []uint16{1, 2, 7, 8}},
		{"bmp ¬ run range", newBmp(1, 2, 3, 4, 5, 6, 7, 8), newRun(3, 4, 5, 6), []uint16{1, 2, 7, 8}},

		// Beginning/end removals
		{"arr ¬ arr beginning", newArr(1, 2, 3, 4, 5), newArr(1, 2), []uint16{3, 4, 5}},
		{"arr ¬ arr ending", newArr(1, 2, 3, 4, 5), newArr(4, 5), []uint16{1, 2, 3}},
		{"run ¬ run beginning", newRun(1, 2, 3, 4, 5), newRun(1, 2), []uint16{3, 4, 5}},
		{"run ¬ run ending", newRun(1, 2, 3, 4, 5), newRun(4, 5), []uint16{1, 2, 3}},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			a, _ := bitmapWith(tt.c1)
			b, bv := bitmapWith(tt.c2)

			a.AndNot(b)

			// Assert the result is correct
			assert.Equal(t, tt.result, valuesOf(a))
			assert.Equal(t, bv, valuesOf(b))
		})
	}
}

func TestOr(t *testing.T) {
	tc := []struct {
		name   string
		c1     *container
		c2     *container
		result []uint16
	}{
		{"empty", newArr(), newArr(), []uint16{}},
		{"arr ∨ arr", newArr(1, 2, 3), newArr(1, 2, 3), []uint16{1, 2, 3}},
		{"arr ∨ bmp", newArr(1, 2, 3), newBmp(1, 2, 3), []uint16{1, 2, 3}},
		{"arr ∨ run", newArr(1, 2, 3), newRun(1, 2, 3), []uint16{1, 2, 3}},
		{"bmp ∨ arr", newBmp(1, 2, 3), newArr(1, 2, 3), []uint16{1, 2, 3}},
		{"bmp ∨ bmp", newBmp(1, 2, 3), newBmp(1, 2, 3), []uint16{1, 2, 3}},
		{"bmp ∨ run", newBmp(1, 2, 3), newRun(1, 2, 3), []uint16{1, 2, 3}},
		{"run ∨ arr", newRun(1, 2, 3), newArr(1, 2, 3), []uint16{1, 2, 3}},
		{"run ∨ bmp", newRun(1, 2, 3), newBmp(1, 2, 3), []uint16{1, 2, 3}},
		{"run ∨ run", newRun(1, 2, 3), newRun(1, 2, 3), []uint16{1, 2, 3}},

		// Partial unions
		{"arr ∨ arr partial", newArr(1, 2, 3), newArr(4, 5, 6), []uint16{1, 2, 3, 4, 5, 6}},
		{"arr ∨ bmp partial", newArr(1, 2, 3), newBmp(4, 5, 6), []uint16{1, 2, 3, 4, 5, 6}},
		{"arr ∨ run partial", newArr(1, 2, 3), newRun(4, 5, 6), []uint16{1, 2, 3, 4, 5, 6}},
		{"bmp ∨ arr partial", newBmp(1, 2, 3), newArr(4, 5, 6), []uint16{1, 2, 3, 4, 5, 6}},
		{"bmp ∨ bmp partial", newBmp(1, 2, 3), newBmp(4, 5, 6), []uint16{1, 2, 3, 4, 5, 6}},
		{"bmp ∨ run partial", newBmp(1, 2, 3), newRun(4, 5, 6), []uint16{1, 2, 3, 4, 5, 6}},
		{"run ∨ arr partial", newRun(1, 2, 3), newArr(4, 5, 6), []uint16{1, 2, 3, 4, 5, 6}},
		{"run ∨ bmp partial", newRun(1, 2, 3), newBmp(4, 5, 6), []uint16{1, 2, 3, 4, 5, 6}},
		{"run ∨ run partial", newRun(1, 2, 3), newRun(4, 5, 6), []uint16{1, 2, 3, 4, 5, 6}},

		// Overlapping unions
		{"arr ∨ arr overlap", newArr(1, 2, 3, 4), newArr(3, 4, 5, 6), []uint16{1, 2, 3, 4, 5, 6}},
		{"arr ∨ bmp overlap", newArr(1, 2, 3, 4), newBmp(3, 4, 5, 6), []uint16{1, 2, 3, 4, 5, 6}},
		{"arr ∨ run overlap", newArr(1, 2, 3, 4), newRun(3, 4, 5, 6), []uint16{1, 2, 3, 4, 5, 6}},
		{"bmp ∨ arr overlap", newBmp(1, 2, 3, 4), newArr(3, 4, 5, 6), []uint16{1, 2, 3, 4, 5, 6}},
		{"bmp ∨ bmp overlap", newBmp(1, 2, 3, 4), newBmp(3, 4, 5, 6), []uint16{1, 2, 3, 4, 5, 6}},
		{"bmp ∨ run overlap", newBmp(1, 2, 3, 4), newRun(3, 4, 5, 6), []uint16{1, 2, 3, 4, 5, 6}},
		{"run ∨ arr overlap", newRun(1, 2, 3, 4), newArr(3, 4, 5, 6), []uint16{1, 2, 3, 4, 5, 6}},
		{"run ∨ bmp overlap", newRun(1, 2, 3, 4), newBmp(3, 4, 5, 6), []uint16{1, 2, 3, 4, 5, 6}},
		{"run ∨ run overlap", newRun(1, 2, 3, 4), newRun(3, 4, 5, 6), []uint16{1, 2, 3, 4, 5, 6}},

		// Single element cases
		{"arr ∨ arr single", newArr(1), newArr(2), []uint16{1, 2}},
		{"arr ∨ bmp single", newArr(1), newBmp(2), []uint16{1, 2}},
		{"arr ∨ run single", newArr(1), newRun(2), []uint16{1, 2}},
		{"bmp ∨ arr single", newBmp(1), newArr(2), []uint16{1, 2}},
		{"bmp ∨ bmp single", newBmp(1), newBmp(2), []uint16{1, 2}},
		{"bmp ∨ run single", newBmp(1), newRun(2), []uint16{1, 2}},
		{"run ∨ arr single", newRun(1), newArr(2), []uint16{1, 2}},
		{"run ∨ bmp single", newRun(1), newBmp(2), []uint16{1, 2}},
		{"run ∨ run single", newRun(1), newRun(2), []uint16{1, 2}},

		// Boundary values
		{"arr ∨ arr boundary", newArr(0, 1), newArr(65534, 65535), []uint16{0, 1, 65534, 65535}},
		{"arr ∨ bmp boundary", newArr(0, 1), newBmp(65534, 65535), []uint16{0, 1, 65534, 65535}},
		{"arr ∨ run boundary", newArr(0, 1), newRun(65534, 65535), []uint16{0, 1, 65534, 65535}},
		{"bmp ∨ arr boundary", newBmp(0, 1), newArr(65534, 65535), []uint16{0, 1, 65534, 65535}},
		{"bmp ∨ bmp boundary", newBmp(0, 1), newBmp(65534, 65535), []uint16{0, 1, 65534, 65535}},
		{"bmp ∨ run boundary", newBmp(0, 1), newRun(65534, 65535), []uint16{0, 1, 65534, 65535}},
		{"run ∨ arr boundary", newRun(0, 1), newArr(65534, 65535), []uint16{0, 1, 65534, 65535}},
		{"run ∨ bmp boundary", newRun(0, 1), newBmp(65534, 65535), []uint16{0, 1, 65534, 65535}},
		{"run ∨ run boundary", newRun(0, 1), newRun(65534, 65535), []uint16{0, 1, 65534, 65535}},

		// One side empty
		{"arr ∨ empty", newArr(1, 2, 3), newArr(), []uint16{1, 2, 3}},
		{"bmp ∨ empty", newBmp(1, 2, 3), newArr(), []uint16{1, 2, 3}},
		{"run ∨ empty", newRun(1, 2, 3), newArr(), []uint16{1, 2, 3}},
		{"empty ∨ arr", newArr(), newArr(1, 2, 3), []uint16{1, 2, 3}},
		{"empty ∨ bmp", newArr(), newBmp(1, 2, 3), []uint16{1, 2, 3}},
		{"empty ∨ run", newArr(), newRun(1, 2, 3), []uint16{1, 2, 3}},

		// Adjacent ranges with runs
		{"run ∨ run adjacent", newRun(1, 2, 3, 4), newRun(5, 6, 7, 8), []uint16{1, 2, 3, 4, 5, 6, 7, 8}},
		{"arr ∨ run ranges", newArr(1, 3, 5, 7, 9), newRun(2, 4, 6, 8, 10), []uint16{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
		{"bmp ∨ run ranges", newBmp(1, 3, 5, 7, 9), newRun(2, 4, 6, 8, 10), []uint16{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},

		// Complex overlapping patterns
		{"run ∨ run complex", newRun(1, 2, 3, 4, 5, 10, 11, 12), newRun(3, 4, 5, 6, 7, 11, 12, 13), []uint16{1, 2, 3, 4, 5, 6, 7, 10, 11, 12, 13}},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			a, _ := bitmapWith(tt.c1)
			b, bv := bitmapWith(tt.c2)

			a.Or(b)

			// Assert the result is correct
			assert.Equal(t, tt.result, valuesOf(a))
			assert.Equal(t, bv, valuesOf(b))
		})
	}
}

func TestXor(t *testing.T) {
	tc := []struct {
		name   string
		c1     *container
		c2     *container
		result []uint16
	}{
		{"empty", newArr(), newArr(), []uint16{}},
		{"arr ⊕ arr", newArr(1, 2, 3), newArr(1, 2, 3), []uint16{}}, // XOR identical = empty
		{"arr ⊕ bmp", newArr(1, 2, 3), newBmp(1, 2, 3), []uint16{}},
		{"arr ⊕ run", newArr(1, 2, 3), newRun(1, 2, 3), []uint16{}},
		{"bmp ⊕ arr", newBmp(1, 2, 3), newArr(1, 2, 3), []uint16{}},
		{"bmp ⊕ bmp", newBmp(1, 2, 3), newBmp(1, 2, 3), []uint16{}},
		{"bmp ⊕ run", newBmp(1, 2, 3), newRun(1, 2, 3), []uint16{}},
		{"run ⊕ arr", newRun(1, 2, 3), newArr(1, 2, 3), []uint16{}},
		{"run ⊕ bmp", newRun(1, 2, 3), newBmp(1, 2, 3), []uint16{}},
		{"run ⊕ run", newRun(1, 2, 3), newRun(1, 2, 3), []uint16{}},

		// Disjoint sets (complete symmetric difference)
		{"arr ⊕ arr disjoint", newArr(1, 2, 3), newArr(4, 5, 6), []uint16{1, 2, 3, 4, 5, 6}},
		{"arr ⊕ bmp disjoint", newArr(1, 2, 3), newBmp(4, 5, 6), []uint16{1, 2, 3, 4, 5, 6}},
		{"arr ⊕ run disjoint", newArr(1, 2, 3), newRun(4, 5, 6), []uint16{1, 2, 3, 4, 5, 6}},
		{"bmp ⊕ arr disjoint", newBmp(1, 2, 3), newArr(4, 5, 6), []uint16{1, 2, 3, 4, 5, 6}},
		{"bmp ⊕ bmp disjoint", newBmp(1, 2, 3), newBmp(4, 5, 6), []uint16{1, 2, 3, 4, 5, 6}},
		{"bmp ⊕ run disjoint", newBmp(1, 2, 3), newRun(4, 5, 6), []uint16{1, 2, 3, 4, 5, 6}},
		{"run ⊕ arr disjoint", newRun(1, 2, 3), newArr(4, 5, 6), []uint16{1, 2, 3, 4, 5, 6}},
		{"run ⊕ bmp disjoint", newRun(1, 2, 3), newBmp(4, 5, 6), []uint16{1, 2, 3, 4, 5, 6}},
		{"run ⊕ run disjoint", newRun(1, 2, 3), newRun(4, 5, 6), []uint16{1, 2, 3, 4, 5, 6}},

		// Partial overlaps (symmetric difference)
		{"arr ⊕ arr overlap", newArr(1, 2, 3, 4), newArr(3, 4, 5, 6), []uint16{1, 2, 5, 6}},
		{"arr ⊕ bmp overlap", newArr(1, 2, 3, 4), newBmp(3, 4, 5, 6), []uint16{1, 2, 5, 6}},
		{"arr ⊕ run overlap", newArr(1, 2, 3, 4), newRun(3, 4, 5, 6), []uint16{1, 2, 5, 6}},
		{"bmp ⊕ arr overlap", newBmp(1, 2, 3, 4), newArr(3, 4, 5, 6), []uint16{1, 2, 5, 6}},
		{"bmp ⊕ bmp overlap", newBmp(1, 2, 3, 4), newBmp(3, 4, 5, 6), []uint16{1, 2, 5, 6}},
		{"bmp ⊕ run overlap", newBmp(1, 2, 3, 4), newRun(3, 4, 5, 6), []uint16{1, 2, 5, 6}},
		{"run ⊕ arr overlap", newRun(1, 2, 3, 4), newArr(3, 4, 5, 6), []uint16{1, 2, 5, 6}},
		{"run ⊕ bmp overlap", newRun(1, 2, 3, 4), newBmp(3, 4, 5, 6), []uint16{1, 2, 5, 6}},
		{"run ⊕ run overlap", newRun(1, 2, 3, 4), newRun(3, 4, 5, 6), []uint16{1, 2, 5, 6}},

		// Single element differences
		{"arr ⊕ arr single", newArr(1, 2, 3), newArr(2), []uint16{1, 3}},
		{"arr ⊕ bmp single", newArr(1, 2, 3), newBmp(2), []uint16{1, 3}},
		{"arr ⊕ run single", newArr(1, 2, 3), newRun(2), []uint16{1, 3}},
		{"bmp ⊕ arr single", newBmp(1, 2, 3), newArr(2), []uint16{1, 3}},
		{"bmp ⊕ bmp single", newBmp(1, 2, 3), newBmp(2), []uint16{1, 3}},
		{"bmp ⊕ run single", newBmp(1, 2, 3), newRun(2), []uint16{1, 3}},
		{"run ⊕ arr single", newRun(1, 2, 3), newArr(2), []uint16{1, 3}},
		{"run ⊕ bmp single", newRun(1, 2, 3), newBmp(2), []uint16{1, 3}},
		{"run ⊕ run single", newRun(1, 2, 3), newRun(2), []uint16{1, 3}},

		// Boundary values
		{"arr ⊕ arr boundary", newArr(0, 1, 65535), newArr(0, 65535), []uint16{1}},
		{"arr ⊕ bmp boundary", newArr(0, 1, 65535), newBmp(0, 65535), []uint16{1}},
		{"arr ⊕ run boundary", newArr(0, 1, 65535), newRun(0, 65535), []uint16{1}},
		{"bmp ⊕ arr boundary", newBmp(0, 1, 65535), newArr(0, 65535), []uint16{1}},
		{"bmp ⊕ bmp boundary", newBmp(0, 1, 65535), newBmp(0, 65535), []uint16{1}},
		{"bmp ⊕ run boundary", newBmp(0, 1, 65535), newRun(0, 65535), []uint16{1}},
		{"run ⊕ arr boundary", newRun(0, 1, 65535), newArr(0, 65535), []uint16{1}},
		{"run ⊕ bmp boundary", newRun(0, 1, 65535), newBmp(0, 65535), []uint16{1}},
		{"run ⊕ run boundary", newRun(0, 1, 65535), newRun(0, 65535), []uint16{1}},

		// One side empty (XOR with empty = identity)
		{"arr ⊕ empty", newArr(1, 2, 3), newArr(), []uint16{1, 2, 3}},
		{"bmp ⊕ empty", newBmp(1, 2, 3), newArr(), []uint16{1, 2, 3}},
		{"run ⊕ empty", newRun(1, 2, 3), newArr(), []uint16{1, 2, 3}},
		{"empty ⊕ arr", newArr(), newArr(1, 2, 3), []uint16{1, 2, 3}},
		{"empty ⊕ bmp", newArr(), newBmp(1, 2, 3), []uint16{1, 2, 3}},
		{"empty ⊕ run", newArr(), newRun(1, 2, 3), []uint16{1, 2, 3}},

		// Complex patterns
		{"arr ⊕ run complex", newArr(1, 3, 5, 7, 9), newRun(2, 3, 6, 7, 10), []uint16{1, 2, 5, 6, 9, 10}},
		{"bmp ⊕ run complex", newBmp(1, 3, 5, 7, 9), newRun(2, 3, 6, 7, 10), []uint16{1, 2, 5, 6, 9, 10}},
		{"run ⊕ run complex", newRun(1, 2, 3, 10, 11, 12), newRun(2, 3, 4, 11, 12, 13), []uint16{1, 4, 10, 13}},

		// Subset relationships
		{"arr ⊕ arr subset", newArr(1, 2, 3, 4, 5), newArr(2, 4), []uint16{1, 3, 5}},
		{"bmp ⊕ arr subset", newBmp(1, 2, 3, 4, 5), newArr(2, 4), []uint16{1, 3, 5}},
		{"run ⊕ arr subset", newRun(1, 2, 3, 4, 5), newArr(2, 4), []uint16{1, 3, 5}},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			a, _ := bitmapWith(tt.c1)
			b, bv := bitmapWith(tt.c2)

			a.Xor(b)

			// Assert the result is correct
			values := valuesOf(a)
			sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
			assert.Equal(t, tt.result, values)
			assert.Equal(t, bv, valuesOf(b))
		})
	}
}

func TestXorRun(t *testing.T) {
	a, _ := bitmapWith(newArr(1, 3, 5, 7, 9))
	b, _ := bitmapWith(newRun(2, 3, 6, 7, 10))

	a.Xor(b)

	assert.Equal(t, []uint16{1, 2, 5, 6, 9, 10}, valuesOf(a))
	for _, value := range []uint32{1, 2, 5, 6, 9, 10} {
		assert.True(t, a.Contains(value), "value %d should remain searchable", value)
	}
	for _, value := range []uint32{3, 7} {
		assert.False(t, a.Contains(value), "value %d should be removed", value)
	}
}

func TestOperationsExtras(t *testing.T) {
	t.Run("and", func(t *testing.T) {
		a := bitmapOf(1, 2, 3, 4)
		a.And(bitmapOf(2, 3, 4), nil, bitmapOf(3, 4))
		assert.Equal(t, []uint32{3, 4}, values32(a))
	})

	t.Run("andnot", func(t *testing.T) {
		a := bitmapOf(1, 2, 3, 4)
		a.AndNot(bitmapOf(2), nil, bitmapOf(4))
		assert.Equal(t, []uint32{1, 3}, values32(a))
	})

	t.Run("or", func(t *testing.T) {
		a := bitmapOf(1)
		a.Or(bitmapOf(2), nil, bitmapOf(3))
		assert.Equal(t, []uint32{1, 2, 3}, values32(a))
	})

	t.Run("xor", func(t *testing.T) {
		a := bitmapOf(1, 2)
		a.Xor(bitmapOf(2, 3), nil, bitmapOf(3, 4))
		assert.Equal(t, []uint32{1, 4}, values32(a))
	})
}

func TestOperationsEmpty(t *testing.T) {
	t.Run("and nil", func(t *testing.T) {
		a := bitmapOf(1)
		a.And(nil)
		assert.Empty(t, values32(a))
	})

	t.Run("andnot nil", func(t *testing.T) {
		a := bitmapOf(1)
		a.AndNot(nil)
		assert.Equal(t, []uint32{1}, values32(a))
	})

	t.Run("or empty", func(t *testing.T) {
		a := bitmapOf(1)
		a.Or(New())
		assert.Equal(t, []uint32{1}, values32(a))
	})

	t.Run("xor empty", func(t *testing.T) {
		a := bitmapOf(1)
		a.Xor(New())
		assert.Equal(t, []uint32{1}, values32(a))
	})

	t.Run("empty or", func(t *testing.T) {
		a := New()
		a.Or(bitmapOf(2))
		assert.Equal(t, []uint32{2}, values32(a))
	})

	t.Run("empty xor", func(t *testing.T) {
		a := New()
		a.Xor(bitmapOf(2))
		assert.Equal(t, []uint32{2}, values32(a))
	})
}

func TestContainerSharing(t *testing.T) {
	tests := []struct {
		name string
		op   func(dst, src *Bitmap)
	}{
		{"or", func(dst, src *Bitmap) { dst.Or(src) }},
		{"xor", func(dst, src *Bitmap) { dst.Xor(src) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := bitmapOf(1, 2, 1<<16|3)
			dst := New()

			tt.op(dst, src)
			dst.Remove(1)
			dst.Set(4)
			src.Remove(2)
			src.Set(5)

			assert.False(t, src.Contains(4))
			assert.True(t, src.Contains(1))
			assert.False(t, dst.Contains(5))
			assert.True(t, dst.Contains(2))
			assert.True(t, dst.Contains(1<<16|3))
		})
	}
}

func TestRunUnion(t *testing.T) {
	a, _ := bitmapWith(&container{Type: typeRun, Size: 19, Data: []uint16{1, 10, 12, 20}})
	b, _ := bitmapWith(&container{Type: typeRun, Size: 98, Data: []uint16{1, 2, 5, 100}})
	a.Or(b)
	assert.Equal(t, []uint16{1, 100}, a.containers[0].Data)
	assert.Equal(t, 100, a.Count())
	assert.Len(t, values32(a), 100)
	assert.Equal(t, []uint16{1, 2, 5, 100}, b.containers[0].Data)
}

func TestSelfXor(t *testing.T) {
	for _, c := range []*container{newArr(1, 3, 4, 5), newBmp(1, 3, 4, 5), newRun(1, 3, 4, 5)} {
		bm, _ := bitmapWith(c)
		bm.Xor(bm, bitmapOf(65535))
		assert.Equal(t, []uint32{65535}, values32(bm))
		assert.Equal(t, 1, bm.Count())
	}
}

func TestMathModel(t *testing.T) {
	types := []struct {
		name string
		typ  ctype
	}{
		{"array", typeArray},
		{"bitmap", typeBitmap},
		{"run", typeRun},
	}
	keys := []uint16{0, 2, 65535}
	leftData := map[uint16][]uint16{
		0:     {0, 1, 2, 10, 11, 65534, 65535},
		2:     {0, 4, 5, 65535},
		65535: {0, 65534, 65535},
	}
	rightKeys := []uint16{0, 1, 65535}
	rightData := map[uint16][]uint16{
		0:     {1, 2, 3, 10, 12, 65535},
		1:     {0, 1, 5, 65534},
		65535: {1, 65535},
	}

	makeBitmap := func(typ ctype, useKeys []uint16, data map[uint16][]uint16) *Bitmap {
		out := New()
		for _, key := range useKeys {
			values := make([]uint32, len(data[key]))
			for i, value := range data[key] {
				values[i] = uint32(value)
			}
			out.ctrAdd(key, len(out.containers), newContainer(typ, values...))
		}
		return out
	}

	toSet := func(values []uint32) map[uint32]struct{} {
		out := make(map[uint32]struct{}, len(values))
		for _, value := range values {
			out[value] = struct{}{}
		}
		return out
	}
	cloneSet := func(values map[uint32]struct{}) map[uint32]struct{} {
		out := make(map[uint32]struct{}, len(values))
		for value := range values {
			out[value] = struct{}{}
		}
		return out
	}
	oracle := func(op string, left, right map[uint32]struct{}) map[uint32]struct{} {
		out := cloneSet(left)
		switch op {
		case "and":
			for value := range out {
				if _, ok := right[value]; !ok {
					delete(out, value)
				}
			}
		case "andnot":
			for value := range right {
				delete(out, value)
			}
		case "or":
			for value := range right {
				out[value] = struct{}{}
			}
		case "xor":
			for value := range right {
				if _, ok := out[value]; ok {
					delete(out, value)
				} else {
					out[value] = struct{}{}
				}
			}
		}
		return out
	}
	ordered := func(values map[uint32]struct{}) []uint32 {
		out := make([]uint32, 0, len(values))
		for value := range values {
			out = append(out, value)
		}
		sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
		return out
	}

	operations := []struct {
		name  string
		apply func(*Bitmap, *Bitmap)
	}{
		{"and", func(left, right *Bitmap) { left.And(right) }},
		{"andnot", func(left, right *Bitmap) { left.AndNot(right) }},
		{"or", func(left, right *Bitmap) { left.Or(right) }},
		{"xor", func(left, right *Bitmap) { left.Xor(right) }},
	}

	for _, leftType := range types {
		for _, rightType := range types {
			left := makeBitmap(leftType.typ, keys, leftData)
			right := makeBitmap(rightType.typ, rightKeys, rightData)
			leftValues := values32(left)
			rightValues := values32(right)
			leftSet := toSet(leftValues)
			rightSet := toSet(rightValues)

			for _, operation := range operations {
				t.Run(leftType.name+"/"+rightType.name+"/"+operation.name, func(t *testing.T) {
					working := left.Clone(nil)
					first := oracle(operation.name, leftSet, rightSet)
					second := oracle(operation.name, first, rightSet)

					for step, expected := range []map[uint32]struct{}{first, second} {
						operation.apply(working, right)
						values := values32(working)
						assert.True(t, sort.SliceIsSorted(values, func(i, j int) bool { return values[i] < values[j] }))
						assert.Equal(t, ordered(expected), values)
						assert.Equal(t, len(expected), working.Count(), "step %d count", step)
						for value := range leftSet {
							_, want := expected[value]
							assert.Equal(t, want, working.Contains(value), "step %d value %d", step, value)
						}
						for value := range rightSet {
							_, want := expected[value]
							assert.Equal(t, want, working.Contains(value), "step %d value %d", step, value)
						}
						assert.Equal(t, leftValues, values32(left), "left source changed at step %d", step)
						assert.Equal(t, rightValues, values32(right), "right source changed at step %d", step)
					}
				})
			}
		}
	}
}

func TestXorCompaction(t *testing.T) {
	for _, right := range []*Bitmap{bitmapOf(1), bitmapOf(1, 2<<16|1)} {
		left := bitmapOf(1, 1<<16|1, 2<<16|1)
		left.Xor(right)
		assert.True(t, left.Contains(1<<16|1))
		assert.False(t, left.Contains(1))
		assert.Equal(t, !right.Contains(2<<16|1), left.Contains(2<<16|1))
		assert.True(t, containerTailCleared(left))
	}
}

func TestMathLarge(t *testing.T) {
	for _, count := range []int{branchlessAt - 1, branchlessAt, branchlessAt + 1} {
		left, right := New(), New()
		for key := 0; key < count; key++ {
			for _, value := range []uint32{0, 2, 4, 65535} {
				left.Set(uint32(key)<<16 | value)
			}
			for _, value := range []uint32{1, 2, 4, 65534} {
				right.Set(uint32(key)<<16 | value)
			}
		}
		for _, operation := range []struct {
			apply func(*Bitmap, *Bitmap)
			want  []uint32
		}{
			{func(a, b *Bitmap) { a.And(b) }, []uint32{2, 4}},
			{func(a, b *Bitmap) { a.AndNot(b) }, []uint32{0, 65535}},
			{func(a, b *Bitmap) { a.Or(b) }, []uint32{0, 1, 2, 4, 65534, 65535}},
			{func(a, b *Bitmap) { a.Xor(b) }, []uint32{0, 1, 65534, 65535}},
		} {
			result := left.Clone(nil)
			operation.apply(result, right)
			want := make([]uint32, 0, count*len(operation.want))
			for key := 0; key < count; key++ {
				for _, value := range operation.want {
					want = append(want, uint32(key)<<16|value)
				}
			}
			assert.Equal(t, want, values32(result))
			assert.Equal(t, len(want), result.Count())
			assert.Equal(t, count*4, left.Count())
			assert.Equal(t, count*4, right.Count())
		}
	}
}

func TestMathTypes(t *testing.T) {
	types := []struct {
		name string
		typ  ctype
	}{
		{"array", typeArray},
		{"bitmap", typeBitmap},
		{"run", typeRun},
	}
	ops := []struct {
		name  string
		apply func(*Bitmap, *Bitmap)
		want  []uint32
	}{
		{"and", func(left, right *Bitmap) { left.And(right) }, []uint32{2, 3}},
		{"andnot", func(left, right *Bitmap) { left.AndNot(right) }, []uint32{1, 4}},
		{"or", func(left, right *Bitmap) { left.Or(right) }, []uint32{1, 2, 3, 4, 5, 6}},
		{"xor", func(left, right *Bitmap) { left.Xor(right) }, []uint32{1, 4, 5, 6}},
	}

	for _, op := range ops {
		for _, leftType := range types {
			for _, rightType := range types {
				name := op.name + "/" + leftType.name + "/" + rightType.name
				t.Run(name, func(t *testing.T) {
					left := New()
					left.ctrAdd(0, 0, newContainer(leftType.typ, 1, 2, 3, 4))
					right := New()
					right.ctrAdd(0, 0, newContainer(rightType.typ, 2, 3, 5, 6))

					op.apply(left, right)

					assert.Equal(t, op.want, values32(left))
				})
			}
		}
	}
}
