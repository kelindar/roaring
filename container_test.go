// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package roaring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainer(t *testing.T) {
	t.Run("fork copies shared data", func(t *testing.T) {
		original := []uint16{1, 2}
		c := container{Type: typeArray, Data: original, Shared: true}

		c.fork()
		c.Data[0] = 9

		assert.False(t, c.Shared)
		assert.Equal(t, []uint16{1, 2}, original)
		assert.Equal(t, []uint16{9, 2}, c.Data)
	})

	t.Run("set reports changes", func(t *testing.T) {
		c := container{Type: typeArray}

		assert.True(t, c.set(7))
		assert.False(t, c.set(7))
		assert.Equal(t, uint32(1), c.Size)
	})
}

func TestContainerBitmap(t *testing.T) {
	t.Run("empty extrema", func(t *testing.T) {
		c := container{Type: typeBitmap, Data: make([]uint16, bitmapSize)}

		_, minOK := c.min()
		_, maxOK := c.max()
		zero, zeroOK := c.minZero()

		assert.False(t, minOK)
		assert.False(t, maxOK)
		assert.True(t, zeroOK)
		assert.Zero(t, zero)
	})

	t.Run("populated extrema", func(t *testing.T) {
		c := container{Type: typeBitmap, Data: make([]uint16, bitmapSize)}
		assert.True(t, c.bmpSet(3))
		assert.True(t, c.bmpSet(65535))
		assert.False(t, c.bmpSet(3))

		min, minOK := c.min()
		max, maxOK := c.max()
		zero, zeroOK := c.minZero()

		assert.True(t, minOK)
		assert.Equal(t, uint16(3), min)
		assert.True(t, maxOK)
		assert.Equal(t, uint16(65535), max)
		assert.True(t, zeroOK)
		assert.Zero(t, zero)
	})

	t.Run("full has no zero", func(t *testing.T) {
		c := container{Type: typeBitmap, Data: make([]uint16, bitmapSize), Size: 1 << 16}
		for i := range c.Data {
			c.Data[i] = ^uint16(0)
		}

		_, ok := c.minZero()
		assert.False(t, ok)
	})
}
