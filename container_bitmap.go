package roaring

import (
	"unsafe"

	"github.com/kelindar/bitmap"
)

// bmp converts the container to a bmp.Bitmap
func (c *container) bmp() bitmap.Bitmap {
	if len(c.Data) == 0 {
		return nil
	}

	return bitmap.Bitmap(unsafe.Slice((*uint64)(unsafe.Pointer(&c.Data[0])), len(c.Data)/8))
}

// bmpSet sets a value in a bitmap container
func (c *container) bmpSet(value uint16) bool {
	if b := c.bmp(); !b.Contains(uint32(value)) {
		b.Set(uint32(value))
		c.Size++
		return true
	}
	return false
}

// bmpDel removes a value from a bitmap container
func (c *container) bmpDel(value uint16) bool {
	if b := c.bmp(); b.Contains(uint32(value)) {
		b.Remove(uint32(value))
		c.Size--
		return true
	}
	return false
}

// bmpHas checks if a value exists in a bitmap container
func (c *container) bmpHas(value uint16) bool {
	return c.bmp().Contains(uint32(value))
}

// bmpShouldConvertToArray returns true if bitmap should be converted to array
func (c *container) bmpShouldConvertToArray() bool {
	return c.Size <= arrMinSize
}

// bmpShouldConvertToRun returns true if bitmap should be converted to run
func (c *container) bmpShouldConvertToRun() bool {
	if c.Size == 0 {
		return false
	}

	numRuns := c.bmpNumberOfRuns()
	cardinality := int(c.Size)

	// Very conservative thresholds to avoid premature conversion
	const sizeAsBitmapContainer = 8192

	// Estimated size as run container (each run takes 4 bytes + 2 bytes header)
	sizeAsRunContainer := 2 + numRuns*4

	// Size as array container (2 bytes per element)
	sizeAsArrayContainer := cardinality * 2

	// Only convert if run representation is MUCH smaller and we have very few runs
	return numRuns <= 5 &&
		sizeAsRunContainer < sizeAsBitmapContainer/4 &&
		sizeAsRunContainer < sizeAsArrayContainer/2
}

// bmpNumberOfRuns counts consecutive runs in the bitmap
// This implements the same logic as the official RoaringBitmap implementation
func (c *container) bmpNumberOfRuns() int {
	if c.Size == 0 {
		return 0
	}

	bm := c.bmp()
	numRuns := 0

	// Use Range to iterate only over set bits
	var lastValue uint16
	var inRun bool

	bm.Range(func(value uint32) {
		v := uint16(value)
		if !inRun {
			// Start of first run
			numRuns++
			inRun = true
		} else if v != lastValue+1 {
			// Gap found, start new run
			numRuns++
		}
		lastValue = v
	})

	return numRuns
}

// bmpToArr converts this container from bitmap to array
func (c *container) bmpToArr() {
	bmp := c.bmp()
	out := make([]uint16, 0, bmp.Count())
	bmp.Range(func(value uint32) {
		out = append(out, uint16(value))
	})

	// Create new array data
	c.Data = make([]byte, len(out)*2)
	c.Type = typeArray
	c.Size = uint16(len(out)) // Set cardinality
	array := c.arr()
	copy(array, out)
}

// bmpToRun converts this container from bitmap to run
func (c *container) bmpToRun() {
	bm := c.bmp()
	var runs []run

	// Use Range to iterate only over set bits
	var curr uint16
	var last uint16
	var inRun bool

	bm.Range(func(value uint32) {
		v := uint16(value)
		switch {
		case !inRun:
			curr = v
			last = v
			inRun = true
		case v == last+1:
			last = v
		default:
			runs = append(runs, run{curr, last})
			curr = v
			last = v
		}
	})

	// Handle the last run if we were in one
	if inRun {
		runs = append(runs, run{curr, last})
	}

	// Create new run data
	c.Data = make([]byte, len(runs)*4) // 4 bytes per run (2 uint16s)
	c.Type = typeRun
	newRuns := c.run()
	copy(newRuns, runs)
}
