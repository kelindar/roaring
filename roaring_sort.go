package roaring

// find16 performs a binary search for the target in the array
// Returns (index, found) where index is the insertion point if not found
func find16(array []uint16, target uint16) (int, bool) {
	const blockSize = 32 // cache line size
	index, n := array, len(array)
	switch {
	case n == 0:
		return 0, false
	case target < index[0]:
		return 0, false
	case target > index[n-1]:
		return n, false
	case target == index[0]:
		return 0, true
	case target == index[n-1]:
		return n - 1, true
	case n <= 16:
		// Simple linear search for small arrays
		for i, key := range index {
			switch {
			case key == target:
				return i, true
			case key > target:
				return i, false
			}
		}
		return n, false
	default:
		// Binary search for the correct block
		numBlocks := (n + blockSize - 1) / blockSize
		left, right := 0, numBlocks-1
		for left <= right {
			mid := left + (right-left)>>1
			blockStart := mid * blockSize
			blockEnd := blockStart + blockSize
			if blockEnd > n {
				blockEnd = n
			}

			switch {
			case target < index[blockStart]:
				right = mid - 1
			case target > index[blockEnd-1]:
				left = mid + 1
			default:
				/*
					var result int64 = -1
						_find16(unsafe.Pointer(&index[blockStart]), hi, unsafe.Pointer(&result), uint64(blockEnd-blockStart))
						if result >= 0 {
							return int(result), true
						}

						return left * blockSize, false
				*/
				return searchBlock(index, blockStart, blockEnd, target)
			}
		}

		return left * blockSize, false
	}
}

// searchBlock performs an optimized linear search within a block
// Returns (index, found) for the key within the block range
func searchBlock(keys []uint16, start, end int, target uint16) (int, bool) {
	for i := start; i < end; {
		remaining := end - i
		switch {
		case remaining >= 4:
			if keys[i] >= target {
				return i, keys[i] == target
			}
			if keys[i+1] >= target {
				return i + 1, keys[i+1] == target
			}
			if keys[i+2] >= target {
				return i + 2, keys[i+2] == target
			}
			if keys[i+3] >= target {
				return i + 3, keys[i+3] == target
			}
			i += 4
		case remaining >= 2:
			if keys[i] >= target {
				return i, keys[i] == target
			}
			if keys[i+1] >= target {
				return i + 1, keys[i+1] == target
			}
			i += 2
		default:
			if keys[i] >= target {
				return i, keys[i] == target
			}
			i++
		}
	}
	return end, false
}
