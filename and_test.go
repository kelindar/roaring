package roaring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			a := New()
			a.ctrAdd(0, 0, tt.c1)
			b := New()
			b.ctrAdd(0, 0, tt.c2)
			a.And(b)

			result := []uint16{}
			a.Range(func(x uint32) {
				result = append(result, uint16(x))
			})

			assert.Equal(t, tt.result, result)
		})
	}
}
