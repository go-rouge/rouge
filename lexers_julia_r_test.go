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

// TestCoalesceIsLinearInTheRun guards coalesce, which joins the consecutive
// tokens of one type. Joining by appending to the previous value copies the whole
// run for every token, so it is quadratic in the run -- and a rule that matches
// ONE character at a time makes the whole body of a construct one long run: a
// Julia block comment or string, an R string. 1 MiB of unterminated Julia block
// comment took about 28 seconds that way.
//
// ⛔ It judges BYTES ALLOCATED by coalesce itself, on a synthetic run, not the
// time of a whole Lex. Two earlier versions failed on CI for reasons that were
// not this defect: one judged wall-clock time and failed under qemu ppc64le
// (28.7x for 4x the input, on an input that measures a steady 130-180 ns/byte on
// amd64); the next judged TotalAlloc of a whole Lex and failed on the linux
// runners because the ENGINE allocates 10-250 MB at 256 KiB there and ~nothing
// at 64 KiB (and nothing at either size locally). A run of N one-byte tokens
// allocates ~N bytes when joined once and ~N^2/2 when joined by +=, so 4x the
// run is 4x vs 16x the bytes, counted exactly whatever the machine is doing.
func TestCoalesceIsLinearInTheRun(t *testing.T) {
	run := func(n int) []TokenValue {
		in := make([]TokenValue, n)
		for i := range in {
			in[i] = TokenValue{Token: Comment, Value: "a"}
		}
		return in
	}
	b0 := allocBytes(func() { coalesce(run(64 << 10)) }, func() { run(64 << 10) })
	b1 := allocBytes(func() { coalesce(run(256 << 10)) }, func() { run(256 << 10) })
	ratio := float64(b1) / float64(b0)
	t.Logf("coalesce of a 64K-token run -> 256K-token run: %d -> %d bytes, %.2fx", b0, b1, ratio)
	// Linear is 4x; the += version measures 16x. 8 sits between.
	if ratio > 8 {
		t.Errorf("4x the run allocated %.1fx the bytes (%d -> %d); linear is 4x, quadratic 16x", ratio, b0, b1)
	}
	out := coalesce(run(1000))
	if len(out) != 1 || len(out[0].Value) != 1000 {
		t.Errorf("coalesce of 1000 equal tokens = %d token(s), want 1 of length 1000", len(out))
	}
}

// TestLongRunsAreAccountedFor lexes 64 KiB of unterminated constructs, each a
// long run of one-character tokens, and checks every byte comes back.
func TestLongRunsAreAccountedFor(t *testing.T) {
	for _, c := range []struct{ tag, open, filler string }{
		{"julia", "#=", "aaaa "},   // block comment: [^=#] one character at a time
		{"julia", "\"", "aaaa "},   // string body: ["\\] and $ one at a time
		{"julia", "\"", "\\\\ $ "}, // string made of single-character rules
		{"r", "\"", "aaaa \\\\\n"}, // string over many lines
		{"r", "'", "\\\\ \\' "},    // single quote with escapes
	} {
		reconstruct(t, FindLexer(c.tag), adversarial(c.open, c.filler, 64<<10))
	}
}

// allocBytes is the smallest number of bytes f allocates over three runs, less
// what the setup alone allocates (runtime.MemStats.TotalAlloc is cumulative, so
// a collection cannot hide an allocation).
func allocBytes(f, setup func()) uint64 {
	measure := func(g func()) uint64 {
		best := ^uint64(0)
		for i := 0; i < 3; i++ {
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			g()
			runtime.ReadMemStats(&after)
			if d := after.TotalAlloc - before.TotalAlloc; d < best {
				best = d
			}
		}
		return best
	}
	return measure(f) - measure(setup)
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
