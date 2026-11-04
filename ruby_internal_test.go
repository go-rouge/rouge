// Copyright (c) the go-rouge/rouge authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rouge

import "testing"

// TestMatchBalancedBrace covers both the balanced and the unbalanced (runs to
// end of input) arms of matchBalancedBrace.
func TestMatchBalancedBrace(t *testing.T) {
	// Balanced, with nesting: the '{' at index 0 closes at the matching '}'.
	if got, closed := matchBalancedBrace("{a{b}c}d", 0); got != 7 || !closed {
		t.Errorf("balanced = %d, %v, want 7, true", got, closed)
	}
	// Unbalanced: no closing brace -> len(s), and NOT closed (the caller must not
	// emit a `}` that is not in the input).
	s := "{a{b}"
	if got, closed := matchBalancedBrace(s, 0); got != len(s) || closed {
		t.Errorf("unbalanced = %d, %v, want %d, false", got, closed, len(s))
	}
}

// TestSigilStringEdges drives the rubyLexer through %-sigil forms that exercise
// sigilString's branches: a non-interpolating %w word list (the # default arm),
// an interpolating %() with a nested-brace #{...}, an escape inside the body, an
// asymmetric nested delimiter, and an unterminated body that runs to EOF. These
// assert the lexer produces a token stream (no panic / full consumption) rather
// than gem parity, since the unterminated case is malformed Ruby.
func TestSigilStringEdges(t *testing.T) {
	for _, src := range []string{
		`%w(a b c)`,             // non-interp: '#'/default arm, symmetric ()
		`%(x #{y} z)`,           // interp with #{...}
		`%(esc \) done)`,        // escape inside body
		`%[outer [inner] tail]`, // asymmetric nested delimiter
		`%(unterminated body`,   // EOF before close
		`%r{a#{b}c}i`,           // regex sigil with interp + flags
		`%!#{`,                  // fuzz finding: an unclosed #{ invented a `}`
		`%(#{a {b}`,             // ... and an unclosed one holding a balanced pair
	} {
		out := rubyLexer.Lex(src)
		if len(out) == 0 {
			t.Errorf("%q produced no tokens", src)
		}
		// The whole input must be consumed (values concatenate back to src).
		var b string
		for _, tv := range out {
			b += tv.Value
		}
		if b != src {
			t.Errorf("%q: reconstructed %q", src, b)
		}
	}
}
