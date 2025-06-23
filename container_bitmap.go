package roaring

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

// bitmapConvertFromArray converts this container from array to bitmap
func (c *container) bitmapConvertFromArray() {
	array := c.array()

	// Create bitmap data (65536 bits = 8192 bytes)
	c.Data = make([]byte, 8192)
	c.Type = typeBitmap
	bm := c.bitmap()

	// Set all bits from the array
	for _, value := range array {
		bm.Set(uint32(value))
	}

	// Update cardinality from bitmap
	c.Size = uint16(bm.Count())
}

// bitmapConvertFromRun converts this container from run to bitmap
func (c *container) bitmapConvertFromRun() {
	runs := c.run()

	// Create bitmap data (65536 bits = 8192 bytes)
	c.Data = make([]byte, 8192)
	c.Type = typeBitmap
	bm := c.bitmap()

	// Set all bits from the runs
	for _, r := range runs {
		for i := r[0]; i <= r[1]; i++ {
			bm.Set(uint32(i))
			if i == r[1] {
				break // Prevent uint16 overflow when r[1] is 65535
			}
		}
	}

	// Update cardinality from bitmap
	c.Size = uint16(bm.Count())
}
