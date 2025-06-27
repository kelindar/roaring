package roaring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
