package roaring

const (
	arrMinSize    = 2048
	runMinSize    = 100
	optimizeEvery = 2048
)

type ctype byte

const (
	typeArray ctype = iota
	typeBitmap
	typeRun
)

type container struct {
	Key    uint16   // High 16 bits of the value range this container handles
	Type   ctype    // Type of the container
	Call   uint16   // Call count
	Size   uint32   // Cardinality
	Data   []uint16 // Data of the container
	shared bool     // COW: true if data is shared between containers
}

type run [2]uint16

// cowEnsureOwned ensures the container owns its data before modification
// If data is shared, creates a copy; otherwise does nothing
func (c *container) cowEnsureOwned() {
	if c.shared {
		newData := make([]uint16, len(c.Data))
		copy(newData, c.Data)
		c.Data = newData
		c.shared = false
	}
}

// cowClone creates a shallow copy of the container with shared data
func (c *container) cowClone() *container {
	clone := &container{
		Key:    c.Key,
		Type:   c.Type,
		Call:   c.Call,
		Size:   c.Size,
		Data:   c.Data, // Share the data
		shared: true,   // Mark as shared
	}
	c.shared = true // Original is now also shared
	return clone
}

// set sets a value in the container and returns true if the value was added (didn't exist before)
func (c *container) set(value uint16) (ok bool) {
	c.cowEnsureOwned() // Ensure we own the data before modifying
	switch c.Type {
	case typeArray:
		if ok = c.arrSet(value); ok {
			c.tryOptimize()
		}
	case typeBitmap:
		if ok = c.bmpSet(value); ok {
			c.tryOptimize()
		}
	case typeRun:
		if ok = c.runSet(value); ok {
			c.tryOptimize()
		}
	}
	return
}

// remove removes a value from the container and returns true if the value was removed (existed before)
func (c *container) remove(value uint16) (ok bool) {
	c.cowEnsureOwned() // Ensure we own the data before modifying
	switch c.Type {
	case typeArray:
		if ok = c.arrDel(value); ok {
			c.tryOptimize()
		}
	case typeBitmap:
		if ok = c.bmpDel(value); ok {
			c.tryOptimize()
		}
	case typeRun:
		if ok = c.runDel(value); ok {
			c.tryOptimize()
		}
	}
	return
}

// contains checks if a value exists in the container
func (c *container) contains(value uint16) bool {
	switch c.Type {
	case typeArray:
		return c.arrHas(value)
	case typeBitmap:
		return c.bmpHas(value)
	case typeRun:
		return c.runHas(value)
	}
	return false
}

// cardinality returns the number of elements in the container
func (c *container) cardinality() int {
	return int(c.Size)
}

// isEmpty returns true if the container has no elements
func (c *container) isEmpty() bool {
	return c.cardinality() == 0
}

// optimize converts the container to the most efficient representation
func (c *container) optimize() {
	c.cowEnsureOwned() // Ensure we own the data before modifying
	switch c.Type {
	case typeArray:
		c.arrOptimize()
	case typeBitmap:
		c.bmpOptimize()
	case typeRun:
		c.runOptimize()
	}
}

// tryOptimize optimizes the container periodically
func (c *container) tryOptimize() {
	if c.Call++; c.Call%optimizeEvery == 0 {
		c.optimize()
	}
}
