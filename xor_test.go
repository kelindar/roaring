package roaring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			assert.Equal(t, tt.result, valuesOf(a))
			assert.Equal(t, bv, valuesOf(b))
		})
	}
}
