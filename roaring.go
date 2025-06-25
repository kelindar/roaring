package roaring

// cblock represents a block of containers using the new block structure
type cblock = block[container]

// Bitmap represents a roaring bitmap for uint32 values
type Bitmap struct {
	blocks  *block[cblock] // Top-level block for container blocks
	scratch []uint32
}

// New creates a new empty roaring bitmap
func New() *Bitmap {
	return &Bitmap{
		blocks: newBlock[cblock](),
	}
}

// Set sets the bit x in the bitmap and grows it if necessary.
func (rb *Bitmap) Set(x uint32) {
	hi, lo := uint16(x>>16), uint16(x&0xFFFF)
	c, exists := rb.ctrFind(hi)
	if !exists {
		c = &container{
			Type: typeArray,
			Size: 0,
			Data: make([]uint16, 0, 64),
		}
		rb.ctrAdd(hi, c)
	}

	c.set(lo)
}

// Remove removes the bit x from the bitmap
func (rb *Bitmap) Remove(x uint32) {
	hi, lo := uint16(x>>16), uint16(x&0xFFFF)
	c, exists := rb.ctrFind(hi)
	if !exists || !c.remove(lo) {
		return
	}

	if c.isEmpty() {
		rb.ctrDel(hi)
	}
}

// Contains checks whether a value is contained in the bitmap
func (rb *Bitmap) Contains(x uint32) bool {
	hi, lo := uint16(x>>16), uint16(x&0xFFFF)
	c, exists := rb.ctrFind(hi)
	if !exists {
		return false
	}

	return c.contains(lo)
}

// Count returns the total number of bits set to 1 in the bitmap
func (rb *Bitmap) Count() int {
	if rb.blocks.isEmpty() {
		return 0
	}

	count := 0
	rb.blocks.iterate(func(hi8 uint8, cblk *cblock) {
		cblk.iterate(func(lo8 uint8, c *container) {
			count += c.cardinality()
		})
	})
	return count
}

// Clear clears the bitmap
func (rb *Bitmap) Clear() {
	rb.blocks = newBlock[cblock]()
}

// Optimize optimizes all containers to use the most efficient representation
func (rb *Bitmap) Optimize() {
	if rb.blocks.isEmpty() {
		return
	}

	rb.blocks.iterate(func(hi8 uint8, cblk *cblock) {
		cblk.iterate(func(lo8 uint8, c *container) {
			c.optimize()
		})
	})
}

// Clone clones the bitmap
func (rb *Bitmap) Clone(into *Bitmap) *Bitmap {
	if into == nil {
		into = &Bitmap{
			blocks: newBlock[cblock](),
		}
	}

	// Clone all blocks and containers
	rb.blocks.iterate(func(hi8 uint8, cblk *cblock) {
		newCblk := newBlock[container]()
		into.blocks.set(hi8, newCblk)

		cblk.iterate(func(lo8 uint8, c *container) {
			newCblk.set(lo8, c.cowClone())
		})
	})

	return into
}

// ---------------------------------------- Container ----------------------------------------

// ctrFind finds the container for the given high bits
func (rb *Bitmap) ctrFind(hi uint16) (*container, bool) {
	hi8, lo8 := uint8(hi>>8), uint8(hi&0xFF)
	cblk := rb.blocks.get(hi8)
	if cblk == nil {
		return nil, false
	}

	container := cblk.get(lo8)
	if container == nil {
		return nil, false
	}

	return container, true
}

// ctrAdd sets a container at the given high bits
func (rb *Bitmap) ctrAdd(hi uint16, c *container) {
	hi8, lo8 := uint8(hi>>8), uint8(hi&0xFF)

	// Get or create the container block
	cblk := rb.blocks.get(hi8)
	if cblk == nil {
		cblk = newBlock[container]()
		rb.blocks.set(hi8, cblk)
	}

	// Set the container
	cblk.set(lo8, c)
}

// ctrDel removes the container at the given high bits
func (rb *Bitmap) ctrDel(hi uint16) {
	hi8, lo8 := uint8(hi>>8), uint8(hi&0xFF)
	cblk := rb.blocks.get(hi8)
	if cblk == nil {
		return
	}

	// Remove the container
	cblk.del(lo8)

	// If block is empty, remove it
	if cblk.isEmpty() {
		rb.blocks.del(hi8)
	}
}

// containers iterates over all containers in the bitmap
func (rb *Bitmap) containers(fn func(base uint32, c *container)) {
	rb.blocks.iterate(func(hi8 uint8, cblk *cblock) {
		cblk.iterate(func(lo8 uint8, c *container) {
			fn((uint32(hi8)<<24)|uint32(lo8)<<16, c)
		})
	})
}

// ---------------------------------------- Block ----------------------------------------

// block represents a sparse array with 256 slots using a fill bitmap
type block[T any] struct {
	span  [2]uint8 // Min/max index for quick iteration
	count uint16   // Number of occupied slots
	data  [256]*T  // 256 slots for data
}

// newBlock creates a new empty block
func newBlock[T any]() *block[T] {
	return &block[T]{}
}

// get retrieves the value at the specified index
func (b *block[T]) get(idx uint8) *T {
	return b.data[idx]
}

// set sets a value at the specified index
func (b *block[T]) set(idx uint8, value *T) {
	wasEmpty := b.isEmpty()
	hadValue := b.has(idx)

	b.data[idx] = value
	if !hadValue {
		b.count++
	}

	switch {
	case wasEmpty:
		b.span[0] = idx
		b.span[1] = idx
	case idx < b.span[0]:
		b.span[0] = idx
	case idx > b.span[1]:
		b.span[1] = idx
	}
}

// del removes a value from the block
func (b *block[T]) del(idx uint8) {
	if !b.has(idx) {
		return
	}

	b.data[idx] = nil
	b.count--
	switch {
	case b.count == 0:
		b.span[0] = 0
		b.span[1] = 0
	case idx == b.span[0] || idx == b.span[1]:
		b.updateSpan()
	}
}

// has checks if there's a value at the specified index
func (b *block[T]) has(idx uint8) bool {
	return b.data[idx] != nil
}

// isEmpty checks if the block has no values
func (b *block[T]) isEmpty() bool {
	return b.count == 0
}

// updateSpan recalculates the span by finding first and last set bits
func (b *block[T]) updateSpan() {
	if b.count == 0 {
		b.span[0] = 0
		b.span[1] = 0
		return
	}

	// Find first set
	for i := 0; i < 256; i++ {
		if b.data[i] != nil {
			b.span[0] = uint8(i)
			break
		}
	}

	// Find last set
	for i := 255; i >= 0; i-- {
		if b.data[i] != nil {
			b.span[1] = uint8(i)
			break
		}
	}
}

// iterate calls fn for each non-nil value in the block
func (b *block[T]) iterate(fn func(idx uint8, value *T)) {
	var ct uint16
	for i := int(b.span[0]); i <= int(b.span[1]) && ct < b.count; i++ {
		if b.data[i] != nil {
			fn(uint8(i), b.data[i])
			ct++
		}
	}
}
