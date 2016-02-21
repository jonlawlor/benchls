// Copyright ©2016 Jonathan J Lawlor. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"math"
	"regexp"
	"testing"
)

func TestParse(t *testing.T) {
	for i, tt := range []struct {
		inre           string
		xtrans, ytrans string
		vars           map[string]float64
		wantX          []float64
		wantY          float64
	}{
		{
			inre:   `(?P<N>\d+)-\d+$`,
			xtrans: "math.Log(N), 1.0",
			ytrans: "Y",
			vars:   map[string]float64{"N": 10.0, "Y": 2.0},
			wantX:  []float64{math.Log(10.0), 1.0},
			wantY:  2.0,
		},
	} {
		inre := regexp.MustCompile(tt.inre)
		names := readNames(inre)
		xExpr, err := parseX(names, tt.xtrans)
		if err != nil {
			panic(err)
		}
		for xi, expr := range xExpr {
			if x := expr.value(tt.vars); x != tt.wantX[xi] {
				t.Errorf("%d: expected x[%d] = %f, got %f", i, xi, tt.wantX[xi], x)
			}
		}
		names["Y"] = struct{}{}
		yExpr, err := parseY(names, tt.ytrans)
		if err != nil {
			panic(err)
		}
		if y := yExpr.value(tt.vars); y != tt.wantY {
			t.Errorf("%d: expected y = %f, got %f", i, tt.wantY, y)
		}
	}
}
