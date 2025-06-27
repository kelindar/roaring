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
		result []uint32
	}{
		{"empty", newArr(), newArr(), []uint32{}},
		{"arrAndarr", newArr(1, 2, 3), newArr(1, 2, 3), []uint32{1, 2, 3}},
		{"arrAndbmp", newArr(1, 2, 3), newBmp(1, 2, 3), []uint32{1, 2, 3}},
		{"arrAndrun", newArr(1, 2, 3), newRun(1, 2, 3), []uint32{1, 2, 3}},
		{"bmpAndarr", newBmp(1, 2, 3), newArr(1, 2, 3), []uint32{1, 2, 3}},
		{"bmpAndbmp", newBmp(1, 2, 3), newBmp(1, 2, 3), []uint32{1, 2, 3}},
		{"bmpAndrun", newBmp(1, 2, 3), newRun(1, 2, 3), []uint32{1, 2, 3}},
		{"runAndarr", newRun(1, 2, 3), newArr(1, 2, 3), []uint32{1, 2, 3}},
		{"runAndbmp", newRun(1, 2, 3), newBmp(1, 2, 3), []uint32{1, 2, 3}},
		{"runAndrun", newRun(1, 2, 3), newRun(1, 2, 3), []uint32{1, 2, 3}},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {

			a := New()
			a.ctrAdd(0, 0, tt.c1)

			b := New()
			b.ctrAdd(0, 0, tt.c2)

			a.And(b)

			var result []uint32
			a.Range(func(x uint32) {
				result = append(result, x)
			})

			assert.Equal(t, tt.result, tt.c1.Data)
		})
	}
}
