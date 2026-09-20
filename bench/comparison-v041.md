# v0.4.1 benchmark comparison

Baseline: `main` source. Candidate: `optimize-pass`. Both runs used `github.com/kelindar/bench v0.4.1`, the same harness and fixtures, 100 samples, and 10 ms per sample.

`Faster` is `(main median ns/op / optimize-pass median ns/op - 1) * 100`; positive values mean the candidate is faster. Medians are calculated from each JSON sample set. Allocation values are median allocs/op.

## Summary

- Common cases: **88**
- Median per-case improvement: **12.4%**
- Geometric-mean latency: **80.9% of main** (**19.1% faster**)
- At least 10% faster: **48/88**
- Slower: **26/88**

## Group summary

| Group | Cases | Geometric mean improvement | >=10% faster | Slower |
| --- | ---: | ---: | ---: | ---: |
| set | 16 | -8.3% faster | 2 | 13 |
| has | 8 | 4.2% faster | 2 | 3 |
| del | 16 | 30.6% faster | 10 | 0 |
| and | 8 | 12.5% faster | 4 | 3 |
| or | 8 | 0.9% faster | 5 | 3 |
| xor | 8 | 16.4% faster | 6 | 2 |
| andnot | 8 | 27.8% faster | 5 | 2 |
| range | 8 | 42.4% faster | 8 | 0 |
| write | 4 | 31.2% faster | 2 | 0 |
| read | 4 | 49.1% faster | 4 | 0 |

## Full comparison

