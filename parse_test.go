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
		xrpn           [][]string
		xstrings       []string
		vars           map[string]float64
		wantX          []float64
		wantY          float64
	}{
		{
			inre:     `(?P<N>\d+)-\d+$`,
			xtrans:   "math.Log(N), 1.0",
			xrpn:     [][]string{{"N", "math.Log"}, {"1.0"}},
			xstrings: []string{"math.Log(N)", "1.0"},
			ytrans:   "Y",
			vars:     map[string]float64{"N": 10.0, "Y": 2.0},
			wantX:    []float64{math.Log(10.0), 1.0},
			wantY:    2.0,
		}, {
			inre:     `(?P<N>\d+)-\d+$`,
			xtrans:   "1.0 / N",
			xrpn:     [][]string{{"1.0", "N", "/"}},
			xstrings: []string{"1.0 / N"},
			ytrans:   "Y / N",
			vars:     map[string]float64{"N": 0.0, "Y": 1.0},
			wantX:    []float64{math.Inf(1)},
			wantY:    math.Inf(1),
		}, {
			inre:     `(?P<N>\d+)-\d+$`,
			xtrans:   "N*N, N, 1.0",
			xrpn:     [][]string{{"N", "N", "*"}, {"N"}, {"1.0"}},
			xstrings: []string{"N*N", "N", "1.0"},
			ytrans:   "Y+1",
			vars:     map[string]float64{"N": 10.0, "Y": 2.0},
			wantX:    []float64{100.0, 10.0, 1.0},
			wantY:    3.0,
		}, {
			inre:     `(?P<M>\d+)(?P<N>\d+)-\d+$`,
			xtrans:   "-math.Hypot(M+N, M-N), +M/N",
			xrpn:     [][]string{{"M", "N", "+", "M", "N", "-", "math.Hypot", "u-"}, {"M", "u+", "N", "/"}},
			xstrings: []string{"-math.Hypot(M+N, M-N)", "+M/N"},
			ytrans:   "Y+1",
			vars:     map[string]float64{"M": 3.5, "N": 0.5, "Y": -1.0},
			wantX:    []float64{-5.0, 7.0},
			wantY:    0.0,
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
			if x := expr.String(); x != tt.xstrings[xi] {
				t.Errorf("%d: expected x[%d].String() = %s, got %s", i, xi, tt.xstrings[xi], x)
			}
			for xouti, xout := range expr.output {
				if xout.String() != tt.xrpn[xi][xouti] {
					t.Errorf("%d: expected x[%d].output[%d].String() = %s, got %s", i, xi, xouti, tt.xrpn[xi][xouti], xout.String())
				}
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
