package roaring

// arrSet sets a value in an array container
func (c *container) arrSet(value uint16) bool {
	idx, exists := find16(c.Data, value)
	if exists {
		return false // Already exists
	}

	// Insert at position idx more efficiently
	oldLen := len(c.Data)
	c.Data = append(c.Data, 0) // Add space for new uint16
	newArray := c.Data

	// Move elements to the right using bulk copy
	if idx < oldLen {
		copy(newArray[idx+1:], c.Data[idx:])
	}

	newArray[idx] = value
	c.Size++
	return true
}

// arrDel removes a value from an array container
func (c *container) arrDel(value uint16) bool {
	idx, exists := find16(c.Data, value)
	if !exists {
		return false
	}

	// Remove element at index idx
	copy(c.Data[idx:], c.Data[idx+1:])
	c.Data = c.Data[:len(c.Data)-1] // Shrink by one uint16
	c.Size--
	return true
}

// arrHas checks if a value exists in an array container
func (c *container) arrHas(value uint16) bool {
	_, exists := find16(c.Data, value)
	return exists
}

// arrOptimize tries to optimize the container
func (c *container) arrOptimize() {
	switch {
	case c.arrIsDense():
		c.arrToRun()
	case c.Size > arrMinSize:
		c.arrToBmp()
	}
}

// arrIsDense quickly estimates if converting to run container would be beneficial
func (c *container) arrIsDense() bool {
	if len(c.Data) < 128 {
		return false
	}

	lo, hi := c.Data[0], c.Data[len(c.Data)-1]
	span := int(hi - lo + 1)
	size := len(c.Data)

	// Quick density filters
	density := float64(size) / float64(span)
	switch {
	case density < 0.1: // Very sparse
		return false
	case density > 0.8: // Very dense
		return true
	}

	// Estimate number of runs using density
	runs := size
	if gap := float64(span) / float64(size); gap < 2.0 {
		runs = int(float64(size) * (1.0 - density*0.7))
	}

	// Check if estimated conversion meets our criteria
	sizeAsArr := size * 2
	sizeAsRun := runs*4 + 2
	return sizeAsRun < sizeAsArr*3/4 && runs <= size/3
}

// arrToRun attempts to convert array to run in a single pass
func (c *container) arrToRun() bool {
	if len(c.Data) == 0 {
		return false
	}
	runsData := make([]uint16, 0, len(c.Data)/2) // estimate runs needed

	// Single iteration: build runs AND count them
	i0 := c.Data[0]
	i1 := c.Data[0]

	for i := 1; i < len(c.Data); i++ {
		if c.Data[i] == i1+1 {
			// Continue current run
			i1 = c.Data[i]
		} else {
			// End current run and start new one
			runsData = append(runsData, i0, i1)
			i0 = c.Data[i]
			i1 = c.Data[i]
		}
	}

	// Add the final run
	runsData = append(runsData, i0, i1)

	// Check conversion criteria with the actual run count
	numRuns := len(runsData) / 2
	sizeAsArray := len(c.Data) * 2
	sizeAsRun := numRuns*4 + 2 // 2 uint16 per run = 4 bytes

	// Only convert if we save at least 25% space and have reasonable compression
	shouldConvert := sizeAsRun < sizeAsArray*3/4 && numRuns <= len(c.Data)/3
	if shouldConvert {
		c.Data = runsData
		c.Type = typeRun
		return true
	}

	return false
}

// arrToBmp converts this container from array to bitmap
func (c *container) arrToBmp() {
	src := c.Data

	// Create bitmap data (65536 bits = 8192 bytes = 4096 uint16s)
	c.Data = make([]uint16, 4096)
	c.Type = typeBitmap
	dst := c.bmp()

	// Use bulk setting for better performance
	for _, value := range src {
		dst.Set(uint32(value))
	}
}
