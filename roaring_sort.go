package roaring

// find16 returns the first index whose value is ≥ target.
// If the value equals target, found == true.
// If not found, index is the insertion point to keep the slice sorted.
func find16(a []uint16, target uint16) (index int, found bool) {
	n := len(a)
	if n == 0 {
		return 0, false
	}

	// quick exits for extreme keys
	if target <= a[0] {
		return 0, target == a[0]
	}
	if target > a[n-1] {
		return n, false
	}

	// binary phase: shrink search window to ≤16
	lo, hi := 0, n // hi is _exclusive_
	for hi-lo > 16 {
		mid := (lo + hi) >> 1
		if a[mid] < target {
			lo = mid + 1
		} else {
			hi = mid // keep mid in the candidate range
		}
	}

	// linear phase inside one cache line
	i := lo
	for ; i+3 < hi; i += 4 { // 4-way unroll
		if a[i] >= target {
			return i, a[i] == target
		}
		if a[i+1] >= target {
			return i + 1, a[i+1] == target
		}
		if a[i+2] >= target {
			return i + 2, a[i+2] == target
		}
		if a[i+3] >= target {
			return i + 3, a[i+3] == target
		}
	}

	// 0-3 leftovers
	for ; i < hi; i++ {
		if a[i] >= target {
			return i, a[i] == target
		}
	}

	// hi is now the first position that may still satisfy ≥ target
	return hi, hi < n && a[hi] == target
}
