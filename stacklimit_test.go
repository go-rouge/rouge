// Copyright (c) the go-rouge/rouge authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rouge

import (
	"runtime"
	"strings"
	"testing"

	onigmo "github.com/go-ruby-regexp/regexp"
)

// TestNegatedSpaceClassHandlesNonLatin1Whitespace pins the engine fix that let
// the Julia lexer go back to the gem's own `[^\S\n]+`.
//
// Onigmo's \s is ASCII-only in Ruby (`"　" =~ /\s/` is nil), so `[^\S\n]` is
// the ASCII whitespace except newline, and U+2003 EM SPACE / U+3000 IDEOGRAPHIC
// SPACE are NOT in it.
//
// ⛔ go-regexp/engine < v0.4.0 got that wrong for code points above U+00FF: a
// negated class holding \S was evaluated byte-wise and swallowed every
// non-ASCII character, so `2 * π` came out as Text " π ", and the lexer spelled
// the set `(?:(?!\n)\s)+` to get around it. The same fix covers `[\D\W\S\H]`,
// which must match those code points.
func TestNegatedSpaceClassHandlesNonLatin1Whitespace(t *testing.T) {
	re, err := onigmo.Compile(`\A[^\S\n]+`)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ in, want string }{
		{" \t　x", " \t"},        // stops at the ideographic space
		{"  ", ""},              // does not start on an em space
		{"　", ""},               //
		{"π ", ""},              // a letter is not whitespace either
		{" \n", " "},            // the newline is what the class excludes
		{"\v\f\r a", "\v\f\r "}, // the rest of ASCII whitespace is in
	} {
		got := ""
		if m := re.Match(c.in); m != nil {
			got = m.Str(0)
		}
		if got != c.want {
			t.Errorf("[^\\S\\n]+ on %q matched %q, want %q", c.in, got, c.want)
		}
	}

	all, err := onigmo.Compile(`\A[\D\W\S\H]`)
	if err != nil {
		t.Fatal(err)
	}
	for _, in := range []string{" ", "　", "π"} {
		if all.Match(in) == nil {
			t.Errorf("[\\D\\W\\S\\H] does not match %q", in)
		}
	}

	// Through the lexer that uses the gem's pattern: no Text run swallows the
	// non-ASCII characters around the whitespace.
	for _, ws := range []string{" ", "　"} {
		src := "π" + ws + "=" + ws + "π"
		var got strings.Builder
		pi := 0
		for _, tv := range FindLexer("julia").Lex(src) {
			got.WriteString(tv.Value)
			if tv.Token == Text && strings.ContainsAny(tv.Value, "π 　") {
				t.Errorf("lexing %q: Text token %q swallowed non-ASCII text", src, tv.Value)
			}
			if tv.Token == NameBuiltin && tv.Value == "π" {
				pi++
			}
		}
		if got.String() != src {
			t.Errorf("lexing %q lost bytes: %q", src, got.String())
		}
		if pi != 1 {
			t.Errorf("lexing %q: %d π builtin names, want 1 (the second is glued to the em space)", src, pi)
		}
	}
}

// bombSource is an R program that ends in an unterminated single-quoted string
// of about size bytes. The rule `'(\\.|.)*?'` keeps 64 bytes of backtrack state
// per input byte (64 MiB for 1 MiB, against the 16 MiB limit) and allocates
// ~510 MB along the way.
func bombSource(size int) string {
	return "x <- 1\n'" + strings.Repeat("a", size-len("x <- 1\n'"))
}

// TestStackLimitBomb is the refusal path: a match that hits the stack limit
// gives up the rest of the input as ONE Text token -- no panic, no unbounded
// memory, and every byte still accounted for.
//
// CI runs this test a second time, alone, in its own process under GOMEMLIMIT
// and `ulimit -v`, so that a regression to unbounded growth is an out-of-memory
// failure rather than a slow pass; see .github/workflows/ci.yml.
func TestStackLimitBomb(t *testing.T) {
	src := bombSource(1 << 20)
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	toks := FindLexer("r").Lex(src)
	runtime.ReadMemStats(&after)
	// Total allocation is exact under load (peak is not). Unbounded, this lex
	// allocates ~510 MB; with the limit it stays within a small multiple of the
	// 16 MiB budget.
	if alloc := after.TotalAlloc - before.TotalAlloc; alloc > 128<<20 {
		t.Errorf("lexing the bomb allocated %d MiB, want under 128", alloc>>20)
	}

	var got strings.Builder
	for _, tv := range toks {
		got.WriteString(tv.Value)
	}
	if got.String() != src {
		t.Fatalf("lexing did not account for every byte: got %d, want %d", got.Len(), len(src))
	}

	last := toks[len(toks)-1]
	if last.Token != Text {
		t.Fatalf("the refused tail is %s, want Text", last.Token.Qualname())
	}
	if want := src[len("x <- 1\n"):]; last.Value != want {
		t.Fatalf("the refused tail is %d bytes, want the whole %d-byte string", len(last.Value), len(want))
	}
	if len(toks) < 2 || toks[0].Token == Text {
		t.Errorf("the input before the bomb was not lexed normally: %v", toks[:1])
	}
}

// TestStackLimitSparesLongTerminatedStrings: the limit must not touch input that
// merely is long. A terminated 64 KiB string needs 4 MiB of the 16 MiB budget.
func TestStackLimitSparesLongTerminatedStrings(t *testing.T) {
	src := "'" + strings.Repeat("a", 64<<10) + "'"
	toks := FindLexer("r").Lex(src)
	if len(toks) != 1 || toks[0].Token != LiteralStringSingle || toks[0].Value != src {
		t.Fatalf("a terminated string was refused: %d tokens", len(toks))
	}
}
