# benchls

Go Benchmark Performance Curve Fitting via Least Squares

[![Build Status](https://travis-ci.org/jonlawlor/benchls.svg?branch=master)](https://travis-ci.org/jonlawlor/benchls)
[![Coverage Status](https://coveralls.io/repos/github/jonlawlor/benchls/badge.svg?branch=master)](https://coveralls.io/github/jonlawlor/benchls?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/jonlawlor/benchls)](https://goreportcard.com/report/github.com/jonlawlor/benchls)
[![GoDoc](https://godoc.org/github.com/jonlawlor/benchls?status.svg)](https://godoc.org/github.com/jonlawlor/benchls)

`go get [-u] github.com/jonlawlor/benchls`

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
// This is just for illustration...
type Ints []int

func (a Ints) Len() int           { return len(a) }
func (a Ints) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Ints) Less(i, j int) bool { return a[i] < a[j] }

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
		sort.Sort(Ints(data))
	}
}

func benchmarkStableSort(b *testing.B, n int) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		rand.Seed(1)
		data := make([]int, n)
		for i := range data {
			data[i] = rand.Int()
		}
		b.StartTimer()
		sort.Stable(Ints(data))
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

func BenchmarkStableSort10(b *testing.B)       { benchmarkStableSort(b, 10) }
func BenchmarkStableSort100(b *testing.B)      { benchmarkStableSort(b, 100) }
func BenchmarkStableSort1000(b *testing.B)     { benchmarkStableSort(b, 1000) }
func BenchmarkStableSort10000(b *testing.B)    { benchmarkStableSort(b, 10000) }
func BenchmarkStableSort100000(b *testing.B)   { benchmarkStableSort(b, 100000) }
func BenchmarkStableSort1000000(b *testing.B)  { benchmarkStableSort(b, 1000000) }
func BenchmarkStableSort10000000(b *testing.B) { benchmarkStableSort(b, 10000000) }
```

And then with a call to `go test -bench Sort > bench.txt`:
```
PASS
BenchmarkSort10-4            	 1000000	      1008 ns/op
BenchmarkSort100-4           	  200000	      8224 ns/op
BenchmarkSort1000-4          	   10000	    152945 ns/op
BenchmarkSort10000-4         	    1000	   1950999 ns/op
BenchmarkSort100000-4        	      50	  25081946 ns/op
BenchmarkSort1000000-4       	       5	 302228845 ns/op
BenchmarkSort10000000-4      	       1	3631295293 ns/op
BenchmarkStableSort10-4      	 1000000	      1260 ns/op
BenchmarkStableSort100-4     	  100000	     16730 ns/op
BenchmarkStableSort1000-4    	    5000	    362024 ns/op
BenchmarkStableSort10000-4   	     300	   5731738 ns/op
BenchmarkStableSort100000-4  	      20	  88171712 ns/op
BenchmarkStableSort1000000-4 	       1	1205361782 ns/op
BenchmarkStableSort10000000-4	       1	14349613704 ns/op
ok  	github.com/jonlawlor/benchls	138.860s
```

You can use benchls to estimate the relationship between the number of sorted items and the time it took to perform the particular sort:

```bash
$ benchls -vars="/?(?P<N>\\d+)-\\d+$" -xtransform="math.Log(N) * N, 1.0" bench.txt
group \ Y ~          math.Log(N) * N    1.0             R^2
BenchmarkSort        2.254e+01±6.4e-02  -2e+06±3.9e+06  0.9999949426719544
BenchmarkStableSort  8.906e+01±1.8e-01  -7e+06±1.1e+07  0.9999973642760738
```

benchls's -xtransform and -ytransform options can construct the explanatory and response variables using addition, subtraction, multiplication, division, literal float64's, any function of float64's in the math package, and any named substring in the -vars flag.  After creating a the model matrix, it uses the LAPACK dgels routine to estimate the model coefficients.  If it can't estimate the coefficients it will produce a "~".  The number to the right of the "±" indicates the 95% confidence interval of the coefficient.

This code is in part derived from and inspired by rsc's [benchstat](https://github.com/rsc/benchstat) library.  It is motivated by the need to characterize benchmarks in [gonum](https://github.com/gonum), particularly the [matrix](https://github.com/gonum/matrix), [blas](https://github.com/gonum/blas), and [lapack](https://github.com/gonum/lapack) libraries.
