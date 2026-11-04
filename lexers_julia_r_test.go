// Copyright (c) the go-rouge/rouge authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rouge

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestJuliaAndRRegistration pins the names Rouge declares in julia.rb and r.rb
// (v5.1.0): the tag, the aliases and the lookups they enable.
func TestJuliaAndRRegistration(t *testing.T) {
	for _, c := range []struct {
		name, tag, title string
		filenames        []string
	}{
		{"julia", "julia", "Julia", []string{"*.jl"}},
		{"jl", "julia", "Julia", nil},
		{"r", "r", "R", []string{"*.R", "*.r", ".Rhistory", ".Rprofile"}},
		{"R", "r", "R", nil},
		{"s", "r", "R", nil},
		{"S", "r", "R", nil},
	} {
		l := FindLexer(c.name)
		if l == nil {
			t.Errorf("FindLexer(%q) = nil", c.name)
			continue
		}
		if l.Tag() != c.tag || l.Title() != c.title {
			t.Errorf("FindLexer(%q) = %s/%s, want %s/%s", c.name, l.Tag(), l.Title(), c.tag, c.title)
		}
		if c.filenames != nil {
			got := strings.Join(l.(*RegexLexer).filename, " ")
			if got != strings.Join(c.filenames, " ") {
				t.Errorf("%s filenames = %q, want %q", c.tag, got, c.filenames)
			}
		}
	}
}

// TestJuliaAndRDetect is Rouge::TextAnalyzer#shebang? as julia.rb and r.rb use
// it: a `#!` line naming the interpreter as a whole word.
func TestJuliaAndRDetect(t *testing.T) {
	for _, c := range []struct {
		text string
		want string // tag, or "" for neither
	}{
		{"#!/usr/bin/env julia\nprintln(1)", "julia"},
		{"#!/usr/bin/julia -O3\n", "julia"},
		{"#!/usr/bin/env Rscript\ncat(1)", "r"},
		{"\n#!/usr/bin/Rscript --vanilla", "r"},
		{"#!/usr/bin/env julialang\n", ""}, // not a whole word
		{"#!/usr/bin/env python\n", ""},
		{"julia\n", ""}, // no shebang at all
		{"", ""},
	} {
		for _, tag := range []string{"julia", "r"} {
			got := FindLexer(tag).(*RegexLexer).detect(c.text)
			if want := tag == c.want; got != want {
				t.Errorf("%s detect(%q) = %v, want %v", tag, c.text, got, want)
			}
		}
	}
}

// TestJuliaTransposeIsNotAnOperator pins a deviation measured against the gem:
// julia.rb's trailing-transpose rule is written with a lookbehind, which the
// gem's StringScanner can never satisfy, so `A'` is a name and a lone `'`
// (rouge's Error), not an operator. The golden julia2.jl carries the same case;
// this names it.
func TestJuliaTransposeIsNotAnOperator(t *testing.T) {
	for _, tv := range FindLexer("julia").Lex("A'") {
		if tv.Token == Operator {
			t.Fatalf("A' lexed with an Operator token %q: the lookbehind rule fires but the gem's never does", tv.Value)
		}
	}
}

// TestJuliaAndRAccountForEveryByte drives rule arms the goldens leave out:
// unterminated constructs, stray bytes and invalid UTF-8.
func TestJuliaAndRAccountForEveryByte(t *testing.T) {
	for _, src := range []string{
		"#= open", "#= #= open =#", "\"open", "I\"open", "`open", "'open", "'\\",
		"function", "function ", "function 1", "struct", "struct ", "mutable struct 1",
		"@", "0x", "1_", "\xff\xfe", "x'", "'''",
	} {
		reconstruct(t, FindLexer("julia"), src)
	}
	for _, src := range []string{
		"'open", "\"open", "`open", "%open", "#' doc", "0x", "1e", "\xff\xfe", ".", "...", "abs", "abs(",
	} {
		reconstruct(t, FindLexer("r"), src)
	}
}

