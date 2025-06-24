package roaring

const (
	arrMinSize    = 2048 // Reduced from 4096 to convert to bitmap sooner for better delete performance
	runMinSize    = 100
	optimizeEvery = 2048 // calls - reduced frequency to minimize overhead
)

type ctype byte

const (
	typeArray ctype = iota
	typeBitmap
	typeRun
)

type container struct {
	Type ctype  // Type of the container
	Call uint16 // Call count
	Size uint32 // Cardinality
	Data []byte // Data of the container
}

type run [2]uint16

// set sets a value in the container and returns true if the value was added (didn't exist before)
func (c *container) set(value uint16) (ok bool) {
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
