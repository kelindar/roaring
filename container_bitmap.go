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

// bmpOptimize tries to optimize the container
func (c *container) bmpOptimize() {
	switch {
	case c.bmpIsDense():
		c.bmpToRun()
	case c.Size <= arrMinSize:
		c.bmpToArr()
	}
}

// bmpIsDense quickly estimates if converting to run container would be beneficial
func (c *container) bmpIsDense() bool {
	if c.Size == 0 || c.Size < 50 {
		return false
	}

	bmp := c.bmp()
	lo, loOk := bmp.Min()
	hi, hiOk := bmp.Max()
	if !loOk || !hiOk {
		return false
	}

	size := int(c.Size)
	span := int(hi - lo + 1)
	density := float64(size) / float64(span)

	// Quick density filters
	switch {
	case density < 0.1: // Very sparse
		return false
	case density > 0.9: // Very dense
		return true
	}

	// Estimate runs based on density
	runs := size
	if gap := float64(span) / float64(size); gap < 2.0 {
		runs = int(float64(size) * (1.0 - density*0.8))
	}

	// Check if estimated conversion meets our criteria
	sizeAsRun := runs*4 + 2
	return runs <= 10 &&
		sizeAsRun < 8192/4 &&
		sizeAsRun < size
}

// bmpToRun attempts to convert bitmap to run in a single pass
// Returns true if conversion was performed, false otherwise
func (c *container) bmpToRun() bool {
	bmp := c.bmp()
	var runs []run
	var curr, last uint16
	var inRun bool

	// Single iteration: build runs AND count them
	bmp.Range(func(value uint32) {
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

	// Now check conversion criteria with the actual run count
	numRuns := len(runs)
	cardinality := int(c.Size)

	// Very conservative thresholds to avoid premature conversion
	const sizeAsBitmapContainer = 8192

	// Estimated size as run container (each run takes 4 bytes + 2 bytes header)
	sizeAsRunContainer := 2 + numRuns*4

	// Size as array container (2 bytes per element)
	sizeAsArrayContainer := cardinality * 2

	// Only convert if run representation is MUCH smaller and we have very few runs
	shouldConvert := numRuns <= 5 &&
		sizeAsRunContainer < sizeAsBitmapContainer/4 &&
		sizeAsRunContainer < sizeAsArrayContainer/2

	if shouldConvert {
		// Convert using the pre-built runs (no second iteration needed!)
		c.Data = make([]byte, len(runs)*4) // 4 bytes per run (2 uint16s)
		c.Type = typeRun
		newRuns := c.run()
		copy(newRuns, runs)
		return true
	}

	return false
}

// bmpToArr converts this container from bitmap to array
func (c *container) bmpToArr() {
	src := c.bmp()

	// Create new array data
	c.Data = make([]byte, src.Count()*2)
	c.Type = typeArray

	// Copy all values to the array
	dst := c.arr()
	src.Range(func(value uint32) {
		dst = append(dst, uint16(value))
	})
}
