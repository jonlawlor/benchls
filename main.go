// Copyright ©2016 Jonathan J Lawlor. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// benchls computes a least squares fit on groups of parameterized benchmarks.
//
// Usage:
//
//	benchls [options] bench.txt
//
// The input bench.txt file should contain the concatenated output of a number
// of runs of ``go test -bench.'' Benchmarks that match the regexp in the
// ``vars'' flag will be collected into a sample for fitting a least squares
// regression.
//
// Example
//
// Suppose we collect benchmark results from running ``go test -bench=Sort''
// on this package.
//
// The file bench.txt contains:
//
//  PASS
//  BenchmarkSort10-4      	 2000000	       981 ns/op
//  BenchmarkSort100-4     	  200000	      9967 ns/op
//  BenchmarkSort1000-4    	   10000	    180906 ns/op
//  BenchmarkSort10000-4   	    1000	   2269930 ns/op
//  BenchmarkSort100000-4  	      50	  29891719 ns/op
//  BenchmarkSort1000000-4 	       3	 351179975 ns/op
//  BenchmarkSort10000000-4	       1	4274436193 ns/op
//  ok  	github.com/jonlawlor/benchls	149.108s
//
// In these benchmarks, the suffix 10 .. 10000000 indicates how many items are
// sorted in the benchmark.  benchls can estimate the relationship between the
// number of elements to sort and how long it takes to perform the sort.
// Assuming that the amount of time is proportional to n*log(n) and an offset,
// we can run benchls with:
//
//    $ benchls -vars="/?(?P<N>\\d+)-\\d+$" -xtransform="math.Log(N) * N, 1.0" bench.txt
//    group \ Y ~    math.Log(N) * N     1.0                R^2
//    BenchmarkSort  22.034725137066147  291718.2837528091  0.999999774648206
//
// Where the coefficient for math.Log(N) * N = 22.03... and the intercept is 291718.28...
//
// Other options are:
//   -html
//    	print results as an HTML table
//  -response string
//    	benchmark field to use as a response variable {"NsPerOp", "AllocedBytesPerOp", "AllocsPerOp", "MBPerS"} (default "NsPerOp")
//  -vars string
//    	where to find named input variables in the benchmark names (default "/?(?P<N>\\d+)-\\d+$")
//  -xt string
//    	how to construct the explanatory variables from the input variables, separated by commas (shorthand) (default "N, 1.0")
//  -xtransform string
//    	how to construct the explanatory variables from the input variables, separated by commas (default "N, 1.0")
//  -yt string
//    	how to transform the response variable (shorthand) (default "Y")
//  -ytransform string
//    	how to transform the response variable (default "Y")
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"golang.org/x/tools/benchmark/parse"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: benchls [options] bench.txt\n")
	fmt.Fprintf(os.Stderr, "performs a least squares fit on parameterized benchmarks\n")
	fmt.Fprintf(os.Stderr, "example:\n")
	fmt.Fprintf(os.Stderr, "   benchls -vars=\"(?P<M>\\d+)x(?P<N>\\d+)-\\d+$\" -xt=\"math.Log(M), math.Log(N), 1.0\" -yt=\"math.Log(Y)\"\n")
	fmt.Fprintf(os.Stderr, "options:\n")
	flag.PrintDefaults()
	os.Exit(2)
}

var (
	flagInputMatch string
	flagXTransform string
	flagYTransform string
	flagYVar       string
	flagHTML       bool
)

var validYs = []string{"NsPerOp", "AllocedBytesPerOp", "AllocsPerOp", "MBPerS"}

func init() {
	flag.StringVar(&flagInputMatch, "vars", `/?(?P<N>\d+)-\d+$`, "where to find named input variables in the benchmark names")

	const (
		defaultXTransform = "N, 1.0"
		XTransformUsage   = "how to construct the explanatory variables from the input variables, separated by commas"
	)
	flag.StringVar(&flagXTransform, "xtransform", defaultXTransform, XTransformUsage)
	flag.StringVar(&flagXTransform, "xt", defaultXTransform, XTransformUsage+" (shorthand)")

	flag.StringVar(&flagYVar, "response", "NsPerOp", `benchmark field to use as a response variable {"`+strings.Join(validYs, `", "`)+`"}`)

	const (
		defaultYTransform = "Y"
		YTransformUsage   = "how to transform the response variable"
	)
	flag.StringVar(&flagYTransform, "ytransform", defaultYTransform, YTransformUsage)
	flag.StringVar(&flagYTransform, "yt", defaultYTransform, YTransformUsage+" (shorthand)")

	flag.BoolVar(&flagHTML, "html", false, "print results as an HTML table")

}

func main() {
	log.SetPrefix("benchls: ")
	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) > 1 {
		log.Fatal("too many input arguments")
	}

	// find the named variables in the input
	inre := regexp.MustCompile(flagInputMatch)
	varNames := readNames(inre)
	if _, exists := varNames["Y"]; exists {
		log.Fatal("`Y` is reserved and cannot be used as a named expression in vars.")
	}
	// construct the functions for explanatory and response
	xExprs, err := parseX(varNames, flagXTransform)
	if err != nil {
		log.Fatal(err)
	}

	varNames["Y"] = struct{}{}
	yExpr, err := parseY(varNames, flagYTransform)
	if err != nil {
		log.Fatal(err)
	}

	// check that Y is a valid name
	found := false
	for _, y := range validYs {
		if y == flagYVar {
			found = true
			break
		}
	}
	if !found {
		log.Fatal("invalid response: ", flagYVar)
	}
	// read the benchmarks from the file
	f, err := os.Open(args[0])
	if err != nil {
		log.Fatal(err)
	}
	benchSet, err := parse.ParseSet(f)
	if err != nil {
		log.Fatal(err)
	}

	// collect the samples
	samps := sampleGroup(benchSet, inre, xExprs, yExpr, flagYVar)

	// estimate the parameters
	fits := make(map[string]model)
	rsquares := make(map[string]float64)
	for g, samp := range samps {
		fits[g] = estimate(samp)
		if fits[g] == nil {
			continue
		}
		// determine goodness of fit
		rsquares[g] = calcR2(fits[g], samp)
	}

	// generate the report
	writeReport(xExprs, yExpr, fits, rsquares, os.Stdout)
}

func readNames(re *regexp.Regexp) map[string]struct{} {
	varNames := make(map[string]struct{})
	for _, n := range re.SubexpNames() {
		varNames[n] = struct{}{}
	}
	return varNames
}
