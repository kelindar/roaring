package roaring

const (
	arrMinSize    = 2048
	runMinSize    = 100
	runMaxSize    = 2048
	optimizeEvery = 2048
)

type ctype byte

const (
	typeArray ctype = iota
	typeBitmap
	typeRun
)

type container struct {
	Type   ctype    // Type of the container
	Shared bool     // COW: true if data is shared between containers
	Call   uint16   // Call count
	Size   uint32   // Cardinality
	Data   []uint16 // Data of the container
}

type run [2]uint16

// cowEnsureOwned ensures the container owns its data before modification
func (c *container) cowEnsureOwned() {
	if c.Shared {
		clone := make([]uint16, len(c.Data), cap(c.Data))
		copy(clone, c.Data)
		c.Data = clone
		c.Shared = false
	}
}

// cowClone creates a copy-on-write clone of the container
func (c *container) cowClone() *container {
	clone := &container{
		Type:   c.Type,
		Call:   c.Call,
		Size:   c.Size,
		Data:   c.Data,
		Shared: true,
	}

	c.Shared = true
	return clone
}

// set sets a value in the container and returns true if the value was added (didn't exist before)
func (c *container) set(value uint16) (ok bool) {
	c.cowEnsureOwned()
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

// isEmpty returns true if the container has no elements
func (c *container) isEmpty() bool {
	return c.Size == 0
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
