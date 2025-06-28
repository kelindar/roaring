package roaring

// Range calls the given function for each value in the bitmap
func (rb *Bitmap) Range(fn func(x uint32) bool) {
	for i := range rb.containers {
		c := &rb.containers[i]
		base := uint32(rb.index[i]) << 16

		switch c.Type {
		case typeArray:
			data := c.Data
			for j := 0; j < len(data); j++ {
				if !fn(base | uint32(data[j])) {
					return
				}
			}

		case typeBitmap:
			c.bmpRange(func(value uint32) bool {
				return fn(base | value)
			})

		case typeRun:
			numRuns := len(c.Data) / 2
			for i := 0; i < numRuns; i++ {
				start, end := uint32(c.Data[i*2]), uint32(c.Data[i*2+1])
				for curr := start; curr <= end; curr++ {
					if !fn(base | curr) {
						return
					}

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
			numRuns := len(c.Data) / 2
			for i := 0; i < numRuns; i++ {
				start, end := uint32(c.Data[i*2]), uint32(c.Data[i*2+1])
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

// Iterate iterates over all of the bits set to one in this bitmap.
func (c *container) bmpRange(fn func(x uint32) bool) {
	dst := c.bmp()
	for blkAt := 0; blkAt < len(dst); blkAt++ {
		blk := dst[blkAt]
		if blk == 0x0 {
			continue // Skip the empty page
		}

		// Iterate in a 4-bit chunks so we can reduce the number of function calls and skip
		// the bits for which we should not call our range function.
		offset := uint32(blkAt << 6)
		for ; blk > 0; blk = blk >> 4 {
			switch blk & 0b1111 {
			case 0b0001:
				if !fn(offset + 0) {
					return
				}
			case 0b0010:
				if !fn(offset + 1) {
					return
				}
			case 0b0011:
				if !fn(offset + 0) {
					return
				}
				if !fn(offset + 1) {
					return
				}
			case 0b0100:
				if !fn(offset + 2) {
					return
				}
			case 0b0101:
				if !fn(offset + 0) {
					return
				}
				if !fn(offset + 2) {
					return
				}
			case 0b0110:
				if !fn(offset + 1) {
					return
				}
				if !fn(offset + 2) {
					return
				}
			case 0b0111:
				if !fn(offset + 0) {
					return
				}
				if !fn(offset + 1) {
					return
				}
				if !fn(offset + 2) {
					return
				}
			case 0b1000:
				if !fn(offset + 3) {
					return
				}
			case 0b1001:
				if !fn(offset + 0) {
					return
				}
				if !fn(offset + 3) {
					return
				}
			case 0b1010:
				if !fn(offset + 1) {
					return
				}
				if !fn(offset + 3) {
					return
				}
			case 0b1011:
				if !fn(offset + 0) {
					return
				}
				if !fn(offset + 1) {
					return
				}
				if !fn(offset + 3) {
					return
				}
			case 0b1100:
				if !fn(offset + 2) {
					return
				}
				if !fn(offset + 3) {
					return
				}
			case 0b1101:
				if !fn(offset + 0) {
					return
				}
				if !fn(offset + 2) {
					return
				}
				if !fn(offset + 3) {
					return
				}
			case 0b1110:
				if !fn(offset + 1) {
					return
				}
				if !fn(offset + 2) {
					return
				}
				if !fn(offset + 3) {
					return
				}
			case 0b1111:
				if !fn(offset + 0) {
					return
				}
				if !fn(offset + 1) {
					return
				}
				if !fn(offset + 2) {
					return
				}
				if !fn(offset + 3) {
					return
				}
			}
			offset += 4
		}
	}
}
