// Copyright ©2016 Jonathan J Lawlor. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"math"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/tools/benchmark/parse"
)

func TestFit(t *testing.T) {
	s := `
PASS
BenchmarkSort10-4      	 2000000	       981 ns/op
BenchmarkSort100-4     	  200000	      9967 ns/op
BenchmarkSort1000-4    	   10000	    180906 ns/op
BenchmarkSort10000-4   	    1000	   2269930 ns/op
BenchmarkSort100000-4  	      50	  29891719 ns/op
BenchmarkSort1000000-4 	       3	 351179975 ns/op
BenchmarkSort10000000-4	       1	4274436193 ns/op
ok  	github.com/jonlawlor/benchlm	149.108s
`
	yVar := "NsPerOp"
	r := strings.NewReader(s)
	benchSet, err := parse.ParseSet(r)
	if err != nil {
		panic(err)
	}
	inre := regexp.MustCompile(`(?P<N>\d+)-\d+$`)
	names := readNames(inre)

	// Sort isn't O(n) obviously, but this was easy to verify.
	xtrans := "N, 1.0"
	ytrans := "Y"
	wantFit := []float64{428.2534163147418, -1.4343020792698523e+07}

	xExprs, err := parseX(names, xtrans)
	if err != nil {
		panic(err)
	}
	names["Y"] = struct{}{}
	yExpr, err := parseY(names, ytrans)
	if err != nil {
		panic(err)
	}

	samps := sampleGroup(benchSet, inre, xExprs, yExpr, yVar)
	fit := estimate(samps["BenchmarkSort"])
	for i, f := range fit {
		if math.Abs(wantFit[i]-f) > 1e-6 {
			t.Errorf("expected fit[%d] = %f, got %f", i, wantFit[i], f)
		}
	}
	if r2, _ := stats(fit, samps["BenchmarkSort"]); r2 < .999 || r2 > 1.0 {
		t.Errorf("expected r2 approximately %f, got %f", .999, r2)
	}
}
