package roaring

// Range calls the given function for each value in the bitmap
func (rb *Bitmap) Range(fn func(x uint32)) {
	rb.containers(func(base uint32, c *container) {
		switch c.Type {
		case typeArray:
			data := c.Data
			for i := 0; i < len(data); i++ {
				fn(base | uint32(data[i]))
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
	})
}

// Filter iterates over the bitmap elements and calls a predicate provided for each
// containing element. If the predicate returns false, the bitmap at the element's
// position is set to zero.
func (rb *Bitmap) Filter(f func(x uint32) bool) {
	rb.containers(func(base uint32, c *container) {
		rb.scratch = rb.scratch[:0]
		c.cowEnsureOwned()

		switch c.Type {
		case typeArray:
			data := c.Data
			for i := 0; i < len(data); i++ {
				if !f(base | uint32(data[i])) {
					rb.scratch = append(rb.scratch, base|uint32(data[i]))
				}
			}

		case typeBitmap:
			bmp := c.bmp()
			bmp.Filter(func(value uint32) bool {
				return f(base | value)
			})

		case typeRun:
			runs := c.run()
			for _, r := range runs {
				start, end := uint32(r[0]), uint32(r[1])
				for curr := start; curr <= end; curr++ {
					if !f(base | curr) {
						rb.scratch = append(rb.scratch, base|curr)
					}
				}
			}
		}

		// Remove all values that failed the predicate for this container
		for _, x := range rb.scratch {
			rb.Remove(x)
		}
	})
}

func (rb *Bitmap) containers(fn func(base uint32, c *container)) {
	if rb.count == 0 {
		return
	}

	// Iterate over blocks
	for i := int(rb.span[0]); i <= int(rb.span[1]); i++ {
		block := rb.blocks[i]
		if block == nil {
			continue
		}

		// Iterate over containers in block
		for j := int(block.span[0]); j <= int(block.span[1]); j++ {
			c := block.content[j]
			if c == nil {
				continue
			}

			fn(uint32(c.Key)<<16, c)
		}
	}
}
