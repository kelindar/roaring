// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

#include <stdint.h>
#include <immintrin.h>  // For AVX2 intrinsics

// Find first element >= target in sorted uint16 array using AVX2
void _find16(uint16_t *input, uint16_t target, int64_t *result, uint64_t size) {
    *result = -1;
    if (size == 0) return;

    __m256i vkey = _mm256_set1_epi16((int16_t)target);
    uint64_t i = 0;

    for (; i + 15 < size; i += 16) {
        __m256i v      = _mm256_loadu_si256((const __m256i *)(input + i));

        /* mask bits set where input[j] ≥ target
           movemask produces one bit per byte, so two bits per uint16_t           */
        uint32_t ge    = ~_mm256_movemask_epi8(_mm256_cmpgt_epi16(vkey, v));

        if (ge) {
            /* tz = index of first set byte-bit, convert to word index with >>1 */
            uint32_t j = _tzcnt_u32(ge) >> 1;
            *result = (int64_t)(i + j);
            return;
        }

        _mm_prefetch((const char *)(input + i + 32), _MM_HINT_T0);
    }

    /* scalar tail, ≤15 elements */
    for (; i < size; ++i){
        if (input[i] >= target) {
             *result = (int64_t)i; return;
        }
    }
}
