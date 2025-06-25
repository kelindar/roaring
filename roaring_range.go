package roaring

// Range calls the given function for each value in the bitmap
func (rb *Bitmap) Range(fn func(x uint32)) {
	// Fast path for empty bitmap
	if rb.count == 0 {
		return
	}

	// Optimized iteration using spans to skip nil blocks and containers
	for i := int(rb.span[0]); i <= int(rb.span[1]); i++ {
		block := rb.blocks[i]
		if block == nil {
			continue
		}

		// Use span to avoid checking nil containers
		for j := int(block.span[0]); j <= int(block.span[1]); j++ {
			c := block.content[j]
			if c == nil {
				continue
			}

			// Pre-compute base value once per container
			base := uint32(c.Key) << 16

			// Optimized container iteration
			switch c.Type {
			case typeArray:
				// Optimized array iteration - critical for dense case
				data := c.Data
				for i := 0; i < len(data); i++ {
					fn(base | uint32(data[i]))
				}

			case typeBitmap:
				// Delegate to optimized bitmap.Range
				c.bmp().Range(func(value uint32) {
					fn(base | value)
				})

			case typeRun:
				// Optimized run iteration
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
}
