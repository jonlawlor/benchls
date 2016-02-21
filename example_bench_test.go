// Copyright ©2016 Jonathan J Lawlor. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"math/rand"
	"sort"
	"testing"
)

func benchmarkSort(b *testing.B, n int) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		rand.Seed(1)
		data := make([]int, n)
		for i := range data {
			data[i] = rand.Int()
		}
		b.StartTimer()
		sort.Ints(data)
	}
}

// replace with subtests

func BenchmarkSort10(b *testing.B)       { benchmarkSort(b, 10) }
func BenchmarkSort100(b *testing.B)      { benchmarkSort(b, 100) }
func BenchmarkSort1000(b *testing.B)     { benchmarkSort(b, 1000) }
func BenchmarkSort10000(b *testing.B)    { benchmarkSort(b, 10000) }
func BenchmarkSort100000(b *testing.B)   { benchmarkSort(b, 100000) }
func BenchmarkSort1000000(b *testing.B)  { benchmarkSort(b, 1000000) }
func BenchmarkSort10000000(b *testing.B) { benchmarkSort(b, 10000000) }
