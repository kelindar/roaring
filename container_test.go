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
