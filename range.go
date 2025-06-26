package roaring

// Range calls the given function for each value in the bitmap
func (rb *Bitmap) Range(fn func(x uint32)) {
	for i := range rb.containers {
		c := &rb.containers[i]
		base := uint32(rb.index[i]) << 16

		switch c.Type {
		case typeArray:
			data := c.Data
			for j := 0; j < len(data); j++ {
				fn(base | uint32(data[j]))
			}

		case typeBitmap:
			c.bmp().Range(func(value uint32) {
				fn(base | value)
			})

		case typeRun:
			runs := c.run()
			for _, r := range runs {
				start, end := uint32(r[0]), uint32(r[1])
				for curr := start; curr <= end; curr++ {
					fn(base | curr)
					if curr == end {
						break // Prevent overflow
					}
				}
			}
		}
	}
}

// Filter iterates over the bitmap elements and calls a predicate provided for each
// containing element. If the predicate returns false, the bitmap at the element's
// position is set to zero.
func (rb *Bitmap) Filter(f func(x uint32) bool) {
	// Collect all values to remove first to avoid modification during iteration
	var toRemove []uint32

	for i := range rb.containers {
		c := &rb.containers[i]
		base := uint32(rb.index[i]) << 16

		switch c.Type {
		case typeArray:
			data := c.Data
			for j := 0; j < len(data); j++ {
				value := base | uint32(data[j])
				if !f(value) {
					toRemove = append(toRemove, value)
				}
			}

		case typeBitmap:
			c.bmp().Range(func(value uint32) {
				fullValue := base | value
				if !f(fullValue) {
					toRemove = append(toRemove, fullValue)
				}
			})

		case typeRun:
			runs := c.run()
			for _, r := range runs {
				start, end := uint32(r[0]), uint32(r[1])
				for curr := start; curr <= end; curr++ {
					value := base | curr
					if !f(value) {
						toRemove = append(toRemove, value)
					}
					if curr == end {
						break // Prevent overflow
					}
				}
			}
		}
	}

	// Remove all values that failed the predicate
	for _, x := range toRemove {
		rb.Remove(x)
	}
}
