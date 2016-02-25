// Copyright ©2016 Jonathan J Lawlor. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// report.go was adapted from benchstat, at https://github.com/rsc/benchstat
// Its license follows:

// Copyright (c) 2009 The Go Authors. All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//    * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//    * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//    * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package main

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"math"
	"strconv"
	"unicode/utf8"
)

type row struct {
	cols []string
}

func newRow(cols ...string) *row {
	return &row{cols: cols}
}

func (r *row) add(col string) {
	r.cols = append(r.cols, col)
}

func (r *row) trim() {
	for len(r.cols) > 0 && r.cols[len(r.cols)-1] == "" {
		r.cols = r.cols[:len(r.cols)-1]
	}
}

func writeReport(xExprs []*evaluation, yExpr *evaluation, fits map[string]model, rsquares map[string]float64, cints map[string][]float64, w io.Writer) {
	// writes the model fits and rsquares to the Writer
	var table []*row
	xs := make([]string, len(xExprs))
	for i, xExpr := range xExprs {
		xs[i] = xExpr.String()
	}
	heading := []string{"group \\ " + yExpr.String() + " ~"}
	heading = append(heading, xs...)
	heading = append(heading, "R^2")
	for group, m := range fits {

		if len(table) == 0 {
			table = append(table, newRow(heading...))
		}

		coeffs := make([]string, len(xs)+2)
		coeffs[0] = group
		if m == nil {
			// put a placeholder
			for i := range coeffs {
				if i > 0 {
					coeffs[i] = "~"
				}
			}
		} else {
			for i, b := range m {
				// determine if we should truncate coefficients due to confidence
				cint := cints[group][i]
				bLog := math.Log10(math.Abs(b))
				cintLog := math.Log10(cint)
				format := "%.1e±%.1e" // if b is not significant
				if logDiff := bLog - cintLog + 1; logDiff > 0 {
					format = "%." + strconv.Itoa(int(logDiff)) + "e±%.1e"
				}
				coeffs[i+1] = fmt.Sprintf(format, b, cint)
			}
			coeffs[len(m)+1] = fmt.Sprintf("%g", rsquares[group])
		}

		table = append(table, newRow(coeffs...))
	}
	numColumn := 0
	for _, row := range table {
		if numColumn < len(row.cols) {
			numColumn = len(row.cols)
		}
	}

	max := make([]int, numColumn)
	for _, row := range table {
		for i, s := range row.cols {
			n := utf8.RuneCountInString(s)
			if max[i] < n {
				max[i] = n
			}
		}
	}

	var buf bytes.Buffer
	if flagHTML {
		fmt.Fprintf(&buf, "<style>.benchls tbody td:nth-child(1n+2) { text-align: right; padding: 0em 1em; }</style>\n")
		fmt.Fprintf(&buf, "<table class='benchls'>\n")
		printRow := func(row *row, tag string) {
			fmt.Fprintf(&buf, "<tr>")
			for _, cell := range row.cols {
				fmt.Fprintf(&buf, "<%s>%s</%s>", tag, html.EscapeString(cell), tag)
			}
			fmt.Fprintf(&buf, "\n")
		}
		printRow(table[0], "th")
		for _, row := range table[1:] {
			printRow(row, "td")
		}
		fmt.Fprintf(&buf, "</table>\n")
	} else {

		// headings
		row := table[0]
		for i, s := range row.cols {
			switch i {
			case 0:
				fmt.Fprintf(&buf, "%-*s", max[i], s)
			default:
				fmt.Fprintf(&buf, "  %-*s", max[i], s)
			case len(row.cols) - 1:
				fmt.Fprintf(&buf, "  %s\n", s)
			}
		}

		// data
		for _, row := range table[1:] {
			for i, s := range row.cols {
				switch i {
				case 0:
					fmt.Fprintf(&buf, "%-*s", max[i], s)
				default:
					fmt.Fprintf(&buf, "  %*s", max[i], s)
				}
			}
			fmt.Fprintf(&buf, "\n")
		}
	}

	w.Write(buf.Bytes())

}
