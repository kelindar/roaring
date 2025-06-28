// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package roaring

// Bitmap represents a roaring bitmap for uint32 values
type Bitmap struct {
	containers []container // Containers in sorted order by key
	index      []uint16    // Container keys for cache-efficient searching
	scratch    []uint16
}

// New creates a new empty roaring bitmap
func New() *Bitmap {
	return &Bitmap{}
}

// Set sets the bit x in the bitmap and grows it if necessary.
func (rb *Bitmap) Set(x uint32) {
	hi, lo := uint16(x>>16), uint16(x&0xFFFF)
	idx, exists := find16(rb.index, hi)
	if !exists {
		rb.ctrAdd(hi, idx, &container{
			Type: typeArray,
			Size: 0,
			Data: make([]uint16, 0, 64),
		})
	}
	rb.containers[idx].set(lo)
}

// Remove removes the bit x from the bitmap
func (rb *Bitmap) Remove(x uint32) {
	hi, lo := uint16(x>>16), uint16(x&0xFFFF)
	idx, exists := find16(rb.index, hi)
	if !exists || !rb.containers[idx].remove(lo) {
		return
	}

	if rb.containers[idx].isEmpty() {
		rb.ctrDel(idx)
	}
}

// Contains checks whether a value is contained in the bitmap
func (rb *Bitmap) Contains(x uint32) bool {
	hi, lo := uint16(x>>16), uint16(x&0xFFFF)
	idx, exists := find16(rb.index, hi)
	if !exists {
		return false
	}

	return rb.containers[idx].contains(lo)
}

// Count returns the total number of bits set to 1 in the bitmap
func (rb *Bitmap) Count() int {
	count := 0
	for i := range rb.containers {
		count += int(rb.containers[i].Size)
	}
	return count
}

// Clear clears the bitmap
func (rb *Bitmap) Clear() {
	rb.containers = rb.containers[:0]
	rb.index = rb.index[:0]
}

// Optimize optimizes all containers to use the most efficient representation
func (rb *Bitmap) Optimize() {
	for i := range rb.containers {
		rb.containers[i].optimize()
	}
}

// Clone clones the bitmap
func (rb *Bitmap) Clone(into *Bitmap) *Bitmap {
	if into == nil {
		into = New()
	}

	// Clone containers
	if cap(into.containers) < len(rb.containers) {
		into.containers = make([]container, len(rb.containers), cap(rb.containers))
	}
	into.containers = into.containers[:len(rb.containers)]
	for i := range rb.containers {
		rb.containers[i].Shared = true
	}
	copy(into.containers, rb.containers)

	// Clone index
	if cap(into.index) < len(rb.index) {
		into.index = make([]uint16, len(rb.index), cap(rb.index))
	}

	into.index = into.index[:len(rb.index)]
	copy(into.index, rb.index)
	return into
}

// And performs bitwise AND operation with other bitmap(s)
func (rb *Bitmap) And(other *Bitmap, extra ...*Bitmap) {
	rb.and(other)
	for _, bm := range extra {
		if bm != nil {
			rb.and(bm)
		}
	}
}

// AndNot performs bitwise AND NOT operation with other bitmap(s)
func (rb *Bitmap) AndNot(other *Bitmap, extra ...*Bitmap) {
	rb.andNot(other)
	for _, bm := range extra {
		if bm != nil {
			rb.andNot(bm)
		}
	}
}

// Or performs bitwise OR operation with other bitmap(s)
func (rb *Bitmap) Or(other *Bitmap, extra ...*Bitmap) {
	rb.or(other)
	for _, bm := range extra {
		if bm != nil {
			rb.or(bm)
		}
	}
}

// Xor performs bitwise XOR operation with other bitmap(s)
func (rb *Bitmap) Xor(other *Bitmap, extra ...*Bitmap) {
	rb.xor(other)
	for _, bm := range extra {
		if bm != nil {
			rb.xor(bm)
		}
	}
}

