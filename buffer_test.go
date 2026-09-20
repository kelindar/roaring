// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package roaring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuffer(t *testing.T) {
	t.Run("empty conversions", func(t *testing.T) {
		assert.Nil(t, asBitmap(nil))
		assert.Nil(t, asUint16s(nil))
	})

	t.Run("round trip", func(t *testing.T) {
		data := []uint16{1, 2, 3, 4}
		assert.Equal(t, data, asUint16s(asBitmap(data)))
	})

	t.Run("borrowed memory is clear", func(t *testing.T) {
		data := borrowBitmap()
		defer release(asUint16s(data))

		assert.Len(t, data, bitmapSize/4)
		for _, word := range data {
			assert.Zero(t, word)
		}
	})
}