// TestLongRunOfOneTokenIsLexedInLinearTime guards coalesce, which joins the
// consecutive tokens of one type. Joining by appending to the previous value is
// quadratic in the run, and a rule that matches ONE character at a time makes
// the whole body of a construct one long run: a Julia block comment, string or
// raw string, an R string. 1 MiB of unterminated block comment took about 28
// seconds that way; it takes a fraction of a second now, so the bound below is
// loose enough for a loaded machine and far below the quadratic cost.
//
// ⛔ The verdict is BYTES ALLOCATED, not wall-clock time. This test judged time
// (4x the input must take under 10x the time) and failed on main under qemu
// ppc64le with 28.7x (43 ms -> 1.24 s) on an input that measures a steady
// 130-180 ns/byte on amd64 at every size: an emulator's scheduling and GC
// pauses, not the algorithm. The cost the old coalesce had is a copy of the
// whole run for every token appended, so TotalAlloc grows with the SQUARE of the
// input (16x for 4x the input) while a linear lexer's grows 4x; allocation is
// counted exactly whatever else the machine is doing. The smaller of three
// runs is taken so a stray allocation by another goroutine cannot move it.
func TestLongRunOfOneTokenIsLexedInLinearTime(t *testing.T) {
	for _, c := range []struct{ tag, open, filler string }{
		{"julia", "#=", "aaaa "},   // block comment: [^=#] one character at a time
		{"julia", "\"", "aaaa "},   // string body: ["\\] and $ one at a time
		{"julia", "\"", "\\\\ $ "}, // string made of single-character rules
		{"r", "\"", "aaaa \\\\\n"}, // string over many lines
		{"r", "'", "\\\\ \\' "},    // single-quoted, escapes
	} {
		l := FindLexer(c.tag)
		t0 := adversarial(c.open, c.filler, 64<<10)
		t1 := adversarial(c.open, c.filler, 256<<10)
		b0, b1 := lexBytes(l, t0), lexBytes(l, t1)
		ratio := float64(b1) / float64(b0)
		t.Logf("%s %q: %d -> %d bytes allocated for 64 KiB -> 256 KiB: %.2fx", c.tag, c.open, b0, b1, ratio)
		// Linear is 4x; the old coalesce measured ~13x here. 8 sits between.
		if ratio > 8 {
			t.Errorf("%s %q: 4x the input allocated %.1fx the bytes (%d -> %d); linear is 4x, quadratic 16x",
				c.tag, c.open, ratio, b0, b1)
		}
		reconstruct(t, l, t0)
	}
}

// lexBytes is the smallest number of bytes one Lex of src allocates over three
// runs (runtime.MemStats.TotalAlloc is cumulative, so a collection cannot hide
// an allocation).
func lexBytes(l Lexer, src string) uint64 {
	best := ^uint64(0)
	for i := 0; i < 3; i++ {
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		l.Lex(src)
		runtime.ReadMemStats(&after)
		if d := after.TotalAlloc - before.TotalAlloc; d < best {
			best = d
		}
	}
	return best
}

// benchSource repeats the given committed corpus files until the result is at
// least n bytes, so each lexer is timed on the same amount of input.
func benchSource(tb testing.TB, files []string, n int) string {
	tb.Helper()
	var b strings.Builder
	for b.Len() < n {
		for _, f := range files {
			src, err := os.ReadFile(filepath.Join("testdata", f))
			if err != nil {
				tb.Fatal(err)
			}
			b.Write(src)
		}
	}
	return b.String()
}

// BenchmarkLexJuliaRPython times the three lexers on 64 KiB of their own
// corpus: go test -bench LexJuliaRPython -benchmem
func BenchmarkLexJuliaRPython(b *testing.B) {
	const n = 64 << 10
	for _, c := range []struct {
		tag   string
		files []string
	}{
		{"python", []string{"python.py", "python2.py", "python3.py", "python4.py", "python5.py"}},
		// Rouge's own visual samples only: the edge-case files end in an
		// unterminated construct, which would turn every repetition after the
		// first into the body of one comment or string.
		{"julia", []string{"julia.jl"}},
		{"r", []string{"r.r"}},
	} {
		src := benchSource(b, c.files, n)
		l := FindLexer(c.tag)
		b.Run(c.tag, func(b *testing.B) {
			b.SetBytes(int64(len(src)))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				l.Lex(src)
			}
		})
	}
}
