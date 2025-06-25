package roaring

// Range calls the given function for each value in the bitmap
func (rb *Bitmap) Range(fn func(x uint32)) {
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

			// Iterate over values in container
			base := uint32(c.Key) << 16
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
		}
	}
}
