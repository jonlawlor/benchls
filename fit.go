// Copyright ©2016 Jonathan J Lawlor. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/gonum/blas"
	"github.com/gonum/blas/blas64"
	"github.com/gonum/lapack/lapack64"
	"github.com/gonum/matrix/mat64"
	"github.com/jonlawlor/parsefloat"
	"golang.org/x/tools/benchmark/parse"
)

type samp struct {
	x []float64 // explanatory
	y []float64 // response
}

// sampleGroup finds the samples in the benchmark.  The resulting samp x and y will
// not be in a stable order.
func sampleGroup(benchSet parse.Set, inre *regexp.Regexp, xExprs []parsefloat.Expression, yExpr parsefloat.Expression, yVar string) map[string]samp {
	samps := make(map[string]samp)
Bench:
	for name, bs := range benchSet {
		// determine if we can find input variables to construct x and y
		input := inre.FindStringSubmatch(name)
		if input == nil {
			continue
		}
		// create the group name from whatever didn't match
		groupName := strings.TrimRight(name, input[0])

		// convert input string matches into a variable map
		vars := make(map[string]float64)
		for i, varname := range inre.SubexpNames() {
			if i == 0 {
				continue
			}
			val, err := strconv.ParseFloat(input[i], 64)
			if err != nil {
				log.Println("non numeric string in \"" + name + "\": " + input[i] + ", skipping.")
				continue Bench
			}
			vars[varname] = val
		}

		// eval x
		x := make([]float64, len(xExprs))
		for i, xExpr := range xExprs {
			x[i] = xExpr.Eval(vars)
		}

		s := samps[groupName]
		for _, b := range bs {
			// add "Y" to the vars
			switch yVar {
			case "NsPerOp":
				vars["Y"] = b.NsPerOp
			case "AllocedBytesPerOp":
				vars["Y"] = float64(b.AllocedBytesPerOp)
			case "AllocsPerOp":
				vars["Y"] = float64(b.AllocsPerOp)
			case "MBPerS":
				vars["Y"] = b.MBPerS
			default:
				panic("unknown YVar: " + yVar)
			}

			// eval y
			y := yExpr.Eval(vars)
			s.x = append(s.x, x...)
			s.y = append(s.y, y)
		}
		samps[groupName] = s
	}
	return samps
}

// model contains the model parameters
type model []float64

// estimate parameters via least squares.  Returns nil if it could not converge.
func estimate(s samp) model {
	y := blas64.General{
		Rows:   len(s.y),
		Cols:   1,
		Stride: 1,
		Data:   make([]float64, len(s.y)),
	}
	copy(y.Data, s.y)

	x := blas64.General{
		Rows:   len(s.y),
		Cols:   len(s.x) / len(s.y),
		Stride: len(s.x) / len(s.y),
		Data:   make([]float64, len(s.x)),
	}
	copy(x.Data, s.x)

	// find optimal work size
	work := make([]float64, 1)
	lapack64.Gels(blas.NoTrans, x, y, work, -1)

	work = make([]float64, int(work[0]))
	ok := lapack64.Gels(blas.NoTrans, x, y, work, len(work))

	if !ok {
		return nil
	}
	return y.Data[:x.Cols]
}

// calculate R squared
func stats(m model, s samp) (r2 float64, cint []float64) {
	RSS := 0.0
	YSS := 0.0

	// also consumed degrees of freedom
	stride := len(s.x) / len(s.y)
	for i, y := range s.y {
		YSS += y * y
		yHat := 0.0
		for j, x := range s.x[i*stride : (i+1)*stride] {
			yHat += m[j] * x
		}
		RSS += (yHat - y) * (yHat - y)
	}
	r2 = 1.0 - RSS/YSS

	mse := RSS / float64(len(s.y)-stride)
	X := mat64.NewDense(len(s.y), stride, s.x)
	XTX := mat64.NewDense(stride, stride, make([]float64, stride*stride))
	XTX.Mul(X.T(), X)
	XTX.Inverse(XTX)
	cint = make([]float64, stride)
	for i := 0; i < stride; i++ {
		cint[i] = conf95(math.Sqrt(XTX.At(i, i)*mse), len(s.y)-stride)
	}

	return
}
