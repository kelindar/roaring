package roaring

const (
	arrMinSize = 4096
	runMinSize = 100
)

type ctype byte

const (
	typeArray ctype = iota
	typeBitmap
	typeRun
)

type container struct {
	Type ctype  // Type of the container
	Size uint16 // Cardinality
	Data []byte // Data of the container
}

type run [2]uint16

// set sets a value in the container and returns true if the value was added (didn't exist before)
func (c *container) set(value uint16) (ok bool) {
	switch c.Type {
	case typeArray:
		if ok = c.arrSet(value); ok && c.arrShouldConvertToBitmap() {
			c.arrToBmp()
		}
		return
	case typeBitmap:
		if ok = c.bmpSet(value); ok && c.bmpShouldConvertToArray() {
			c.bmpToArr()
		}
		return
	case typeRun:
		if ok = c.runSet(value); ok && len(c.run()) > runMinSize {
			c.runToBmp()
		}
	}
	return false
}

// remove removes a value from the container and returns true if the value was removed (existed before)
func (c *container) remove(value uint16) (ok bool) {
	switch c.Type {
	case typeArray:
		return c.arrDel(value)
	case typeBitmap:
		if ok = c.bmpDel(value); ok && c.bmpShouldConvertToArray() {
			c.bmpToArr()
		}
		return
	case typeRun:
		if ok = c.runDel(value); ok && len(c.run()) > runMinSize {
			c.runToBmp()
		}
		return
	}
	return false
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

// runOptimize converts the container to run format if it would be more efficient
// This method analyzes the current container and converts it to the most efficient representation
func (c *container) runOptimize() {
	switch c.Type {
	case typeArray:
		switch {
		case c.arrTryConvertToRun():
		case c.arrShouldConvertToBitmap():
			c.arrToBmp()
		}

	case typeBitmap:
		switch {
		case c.bmpTryConvertToRun():
		case c.bmpShouldConvertToArray():
			c.bmpToArr()
		}

	case typeRun:
		switch {
		case c.Size <= arrMinSize:
			c.runToArray()
		case len(c.run()) > runMinSize:
			c.runToBmp()
		}
	}
}
