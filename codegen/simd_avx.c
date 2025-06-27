// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

#include <stdint.h>
#include <immintrin.h>  // For AVX2 intrinsics

// Find first element >= target in sorted uint16 array using AVX2
void _find16(uint16_t *input, uint16_t target, int64_t *result, uint64_t size) {
    *result = -1;
    
    if (size == 0) return;
    

    // Broadcast target to all 16 lanes
    __m256i target_vec = _mm256_set1_epi16((int16_t)target);
    
    uint64_t i = 0;
    
    // Process 16 uint16_t values at a time
    for (; i + 15 < size; i += 16) {
        // Load 16 uint16_t values
        __m256i data = _mm256_loadu_si256((__m256i*)(input + i));
        
        // Compare: data >= target (implemented as NOT(data < target))
        __m256i cmp_lt = _mm256_cmpgt_epi16(target_vec, data);  // target > data
        __m256i cmp_ge = _mm256_andnot_si256(cmp_lt, _mm256_set1_epi16(-1));  // NOT(target > data)
        
        // Convert to bitmask
        uint32_t mask = _mm256_movemask_epi8(cmp_ge);
        
        if (mask != 0) {
            // Use tzcnt to find the first set bit efficiently
            uint32_t first_bit = _tzcnt_u32(mask);
            
            // Since each uint16_t produces 2 bytes in the mask, divide by 2
            uint32_t element_index = first_bit / 2;
            
            *result = i + element_index;
            return;
        }
    }
    
    // Handle remaining elements
    for (; i < size; i++) {
        if (input[i] >= target) {
            *result = i;
            return;
        }
    }
}
