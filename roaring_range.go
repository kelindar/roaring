package roaring

// Range calls the given function for each value in the bitmap
func (rb *Bitmap) Range(fn func(x uint32)) {
	for _, block := range rb.index {
		if block == nil {
			continue
		}

		for _, c := range block.containers {
			if c == nil {
				continue
			}

			base := uint32(c.Key) << 16
			switch c.Type {
			case typeArray:
				for _, value := range c.Data {
					fn(base | uint32(value))
				}
			case typeBitmap:
				c.bmp().Range(func(value uint32) {
					fn(base | value)
				})
			case typeRun:
				runs := c.run()
				for _, r := range runs {
					for i := r[0]; i <= r[1]; i++ {
						fn(base | uint32(i))
						if i == r[1] {
							break // Prevent overflow
						}
					}
				}
			}
		}
	}
}
