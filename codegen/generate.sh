#!/bin/bash

gocc simd_avx.c    --arch avx2   -O1 --package roaring -o ../ 
# gocc simd_avx512.c --arch avx512 -O3 --package roaring -o ../
# gocc simd_neon.c   --arch neon   -O3 --package roaring -o ../
# gocc simd_apple.c  --arch apple  -O3 --package roaring -o ../