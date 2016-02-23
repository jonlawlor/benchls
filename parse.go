// Copyright ©2016 Jonathan J Lawlor. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"math"
	"strconv"
)

// Converts a string in go expression format into a reverse polish notation
// representation of its mathematical transformation, as long as it only has
// float64's, functions of float64 from the math package, and predefined
// identifiers.
//
// Doing this in RPN might be kind of dumb.  There may be a better way of
// evaluating go expressions.
//
// Note that only functions of float64s and float64 literals are allowed, with
// the expection of << and >>, because while those functions can be applied
// to float64's, it will panic if the amount to shift is not an integer.

// unaryFuncs is the set of functions that will be recognized as math.(func) unary functions.
var unaryFuncs = map[string]func(float64) float64{
	"Abs":   math.Abs,
	"Acos":  math.Acos,
	"Acosh": math.Acosh,
	"Asin":  math.Asin,
	"Asinh": math.Asinh,
	"Atan":  math.Atan,
	"Atanh": math.Atanh,
	"Cbrt":  math.Cbrt,
	"Ceil":  math.Ceil,
	"Cos":   math.Cos,
	"Cosh":  math.Cosh,
	"Erf":   math.Erf,
	"Erfc":  math.Erfc,
	"Exp":   math.Exp,
	"Exp2":  math.Exp2,
	"Expm1": math.Expm1,
	"Floor": math.Floor,
	"Gamma": math.Gamma,
	"J0":    math.J0,
	"J1":    math.J1,
	"Log":   math.Log,
	"Log10": math.Log10,
	"Log1p": math.Log1p,
	"Log2":  math.Log2,
	"Logb":  math.Logb,
	"Sin":   math.Sin,
	"Sinh":  math.Sinh,
	"Sqrt":  math.Sqrt,
	"Tan":   math.Tan,
	"Tanh":  math.Tanh,
	"Trunc": math.Trunc,
	"Y0":    math.Y0,
	"Y1":    math.Y1,
}

// binaryFuncs is the set of functions that will be recognized as math.(func) binary functions.
var binaryFuncs = map[string]func(float64, float64) float64{
	"Atan2":     math.Atan2,
	"Copysign":  math.Copysign,
	"Dim":       math.Dim,
	"Hypot":     math.Hypot,
	"Max":       math.Max,
	"Min":       math.Min,
	"Mod":       math.Mod,
	"Nextafter": math.Nextafter,
	"Pow":       math.Pow,
	"Remainder": math.Remainder,
}

func parseX(varNames map[string]struct{}, expr string) ([]*evaluation, error) {
	// find the comma delimited explantory transformation
	fset := token.NewFileSet()
	expr = "float64{" + expr + "}"
	fexpr, err := parser.ParseExprFrom(fset, "", expr, 0)
	if err != nil {
		return nil, err
	}
	slexpr := fexpr.(*ast.CompositeLit)
	var vs []*evaluation
	for _, exp := range slexpr.Elts {
		v, err := newEvaluation(expr[exp.Pos()-1:exp.End()-1], varNames)
		if err != nil {
			return nil, err
		}
		vs = append(vs, v)
	}

	// populate the output
	return vs, nil
}

func parseY(varNames map[string]struct{}, expr string) (*evaluation, error) {
	return newEvaluation(expr, varNames)
}

// for representing the transformation functions in RPN
type operand interface {
	String() string
	value(map[string]float64) float64
}

type float64Literal struct {
	s string
	v float64
}

func (e float64Literal) String() string {
	return e.s
}
func (e float64Literal) value(vars map[string]float64) float64 {
	return e.v
}

type ident string

func (e ident) String() string {
	return string(e)
}
func (e ident) value(vars map[string]float64) float64 {
	return vars[string(e)]
}

type operator interface {
	String() string
	eval(stack []float64) int
}
type uplus struct{}

func (op uplus) String() string {
	return "u+"
}
func (op uplus) eval(stack []float64) int {
	return 0 // do nothing
}

type uminus struct{}

func (op uminus) String() string {
	return "u-"
}
func (op uminus) eval(stack []float64) int {
	l := len(stack)
	stack[l-1] = -stack[l-1]
	return 0 // do nothing
}

type add struct{}

func (op add) String() string {
	return "+"
}
func (op add) eval(stack []float64) int {
	l := len(stack)
	stack[l-2] += stack[l-1]
	return 1
}

type sub struct{}

func (op sub) String() string {
	return "-"
}
func (op sub) eval(stack []float64) int {
	l := len(stack)
	stack[l-2] -= stack[l-1]
	return 1
}

type mul struct{}

func (op mul) String() string {
	return "*"
}
func (op mul) eval(stack []float64) int {
	l := len(stack)
	stack[l-2] *= stack[l-1]
	return 1
}

type quo struct{}

