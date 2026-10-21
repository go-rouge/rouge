// Copyright (c) the go-rouge/rouge authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rouge

import "testing"

// TestXMLDetectRefusesAnHTMLDoctype drives the first arm of the XML content
// detector. Guess cannot reach it: the HTML lexer is registered first and its
// own detector claims `<!DOCTYPE html>` before XML is ever asked. The arm is
// still the gem's behaviour (`return false if text.doctype?(/html/)`) and it is
// what keeps a future reordering of the registry from handing XHTML to XML, so
// it is driven here directly.
func TestXMLDetectRefusesAnHTMLDoctype(t *testing.T) {
	detect := xmlLexer.detect
	if detect == nil {
		t.Fatal("the XML lexer has no content detector")
	}
	for _, c := range []struct {
		text string
		want bool
	}{
		{"<!DOCTYPE html>\n<p>hi</p>", false},                     // refused: html doctype
		{"<?xml version=\"1.0\"?>\n<!DOCTYPE html>\n<p/>", false}, // refused through the <?xml?> prologue too
		{"<?xml version=\"1.0\"?>\n<root/>", true},                // claimed: xml declaration
		{"<!DOCTYPE plist PUBLIC \"-//Apple//DTD\">\n<plist/>", true},
		{"just prose", false},
		{"<root/>", false}, // a bare element is not claimed
	} {
		if got := detect(c.text); got != c.want {
			t.Errorf("detect(%q) = %v, want %v", c.text, got, c.want)
		}
	}
}

// TestSubDelegateContinuesTheSubLexer pins the contract subDelegate exists for:
// a delegated lexer keeps its state stack between chunks (Rouge's
// `delegate @sub`, which calls continue_lex), and subReset puts it back to its
// starting stack -- including any state the delegate's own `start` block
// pushes.
//
// No lexer that is CURRENTLY delegated to has such a `start` push (Shell and
// HTML both begin in :root), so the corpus cannot reach that line; the JSX
// lexer, which does push :expr_start, is used here as the probe.
func TestSubDelegateContinuesTheSubLexer(t *testing.T) {
	// A continuing delegate: the double quote opened in the first chunk is
	// still open in the second, so "b" arrives as string content and not as a
	// bare word. A fresh lex of each chunk could not do that.
	ls := &lexState{rl: dockerLexer, stack: []*stateDef{dockerLexer.getState("root")}}
	ls.subDelegate(shellLexer, `echo "a`)
	if n := len(ls.sub.stack); n < 2 {
		t.Fatalf("after an unterminated quote the sub-lexer is at depth %d, want >= 2", n)
	}
	ls.subDelegate(shellLexer, `b"`)
	var joined string
	for _, tv := range ls.out {
		joined += tv.Value
	}
	if joined != `echo "ab"` {
		t.Errorf("delegated chunks reconstructed as %q", joined)
	}
	// "b" arrived while the string was still open, so it is string content. A
	// fresh lex of the second chunk would have made it a bare word.
	if tok := find(t, ls.out, "b"); tok != LiteralStringDouble {
		t.Errorf("the second chunk's %q is %s, want Literal.String.Double", "b", tok.Qualname())
	}
	// Closing the quote pops the string state again.
	if n := len(ls.sub.stack); n != 1 {
		t.Errorf("after the closing quote the sub-lexer is at depth %d, want 1", n)
	}

	// subReset restores the delegate's STARTING stack, which for a lexer with
	// a `start` push is deeper than just :root.
	ls.subReset(jsxLexer)
	if n := len(ls.sub.stack); n != 2 {
		t.Fatalf("subReset gave a stack of %d state(s), want 2 (root + the start push)", n)
	}
	if name := ls.sub.stack[1].name; name != "expr_start" {
		t.Errorf("subReset pushed %q, want the lexer's start state expr_start", name)
	}
}