// Min get the smallest value stored in this bitmap, assuming the bitmap is not empty.
func (rb *Bitmap) Min() (uint32, bool) {
	for i := 0; i < len(rb.containers); i++ {
		if min, ok := rb.containers[i].min(); ok {
			return uint32(rb.index[i])<<16 | uint32(min), true
		}
	}
	return 0, false
}

// Max get the largest value stored in this bitmap, assuming the bitmap is not empty.
func (rb *Bitmap) Max() (uint32, bool) {
	for i := len(rb.containers) - 1; i >= 0; i-- {
		if max, ok := rb.containers[i].max(); ok {
			return uint32(rb.index[i])<<16 | uint32(max), true
		}
	}
	return 0, false
}

// MinZero finds the first zero bit and returns its index, assuming the bitmap is not empty.
func (rb *Bitmap) MinZero() (uint32, bool) {
	// Check if position 0 is unset (before first container or within first container)
	if len(rb.containers) == 0 || rb.index[0] > 0 {
		return 0, true
	}

	// Check within first container
	if minZero, ok := rb.containers[0].minZero(); ok {
		return uint32(rb.index[0])<<16 | uint32(minZero), true
	}

	// Check gaps between containers
	for i := 0; i < len(rb.containers)-1; i++ {
		currentHi := rb.index[i]
		nextHi := rb.index[i+1]

		// If there's a gap between containers
		if nextHi > currentHi+1 {
			return uint32(currentHi+1) << 16, true
		}

		// Check within the next container
		if minZero, ok := rb.containers[i+1].minZero(); ok {
			return uint32(nextHi)<<16 | uint32(minZero), true
		}
	}

	// Check after last container
	if len(rb.containers) > 0 {
		lastHi := rb.index[len(rb.containers)-1]
		if lastHi < 65535 {
			return uint32(lastHi+1) << 16, true
		}
	}

	return 0, false // No zero bits found
}

// ---------------------------------------- Container ----------------------------------------

// ctrAdd inserts a container at the given position
func (rb *Bitmap) ctrAdd(hi uint16, pos int, c *container) {
	// Insert new container at position to maintain order
	rb.containers = append(rb.containers, container{})
	if pos < len(rb.containers)-1 {
		copy(rb.containers[pos+1:], rb.containers[pos:len(rb.containers)-1])
	}
	rb.containers[pos] = *c

	// Keep index in sync
	rb.index = append(rb.index, 0)
	if pos < len(rb.index)-1 {
		copy(rb.index[pos+1:], rb.index[pos:len(rb.index)-1])
	}
	rb.index[pos] = hi
}

// ctrDel removes the container at the given position
func (rb *Bitmap) ctrDel(pos int) {
	if pos < 0 || pos >= len(rb.containers) {
		return
	}

	// Remove container by shifting slice
	copy(rb.containers[pos:], rb.containers[pos+1:])
	rb.containers = rb.containers[:len(rb.containers)-1]

	// Keep index in sync
	copy(rb.index[pos:], rb.index[pos+1:])
	rb.index = rb.index[:len(rb.index)-1]
}

// find16 returns the first index whose value is ≥ target.
// If the value equals target, found == true.
// If not found, index is the insertion point to keep the slice sorted.
//
//go:nosplit
func find16(a []uint16, target uint16) (index int, found bool) {
	n := len(a)
	switch {
	case n == 0:
		return 0, false
	case target <= a[0]:
		return 0, target == a[0]
	case target > a[n-1]:
		return n, false
	}

	// binary phase: shrink search window to ≤16
	lo, hi := 0, n
	for hi-lo > 16 {
		mid := (lo + hi) >> 1
		switch {
		case a[mid] < target:
			lo = mid + 1
		case a[mid] >= target:
			hi = mid // keep mid in the candidate range
		}
	}

	// linear phase inside one cache line
	i := lo
	for ; i+3 < hi; i += 4 { // 4-way unroll
		switch {
		case a[i] >= target:
			return i, a[i] == target
		case a[i+1] >= target:
			return i + 1, a[i+1] == target
		case a[i+2] >= target:
			return i + 2, a[i+2] == target
		case a[i+3] >= target:
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
