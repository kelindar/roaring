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