| Case | main ns/op | optimize-pass ns/op | Faster | main allocs/op | optimize-pass allocs/op |
| --- | ---: | ---: | ---: | ---: | ---: |
| set 1K (dns) | 20.93 | 20.29 | 3.2% | 0 | 0 |
| set 1K (dns) clr | 8.41 | 10.01 | -16% | 0 | 0 |
| set 1K (rnd) | 31.12 | 33.05 | -5.8% | 0 | 0 |
| set 1K (rnd) clr | 18.79 | 19.3 | -2.7% | 0.01 | 0.01 |
| set 1K (seq) | 19.17 | 24.48 | -21.7% | 0 | 0 |
| set 1K (seq) clr | 5.49 | 7.65 | -28.3% | 0.01 | 0.01 |
| set 1K (sps) | 20.78 | 22.29 | -6.8% | 0 | 0 |
| set 1K (sps) clr | 9.1 | 10.48 | -13.2% | 0.04 | 0.04 |
| set 1M (dns) | 12.91 | 17.36 | -25.6% | 0 | 0 |
| set 1M (dns) clr | 8.21 | 9.3 | -11.7% | 0 | 0 |
| set 1M (rnd) | 20.14 | 23.5 | -14.3% | 0 | 0 |
| set 1M (rnd) clr | 31.11 | 28.2 | 10.3% | 0 | 0 |
| set 1M (seq) | 11.84 | 13.48 | -12.1% | 0 | 0 |
| set 1M (seq) clr | 11.13 | 5.06 | 120% | 0 | 0 |
| set 1M (sps) | 31.29 | 44.12 | -29.1% | 0 | 0 |
| set 1M (sps) clr | 12.06 | 13.21 | -8.7% | 0.03 | 0.03 |
| has 1K (dns) | 19.35 | 15.23 | 27.1% | 0 | 0 |
| has 1K (rnd) | 25.83 | 23.65 | 9.2% | 0 | 0 |
| has 1K (seq) | 17.86 | 21.53 | -17.1% | 0 | 0 |
| has 1K (sps) | 13.27 | 13.52 | -1.9% | 0 | 0 |
| has 1M (dns) | 12.05 | 11.1 | 8.6% | 0 | 0 |
| has 1M (rnd) | 18.93 | 19.08 | -0.8% | 0 | 0 |
| has 1M (seq) | 9.62 | 8.65 | 11.2% | 0 | 0 |
| has 1M (sps) | 29.22 | 27.99 | 4.4% | 0 | 0 |
| del 1K (dns) | 6.94 | 5.35 | 29.8% | 0 | 0 |
| del 1K (dns) clr | 4.91 | 3.84 | 27.7% | 0 | 0 |
| del 1K (rnd) | 6.96 | 5.23 | 32.9% | 0 | 0 |
| del 1K (rnd) clr | 14.11 | 12.49 | 13% | 0 | 0 |
| del 1K (seq) | 6.61 | 5.26 | 25.6% | 0 | 0 |
| del 1K (seq) clr | 13.17 | 12.05 | 9.3% | 0 | 0 |
| del 1K (sps) | 6.96 | 5.32 | 30.8% | 0 | 0 |
| del 1K (sps) clr | 7.75 | 7.26 | 6.8% | 0.02 | 0 |
| del 1M (dns) | 39.85 | 39.32 | 1.3% | 0 | 0 |
| del 1M (dns) clr | 18.53 | 17.98 | 3% | 0 | 0 |
| del 1M (rnd) | 20.43 | 20.29 | 0.7% | 0 | 0 |
| del 1M (rnd) clr | 18.01 | 17.52 | 2.8% | 0 | 0 |
| del 1M (seq) | 6.99 | 5.36 | 30.5% | 0 | 0 |
| del 1M (seq) clr | 10.21 | 4.17 | 144.7% | 0 | 0 |
| del 1M (sps) | 7.07 | 5.28 | 33.9% | 0 | 0 |
| del 1M (sps) clr | 105.71 | 6.82 | 1448.9% | 0.02 | 0 |
| and 1K (dns) | 181.46 | 178.57 | 1.6% | 4 | 4 |
| and 1K (rnd) | 611.04 | 665.86 | -8.2% | 4 | 4 |
| and 1K (seq) | 825.87 | 921.06 | -10.3% | 4 | 4 |
| and 1K (sps) | 1233.97 | 1274.23 | -3.2% | 19 | 4 |
| and 1M (dns) | 3112.86 | 2787.11 | 11.7% | 5 | 4 |
| and 1M (rnd) | 25694.17 | 20581.15 | 24.8% | 19.01 | 4 |
| and 1M (seq) | 25500.07 | 20796.01 | 22.6% | 19.01 | 4 |
| and 1M (sps) | 3870050 | 1835433.33 | 110.9% | 15262 | 4 |
| or 1K (dns) | 356.74 | 441.6 | -19.2% | 12 | 5 |
| or 1K (rnd) | 1613.73 | 1095.8 | 47.3% | 16 | 5 |
| or 1K (seq) | 1826.54 | 1442.99 | 26.6% | 16 | 5 |
| or 1K (sps) | 1953.09 | 1641.44 | 19% | 32 | 5 |
| or 1M (dns) | 3288.36 | 2881.1 | 14.1% | 8 | 4 |
| or 1M (rnd) | 27076.83 | 45705.52 | -40.8% | 27.01 | 4.01 |
| or 1M (seq) | 27281.53 | 50314.1 | -45.8% | 27.01 | 4 |
| or 1M (sps) | 4216750 | 2586605 | 63% | 15306 | 5 |
| xor 1K (dns) | 169.81 | 228.32 | -25.6% | 4 | 5 |
| xor 1K (rnd) | 1169.62 | 1002.02 | 16.7% | 14 | 5 |
| xor 1K (seq) | 1423.15 | 1445.65 | -1.6% | 14 | 5 |
| xor 1K (sps) | 1963.68 | 1597.28 | 22.9% | 32 | 5 |
| xor 1M (dns) | 3168.05 | 2459.94 | 28.8% | 8 | 4 |
| xor 1M (rnd) | 26367.66 | 20701 | 27.4% | 27 | 4 |
| xor 1M (seq) | 26862.35 | 20902.78 | 28.5% | 27.01 | 4 |
| xor 1M (sps) | 4171233.33 | 2211480 | 88.6% | 15306 | 5 |
| andnot 1K (dns) | 179 | 162.23 | 10.3% | 5 | 4 |
| andnot 1K (rnd) | 747.06 | 773.34 | -3.4% | 4 | 4 |
| andnot 1K (seq) | 932.85 | 1001.82 | -6.9% | 4 | 4 |
| andnot 1K (sps) | 1424.77 | 1401.48 | 1.7% | 19 | 4 |
| andnot 1M (dns) | 3073.51 | 2437.95 | 26.1% | 5 | 4 |
| andnot 1M (rnd) | 94444.34 | 20907.72 | 351.7% | 19 | 4 |
| andnot 1M (seq) | 25236.29 | 21412.93 | 17.9% | 19 | 4 |
| andnot 1M (sps) | 4004866.67 | 2002690 | 100% | 15262 | 4 |
| range 1K (dns) | 133.3 | 110.73 | 20.4% | 0 | 0 |
| range 1K (rnd) | 468.33 | 396.29 | 18.2% | 0 | 0 |
| range 1K (seq) | 615.89 | 518.47 | 18.8% | 0 | 0 |
| range 1K (sps) | 679.98 | 570.83 | 19.1% | 0 | 0 |
| range 1M (dns) | 194677.42 | 103128.86 | 88.8% | 0 | 0 |
| range 1M (rnd) | 2101720 | 469832.41 | 347.3% | 0 | 0 |
| range 1M (seq) | 2301840 | 570598.83 | 303.4% | 0 | 0 |
| range 1M (sps) | 667633.33 | 556430.56 | 20% | 0 | 0 |
| write dns | 1099.05 | 1034.62 | 6.2% | 7 | 4 |
| write rnd | 3064.7 | 2829.95 | 8.3% | 11 | 5 |
| write seq | 148.43 | 64.64 | 129.6% | 9 | 3 |
| write sps | 143015.31 | 84559.67 | 69.1% | 4593 | 15.01 |
| read dns | 2862.86 | 1721.83 | 66.3% | 8 | 5 |
| read rnd | 5823.84 | 3359.82 | 73.3% | 13 | 6 |
| read seq | 226.3 | 92.17 | 145.5% | 13 | 6 |
| read sps | 160758.23 | 76553.16 | 110% | 6129.02 | 1530.01 |

Raw result files: [`main-v041.json`](main-v041.json), [`optimize-v041.json`](optimize-v041.json), [`main-v041.txt`](main-v041.txt), [`optimize-v041.txt`](optimize-v041.txt).