func (op quo) String() string {
	return "/"
}
func (op quo) eval(stack []float64) int {
	l := len(stack)
	stack[l-2] /= stack[l-1]
	return 1
}

type unaryFunc struct {
	s string
	f func(float64) float64
}

func (e unaryFunc) String() string {
	return e.s
}
func (e unaryFunc) eval(stack []float64) int {
	l := len(stack)
	stack[l-1] = e.f(stack[l-1])
	return 0 // number of items to remove from the end
}

type binaryFunc struct {
	s string
	f func(float64, float64) float64
}

func (e binaryFunc) String() string {
	return e.s
}
func (e binaryFunc) eval(stack []float64) int {
	l := len(stack)
	stack[l-2] = e.f(stack[l-2], stack[l-1])
	return 1 // number of items to remove from the end
}

// evaluation takes a limited set of go expressions and turns them into
// operands and operators in RPN for later evaluation.  parseError contains
// errors that can occur during string parsing.
type evaluation struct {
	s          string
	output     []fmt.Stringer // RPN
	knownVars  map[string]struct{}
	parseError error
}

func (e *evaluation) String() string {
	return e.s
}

func (e *evaluation) value(vars map[string]float64) float64 {
	var stack []float64
	if len(e.output) == 0 {
		log.Fatal("no expressions in " + e.String())
	}
	for _, o := range e.output {
		if val, ok := o.(operand); ok {
			stack = append(stack, val.value(vars))
			continue
		}
		if op, ok := o.(operator); ok {
			advance := op.eval(stack)
			if advance > 0 {
				l := len(stack)
				stack = stack[:l-advance]
			}
			continue
		}
	}
	// check for validity
	if len(stack) != 1 {
		log.Fatal("invalid expression: " + e.String())
	}
	return stack[0]
}
func newEvaluation(expr string, varNames map[string]struct{}) (*evaluation, error) {
	fset := token.NewFileSet()
	fexpr, err := parser.ParseExprFrom(fset, "", expr, 0)
	if err != nil {
		return nil, err
	}
	v := &evaluation{s: expr, knownVars: varNames}

	// populate the output
	ast.Walk(v, fexpr)
	return v, v.parseError
}

// Visit implements the ast.Visitor interface.  It populates the
// evaluation output field with identifiers, literals, and functions.
func (e *evaluation) Visit(node ast.Node) (w ast.Visitor) {
	if node == nil || e.parseError != nil {
		// all done
		return e
	}
	switch t := node.(type) {
	case *ast.BasicLit:
		val, err := strconv.ParseFloat(t.Value, 64)
		if err != nil {
			e.parseError = err
			return nil
		}
		e.output = append(e.output, float64Literal{s: t.Value, v: val})
	case *ast.Ident:
		// check to see if we know this literal
		if _, ok := e.knownVars[t.Name]; !ok {
			e.parseError = errors.New("unknown variable: " + t.Name)
			return nil
		}
		e.output = append(e.output, ident(t.Name))
	case *ast.CallExpr:
		// look up the calling function
		fun, ok := t.Fun.(*ast.SelectorExpr)
		if !ok {
			e.parseError = errors.New("unknown function call")
			return nil
		}
		pkg, ok := fun.X.(*ast.Ident)
		if !ok {
			e.parseError = errors.New("unknown function call")
			return nil
		}
		if pkg.Name != "math" {
			e.parseError = errors.New("only math package functions allowed, found package " + pkg.Name)
			return nil
		}

		// walk the args
		for _, a := range t.Args {
			ast.Walk(e, a)
		}

		// add the op to the output
		if op, ok := unaryFuncs[fun.Sel.Name]; ok {
			e.output = append(e.output, unaryFunc{s: "math." + fun.Sel.Name, f: op})
			return nil
		}

		if op, ok := binaryFuncs[fun.Sel.Name]; ok {
			e.output = append(e.output, binaryFunc{s: "math." + fun.Sel.Name, f: op})
			return nil
		}

		e.parseError = errors.New("unknown math function math." + fun.Sel.Name)
		return nil
	case *ast.UnaryExpr:
		ast.Walk(e, t.X)
		switch t.Op {
		case token.ADD:
			e.output = append(e.output, uplus{})
		case token.SUB:
			e.output = append(e.output, uminus{})
		default:
			e.parseError = errors.New("unrecognized unary expression: " + t.Op.String())
		}

		return nil
	case *ast.BinaryExpr:
		ast.Walk(e, t.X)
		ast.Walk(e, t.Y)
		switch t.Op {
		case token.ADD:
			e.output = append(e.output, add{})
		case token.SUB:
			e.output = append(e.output, sub{})
		case token.MUL:
			e.output = append(e.output, mul{})
		case token.QUO:
			e.output = append(e.output, quo{})
		default:
			e.parseError = errors.New("unrecognized binary expression: " + t.Op.String())
		}
		return nil
	}
	return e
}
