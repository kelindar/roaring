package roaring

import (
	"unsafe"

	"github.com/kelindar/bitmap"
)

// bitmap converts the container to a bitmap.Bitmap
func (c *container) bitmap() bitmap.Bitmap {
	if len(c.Data) == 0 {
		return nil
	}

	return bitmap.Bitmap(unsafe.Slice((*uint64)(unsafe.Pointer(&c.Data[0])), len(c.Data)/8))
}

// bitmapSet sets a value in a bitmap container
func (c *container) bitmapSet(value uint16) bool {
	bm := c.bitmap()
	if bm.Contains(uint32(value)) {
		return false // Already exists
	}
	bm.Set(uint32(value))
	c.Size = uint16(bm.Count()) // Update cardinality from bitmap
	return true
}

// bitmapRemove removes a value from a bitmap container
func (c *container) bitmapRemove(value uint16) bool {
	bm := c.bitmap()
	if bm.Contains(uint32(value)) {
		bm.Remove(uint32(value))
		c.Size = uint16(bm.Count()) // Update cardinality from bitmap
		return true
	}
	return false
}

// bitmapContains checks if a value exists in a bitmap container
func (c *container) bitmapContains(value uint16) bool {
	bm := c.bitmap()
	return bm.Contains(uint32(value))
}

// bitmapShouldConvertToArray returns true if bitmap should be converted to array
func (c *container) bitmapShouldConvertToArray() bool {
	return c.Size <= 4096
}

// bitmapShouldConvertToRun returns true if bitmap should be converted to run
func (c *container) bitmapShouldConvertToRun() bool {
	if c.Size == 0 {
		return false
	}

	numRuns := c.bitmapNumberOfRuns()
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

// bitmapNumberOfRuns counts consecutive runs in the bitmap
// This implements the same logic as the official RoaringBitmap implementation
func (c *container) bitmapNumberOfRuns() int {
	if c.Size == 0 {
		return 0
	}

	bm := c.bitmap()
	numRuns := 0

	// Scan through all 65536 bits to count runs
	inRun := false
	for i := uint32(0); i < 65536; i++ {
		isSet := bm.Contains(i)

		if isSet && !inRun {
			// Start of a new run
			numRuns++
			inRun = true
		} else if !isSet && inRun {
			// End of current run
			inRun = false
		}
	}

	return numRuns
}

// bitmapToArray converts this container from bitmap to array
func (c *container) bitmapToArray() {
	bm := c.bitmap()
	var values []uint16

	// Collect all set bits
	for i := uint32(0); i < 65536; i++ {
		if bm.Contains(i) {
			values = append(values, uint16(i))
		}
	}

	// Create new array data
	c.Data = make([]byte, len(values)*2)
	c.Type = typeArray
	c.Size = uint16(len(values)) // Set cardinality
	array := c.array()
	copy(array, values)
}

// bitmapToRun converts this container from bitmap to run
func (c *container) bitmapToRun() {
	bm := c.bitmap()
	cardinality := c.Size // Preserve cardinality
	var runs []run

	// Find consecutive ranges in the bitmap
	var currentStart uint16 = 0
	var inRun bool = false

	for i := uint32(0); i < 65536; i++ {
		value := uint16(i)
		if bm.Contains(i) {
			if !inRun {
				// Start of new run
				currentStart = value
				inRun = true
			}
			// Continue run
		} else {
			if inRun {
				// End of current run
				runs = append(runs, run{currentStart, value - 1})
				inRun = false
			}
		}
	}

	// Handle case where last run extends to the end
	if inRun {
		runs = append(runs, run{currentStart, 65535})
	}

	// Create new run data
	c.Data = make([]byte, len(runs)*4) // 4 bytes per run (2 uint16s)
	c.Type = typeRun
	c.Size = cardinality // Restore cardinality
	newRuns := c.run()
	copy(newRuns, runs)
}
