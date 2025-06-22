package roaring

// set sets a value in the container and returns true if the value was added (didn't exist before)
func (c *container) set(value uint16) bool {
	switch c.Type {
	case typeArray:
		modified := c.arraySet(value)
		if modified && c.arrayShouldConvertToBitmap() {
			c.bitmapConvertFromArray()
		}
		// Note: arrayShouldConvertToRun() returns false for now
		return modified
	case typeBitmap:
		modified := c.bitmapSet(value)
		if modified && c.bitmapShouldConvertToArray() {
			c.arrayConvertFromBitmap()
		}
		return modified
	case typeRun:
		return c.runSet(value)
	}
	return false
}

// remove removes a value from the container and returns true if the value was removed (existed before)
func (c *container) remove(value uint16) bool {
	switch c.Type {
	case typeArray:
		modified := c.arrayRemove(value)
		// Note: arrayShouldConvertToRun() returns false for now
		return modified
	case typeBitmap:
		modified := c.bitmapRemove(value)
		if modified && c.bitmapShouldConvertToArray() {
			c.arrayConvertFromBitmap()
		}
		// Note: bitmapShouldConvertToRun() returns false for now
		return modified
	case typeRun:
		modified := c.runRemove(value)
		if modified && c.runShouldConvert() {
			c.bitmapConvertFromRun()
		}
		return modified
	}
	return false
}

// contains checks if a value exists in the container
func (c *container) contains(value uint16) bool {
	switch c.Type {
	case typeArray:
		return c.arrayContains(value)
	case typeBitmap:
		return c.bitmapContains(value)
	case typeRun:
		return c.runContains(value)
	}
	return false
}

// cardinality returns the number of elements in the container
func (c *container) cardinality() int {
	return int(c.Size)
}

// isEmpty returns true if the container has no elements
func (c *container) isEmpty() bool {
	return c.cardinality() == 0
}

// runOptimize converts the container to run format if it would be more efficient
// This method analyzes the current container and converts it to the most efficient representation
func (c *container) runOptimize() {
	switch c.Type {
	case typeArray:
		// Array -> Bitmap -> Run (if beneficial)
		if c.arrayShouldConvertToBitmap() {
			c.bitmapConvertFromArray()
			if c.bitmapShouldConvertToRun() {
				c.runConvertFromBitmap()
			}
		}
	case typeBitmap:
		if c.bitmapShouldConvertToRun() {
			c.runConvertFromBitmap()
		} else if c.bitmapShouldConvertToArray() {
			c.arrayConvertFromBitmap()
		}
	case typeRun:
		// Already a run container, check if it should convert to something else for efficiency
		if c.runShouldConvert() {
			c.bitmapConvertFromRun()
			// After converting to bitmap, check if array would be better
			if c.bitmapShouldConvertToArray() {
				c.arrayConvertFromBitmap()
			}
		}
	}
}
