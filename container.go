// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package roaring

const (
	arrMinSize    = 2048
	runMinSize    = 128
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
	Type   ctype  // Type of the container
	Shared bool   // COW: true if data is shared between containers
	Call   uint16 // Call count
	Size   uint32 // Cardinality
	Data   []uint16
}

// fork ensures the container owns its data before modification
func (c *container) fork() {
	if c.Shared {
		clone := make([]uint16, len(c.Data), cap(c.Data))
		copy(clone, c.Data)
		c.Data = clone
		c.Shared = false
	}
}

// set sets a value in the container and returns true if the value was added (didn't exist before)
func (c *container) set(value uint16) (ok bool) {
	c.fork()
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
	c.fork()
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
	c.fork()
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

// min returns the smallest value in the container
func (c *container) min() (uint16, bool) {
	if c.Size == 0 {
		return 0, false
	}

	switch c.Type {
	case typeArray:
		return c.arrMin()
	case typeBitmap:
		return c.bmpMin()
	case typeRun:
		return c.runMin()
	}
	return 0, false
}

// max returns the largest value in the container
func (c *container) max() (uint16, bool) {
	if c.Size == 0 {
		return 0, false
	}

	switch c.Type {
	case typeArray:
		return c.arrMax()
	case typeBitmap:
		return c.bmpMax()
	case typeRun:
		return c.runMax()
	}
	return 0, false
}

// minZero returns the smallest unset value in the container (0-65535 range)
func (c *container) minZero() (uint16, bool) {
	if c.Size == 65536 {
		return 0, false // Container is full, no zero bits
	}

	switch c.Type {
	case typeArray:
		return c.arrMinZero()
	case typeBitmap:
		return c.bmpMinZero()
	case typeRun:
		return c.runMinZero()
	}
	return 0, false
}

// maxZero returns the largest unset value in the container (0-65535 range)
func (c *container) maxZero() (uint16, bool) {
	if c.Size == 65536 {
		return 0, false // Container is full, no zero bits
	}

	switch c.Type {
	case typeArray:
		return c.arrMaxZero()
	case typeBitmap:
		return c.bmpMaxZero()
	case typeRun:
		return c.runMaxZero()
	}
	return 0, false
}
