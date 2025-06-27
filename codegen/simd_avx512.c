// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

#include <stdint.h>

void _find16(uint16_t *input, uint16_t target, int64_t *result, uint64_t size) {
    *result = -1;
    #pragma clang loop vectorize(enable) interleave(enable)
    for (int i = 0; i < (int)size; i++) {
        if (input[i] == target) {
            *result = i;
            return;
        }
    }
}
