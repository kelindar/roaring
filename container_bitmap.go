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

	return bitmap.Bitmap(unsafe.Slice((*uint64)(unsafe.Pointer(&c.Data[0])), len(c.Data)/4))
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
	blkAt := int(value >> 6)
	if size := len(c.bmp()); blkAt >= size {
		return false
	}

	bitAt := int(value % 64)
	blk := &c.bmp()[blkAt]
	if (*blk & (1 << bitAt)) > 0 {
		*blk &^= (1 << bitAt)
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
func (c *container) bmpToRun() bool {
	bmp := c.bmp()
	runsData := make([]uint16, 0, 32) // estimate for initial capacity
	var curr, last uint16
	var inRun bool

	// Single iteration: build runs
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
			runsData = append(runsData, curr, last)
			curr = v
			last = v
		}
	})

	// Handle the last run if we were in one
	if inRun {
		runsData = append(runsData, curr, last)
	}

	// Check conversion criteria with the actual run count
	numRuns := len(runsData) / 2
	cardinality := int(c.Size)
	sizeAsRunContainer := 2 + numRuns*4
	sizeAsArrayContainer := cardinality * 2

	// Only convert if run representation is much smaller and we have very few runs
	shouldConvert := numRuns <= 5 &&
		sizeAsRunContainer < 8192/4 &&
		sizeAsRunContainer < sizeAsArrayContainer/2

	if shouldConvert {
		c.Data = runsData
		c.Type = typeRun
		return true
	}

	return false
}

// bmpToArr converts this container from bitmap to array
func (c *container) bmpToArr() {
	src := c.bmp()

	// Pre-allocate array data based on cardinality
	c.Data = make([]uint16, c.Size) // uint16 per element
	c.Type = typeArray

	// Copy all values to the array efficiently
	dst := c.arr()
	idx := 0
	src.Range(func(value uint32) {
		dst[idx] = uint16(value)
		idx++
	})
}
