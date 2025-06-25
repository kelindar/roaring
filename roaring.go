package roaring

// cblock represents a block of containers for the two-level index
type cblock struct {
	cindex
	content [256]*container // 256 slots for containers with same high 8 bits
}

// Bitmap represents a roaring bitmap for uint32 values
type Bitmap struct {
	cindex
	blocks  [256]*cblock // Level 1: index by high 8 bits of container key
	scratch []uint32
}

// New creates a new empty roaring bitmap
func New() *Bitmap {
	return &Bitmap{}
}

// findContainer finds the container for the given high bits using two-level index
// Returns (container, found) where container is nil if not found
func (rb *Bitmap) findContainer(hi uint16) (*container, bool) {
	hi8, lo8 := uint8(hi>>8), uint8(hi&0xFF)
	block := rb.blocks[hi8]
	if block == nil {
		return nil, false
	}

	if c := block.content[lo8]; c != nil {
		return c, true
	}
	return nil, false
}

// setContainer sets a container at the given high bits
func (rb *Bitmap) setContainer(hi uint16, c *container) {
	c.Key = hi // Set the key in the container
	hi8, lo8 := uint8(hi>>8), uint8(hi&0xFF)

	// Get or create the block
	block := rb.blocks[hi8]
	if block == nil {
		block = &cblock{cindex: cindex{span: [2]uint8{lo8, lo8}}}
		rb.blocks[hi8] = block
		rb.count++
		rb.cindex.update(hi8)
	}

	// Set the container in the block
	if block.content[lo8] == nil {
		block.count++
		block.cindex.update(lo8)
	}
	block.content[lo8] = c
}

// removeContainer removes the container at the given high bits
func (rb *Bitmap) removeContainer(hi uint16) {
	hi8, lo8 := uint8(hi>>8), uint8(hi&0xFF)

	block := rb.blocks[hi8]
	if block == nil {
		return
	}

	if block.content[lo8] != nil {
		block.content[lo8] = nil
		block.count--

		// If block is empty, remove it
		if block.count == 0 {
			rb.blocks[hi8] = nil
			rb.count--

			// Update bitmap-level span if necessary
			if rb.count == 0 {
				return
			} else if hi8 == rb.span[0] || hi8 == rb.span[1] {
				rb.cindex.reload(func(i uint8) bool { return rb.blocks[i] != nil })
			}
			return
		}

		// Update span if necessary
		if lo8 == block.span[0] || lo8 == block.span[1] {
			block.cindex.reload(func(i uint8) bool { return block.content[i] != nil })
		}
	}
}

// Set sets the bit x in the bitmap and grows it if necessary.
func (rb *Bitmap) Set(x uint32) {
	hi, lo := uint16(x>>16), uint16(x&0xFFFF)
	c, exists := rb.findContainer(hi)
	if !exists {
		c = &container{
			Type: typeArray,
			Size: 0,                     // Start with zero cardinality
			Data: make([]uint16, 0, 64), // Start empty with some capacity (64 uint16s = 128 bytes)
		}
		rb.setContainer(hi, c)
	}

	c.set(lo)
}

// Remove removes the bit x from the bitmap, but does not shrink it.
func (rb *Bitmap) Remove(x uint32) {
	hi, lo := uint16(x>>16), uint16(x&0xFFFF)
	c, exists := rb.findContainer(hi)
	if !exists || !c.remove(lo) {
		return
	}

	if c.isEmpty() {
		rb.removeContainer(hi)
	}
}

// Contains checks whether a value is contained in the bitmap or not.
func (rb *Bitmap) Contains(x uint32) bool {
	hi, lo := uint16(x>>16), uint16(x&0xFFFF)
	c, exists := rb.findContainer(hi)
	if !exists {
		return false
	}

	return c.contains(lo)
}

// Count returns the total number of bits set to 1 in the bitmap
func (rb *Bitmap) Count() int {
	if rb.count == 0 {
		return 0
	}

	count := 0
	for i := int(rb.span[0]); i <= int(rb.span[1]); i++ {
		block := rb.blocks[i]
		if block != nil {
			for j := int(block.span[0]); j <= int(block.span[1]); j++ {
				c := block.content[j]
				if c != nil {
					count += c.cardinality()
				}
			}
		}
	}
	return count
}

// Clear clears the bitmap and resizes it to zero.
func (rb *Bitmap) Clear() {
	rb.blocks = [256]*cblock{} // Clear all blocks
	rb.count = 0
	rb.span[0] = 0
	rb.span[1] = 0
}

// Optimize optimizes all containers to use the most efficient representation.
// This can significantly reduce memory usage, especially after bulk operations.
func (rb *Bitmap) Optimize() {
	if rb.count == 0 {
		return
	}

	for i := int(rb.span[0]); i <= int(rb.span[1]); i++ {
		block := rb.blocks[i]
		if block != nil {
			for j := int(block.span[0]); j <= int(block.span[1]); j++ {
				c := block.content[j]
				if c != nil {
					c.optimize()
				}
			}
		}
	}
}

// Grow grows the bitmap size until we reach the desired bit.
func (rb *Bitmap) Grow(desiredBit uint32) {
	panic("not implemented")
}

// Clone clones the bitmap
func (rb *Bitmap) Clone(into *Bitmap) *Bitmap {
	panic("not implemented")
}

// ---------------------------------------- Index ----------------------------------------

type cindex struct {
	span  [2]uint8
	count int
}

// update updates the span when adding a new index
func (idx *cindex) update(newIdx uint8) {
	switch idx.count {
	case 1:
		idx.span[0] = newIdx
		idx.span[1] = newIdx
	default:
		if newIdx < idx.span[0] {
			idx.span[0] = newIdx
		}
		if newIdx > idx.span[1] {
			idx.span[1] = newIdx
		}
	}
}

// reload recalculates span by scanning for non-nil items
func (idx *cindex) reload(has func(uint8) bool) {
	if idx.count == 0 {
		return
	}

	// Find new min
	for i := 0; i < 256; i++ {
		if has(uint8(i)) {
			idx.span[0] = uint8(i)
			break
		}
	}

	// Find new max
	for i := 255; i >= 0; i-- {
		if has(uint8(i)) {
			idx.span[1] = uint8(i)
			break
		}
	}
}
