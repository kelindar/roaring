package roaring

// bitmapSet sets a value in a bitmap container
func (c *container) bitmapSet(value uint16) bool {
	bm := c.bitmap()
	if bm.Contains(uint32(value)) {
		return false // Already exists
	}
	bm.Set(uint32(value))
	c.Size++ // Increment cardinality
	return true
}

// bitmapRemove removes a value from a bitmap container
func (c *container) bitmapRemove(value uint16) bool {
	bm := c.bitmap()
	if bm.Contains(uint32(value)) {
		bm.Remove(uint32(value))
		c.Size-- // Decrement cardinality
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
	numRuns := c.bitmapNumberOfRuns()
	cardinality := int(c.Size)

	// Estimated size as run container (each run takes 4 bytes + 2 bytes header)
	sizeAsRunContainer := 2 + numRuns*4

	// Size as bitmap container (always 8192 bytes)
	sizeAsBitmapContainer := 8192

	// Size as array container (2 bytes per element)
	sizeAsArrayContainer := cardinality * 2

	// Convert to run if it's smaller than both bitmap and array representations
	return sizeAsRunContainer < sizeAsBitmapContainer && sizeAsRunContainer < sizeAsArrayContainer
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
	cardinality := c.Size // Preserve cardinality

	// Create bitmap data (65536 bits = 8192 bytes)
	c.Data = make([]byte, 8192)
	c.Type = typeBitmap
	c.Size = cardinality // Restore cardinality
	bm := c.bitmap()

	// Set all bits from the array
	for _, value := range array {
		bm.Set(uint32(value))
	}
}

// bitmapConvertFromRun converts this container from run to bitmap
func (c *container) bitmapConvertFromRun() {
	runs := c.run()
	cardinality := c.Size // Preserve cardinality

	// Create bitmap data (65536 bits = 8192 bytes)
	c.Data = make([]byte, 8192)
	c.Type = typeBitmap
	c.Size = cardinality // Restore cardinality
	bm := c.bitmap()

	// Set all bits from the runs
	for _, r := range runs {
		for i := r[0]; i <= r[1]; i++ {
			bm.Set(uint32(i))
		}
	}
}
