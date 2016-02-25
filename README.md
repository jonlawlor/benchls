# benchls

Golang Benchmark Parameterization via Least Squares

[![Build Status](https://travis-ci.org/jonlawlor/benchls.svg?branch=master)](https://travis-ci.org/jonlawlor/benchls)
[![Coverage Status](https://coveralls.io/repos/github/jonlawlor/benchls/badge.svg?branch=master)](https://coveralls.io/github/jonlawlor/benchls?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/jonlawlor/benchls)](https://goreportcard.com/report/github.com/jonlawlor/benchls)
[![GoDoc](https://godoc.org/github.com/jonlawlor/benchls?status.svg)](https://godoc.org/github.com/jonlawlor/benchls)

`go get -u github.com/jonlawlor/benchls`

With the likely support of [sub-benchmarks](https://github.com/golang/proposal/blob/master/design/12166-subtests.md), I think we're going to see quite a few benchmarks that measure performance over a range of parameters, like:

```
benchFoo/10
benchFoo/100
benchFoo/1000
benchFoo/10000
```
or

```benchBar/10x10
benchFoo/100x10
benchFoo/1000x1000
benchFoo/10000x1
```
... and so on

Where the number(s) at the end indicates a size of input, number of iterations, or what have you.  The motivation is usually to see the performance over a range of inputs.

benchls takes benchmark output in that form, and fits the performance against a function of those parameters.  Here's an example:

```golang
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

// replace with subtests in go 1.7?

func BenchmarkSort10(b *testing.B)       { benchmarkSort(b, 10) }
func BenchmarkSort100(b *testing.B)      { benchmarkSort(b, 100) }
func BenchmarkSort1000(b *testing.B)     { benchmarkSort(b, 1000) }
func BenchmarkSort10000(b *testing.B)    { benchmarkSort(b, 10000) }
func BenchmarkSort100000(b *testing.B)   { benchmarkSort(b, 100000) }
func BenchmarkSort1000000(b *testing.B)  { benchmarkSort(b, 1000000) }
func BenchmarkSort10000000(b *testing.B) { benchmarkSort(b, 10000000) }
```

And then with a call to `go test -bench Sort > bench.txt`:
```
PASS
BenchmarkSort10-4      	 2000000	       981 ns/op
BenchmarkSort100-4     	  200000	      9967 ns/op
BenchmarkSort1000-4    	   10000	    180906 ns/op
BenchmarkSort10000-4   	    1000	   2269930 ns/op
BenchmarkSort100000-4  	      50	  29891719 ns/op
BenchmarkSort1000000-4 	       3	 351179975 ns/op
BenchmarkSort10000000-4	       1	4274436193 ns/op
ok  	github.com/jonlawlor/benchls	149.108s
```

You can use benchls to estimate the relationship between the number of sorted items and the time it took to perform the sort:

```bash
$ benchls -vars="/?(?P<N>\\d+)-\\d+$" -xtransform="math.Log(N) * N, 1.0" bench.txt
group \ Y ~    math.Log(N) * N    1.0             R^2
BenchmarkSort  2.653e+01±1.1e-01  -3e+06±6.6e+06  0.9999895008109141
```

benchls's -xtransform and -ytransform options can construct the explanatory and response variables using addition, subtraction, multiplication, division, literal float64's, any function of float64's in the math package, and any named substring in the -vars flag.  After creating a the model matrix, it uses the LAPACK dgels routine to estimate the model coefficients.  If it can't estimate the coefficients it will produce a "~".  The number to the right of the "±" indicates the 95% confidence interval of the coefficient.

This code is in part derived from and inspired by rsc's [benchstat](https://github.com/rsc/benchstat) library.  It is motivated by the need to characterize benchmarks in [gonum](https://github.com/gonum), particularly the [matrix](https://github.com/gonum/matrix), [blas](https://github.com/gonum/blas), and [lapack](https://github.com/gonum/lapack) libraries.
