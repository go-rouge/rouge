// Copyright (c) the go-rouge/rouge authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rouge

import "testing"

// texStream lexes src and renders the token stream as a slice of
// "Qualname\tvalue" lines, the compact shape used to assert TeX tokenization
// against the reference rouge gem (rouge 4.7.0, lib/rouge/lexers/tex.rb, whose
// tokenization is identical to the 5.0.0 line this port targets).
func texStream(src string) []string {
	tv := texLexer.Lex(src)
	out := make([]string, len(tv))
	for i, t := range tv {
		out[i] = t.Token.Qualname() + "\t" + t.Value
	}
	return out
}

// TestTeXTokens drives every rule and state of the TeX lexer and asserts the
// exact (token, value) stream, matching the reference gem token for token. Each
// case's want slice was captured from Rouge::Lexers::TeX.
func TestTeXTokens(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "comment",
			src:  "% a comment\n",
			want: []string{"Comment\t% a comment", "Text\t\n"},
		},
		{
			name: "control word with braces",
			src:  "\\section{Hi}\n",
			want: []string{"Keyword\t\\section", "Punctuation\t{", "Text\tHi", "Punctuation\t}", "Text\t\n"},
		},
		{
			name: "begin and end environment",
			src:  "\\begin{itemize}\n\\end{itemize}",
			want: []string{"Name.Tag\t\\begin{itemize}", "Text\t\n", "Name.Tag\t\\end{itemize}"},
		},
		{
			name: "starred command",
			src:  "\\section*{Starred}",
			want: []string{"Keyword\t\\section*", "Punctuation\t{", "Text\tStarred", "Punctuation\t}"},
		},
		{
			name: "optional argument",
			src:  "\\includegraphics[width=1cm]{a}",
			want: []string{"Keyword\t\\includegraphics", "Name.Attribute\t[width=1cm]", "Punctuation\t{", "Text\ta", "Punctuation\t}"},
		},
		{
			name: "specials and control symbol",
			src:  "a \\% b & c _ d ^ e {f}",
			want: []string{
				"Text\ta ", "Keyword\t\\%", "Text\t b ", "Punctuation\t&", "Text\t c ",
				"Punctuation\t_", "Text\t d ", "Punctuation\t^", "Text\t e ",
				"Punctuation\t{", "Text\tf", "Punctuation\t}",
			},
		},
		{
			name: "verb pipe delimiter",
			src:  "\\verb|foo|",
			want: []string{"Name.Builtin\t\\verb", "Keyword.Pseudo\t|", "Literal.String.Other\tfoo", "Keyword.Pseudo\t|"},
		},
		{
			name: "verb hash delimiter",
			src:  "\\verb#hi#",
			want: []string{"Name.Builtin\t\\verb", "Keyword.Pseudo\t#", "Literal.String.Other\thi", "Keyword.Pseudo\t#"},
		},
		{
			name: "inline math dollar",
			src:  "$x+1$",
			want: []string{"Punctuation\t$", "Name.Builtin\tx", "Operator\t+", "Literal.Number\t1", "Punctuation\t$"},
		},
		{
			name: "display math double dollar",
			src:  "$$a-b$$",
			want: []string{"Punctuation\t$$", "Name.Builtin\ta", "Operator\t-", "Name.Builtin\tb", "Punctuation\t$$"},
		},
		{
			name: "display math bracket delimiters",
			src:  "\\[ x=2 \\]",
			want: []string{"Punctuation\t\\[", "Name.Builtin\t x", "Operator\t=", "Literal.Number\t2", "Name.Builtin\t ", "Punctuation\t\\]"},
		},
		{
			name: "inline math paren delimiters",
			src:  "\\( y \\)",
			want: []string{"Punctuation\t\\(", "Name.Builtin\t y ", "Punctuation\t\\)"},
		},
		{
			name: "line break control symbol",
			src:  "\\\\",
			want: []string{"Keyword\t\\\\"},
		},
		{
			name: "thin space control symbol",
			src:  "\\,",
			want: []string{"Keyword\t\\,"},
		},
		{
			name: "plain text",
			src:  "plain text",
			want: []string{"Text\tplain text"},
		},
		{
			name: "math control word and subscript",
			src:  "$\\alpha + x_i$",
			want: []string{
				"Punctuation\t$", "Name.Variable\t\\alpha", "Name.Builtin\t ", "Operator\t+",
				"Name.Builtin\t x", "Punctuation\t_", "Name.Builtin\ti", "Punctuation\t$",
			},
		},
		{
			name: "lone dollar inside display math",
			src:  "$$ a $ b $$",
			want: []string{"Punctuation\t$$", "Name.Builtin\t a $ b ", "Punctuation\t$$"},
		},
		{
			name: "braces inside math",
			src:  "$ {x} $",
			want: []string{
				"Punctuation\t$", "Name.Builtin\t ", "Punctuation\t{", "Name.Builtin\tx",
				"Punctuation\t}", "Name.Builtin\t ", "Punctuation\t$",
			},
		},
		{
			name: "comment inside math",
			src:  "$x % c\n$",
			want: []string{"Punctuation\t$", "Name.Builtin\tx ", "Comment\t% c", "Name.Builtin\t\n", "Punctuation\t$"},
		},
		{
			name: "bare command at end of input",
			src:  "\\emph",
			want: []string{"Keyword\t\\emph"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := texStream(c.src)
			if len(got) != len(c.want) {
				t.Fatalf("token count = %d, want %d\n got: %#v\nwant: %#v", len(got), len(c.want), got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("token %d:\n got: %q\nwant: %q", i, got[i], c.want[i])
				}
			}
		})
	}
}

// TestTeXDetect covers Rouge::Lexers::TeX.detect?: a preamble control word (with
// optional leading whitespace) is TeX, arbitrary prose is not.
func TestTeXDetect(t *testing.T) {
	for _, c := range []struct {
		text string
		want bool
	}{
		{"\\documentclass{article}\n", true}, // leading control word
		{"  \\input{chapter}\n", true},       // leading whitespace then control word
		{"\\ProvidesPackage{p}", true},       // another preamble word
		{"hello world", false},               // prose, no marker
		{"\\section{x}", false},              // a control word, but not a preamble one
	} {
		if got := Guess(c.text); (got.Tag() == "latex") != c.want {
			t.Errorf("Guess(%q).Tag() = %q, want latex==%v", c.text, got.Tag(), c.want)
		}
	}
}

// TestTeXRegistration checks the tag and aliases resolve to this lexer.
func TestTeXRegistration(t *testing.T) {
	if FindLexer("latex") != texLexer {
		t.Error(`FindLexer("latex") did not resolve to the TeX lexer`)
	}
	for _, alias := range []string{"tex", "TeX", "LaTeX"} {
		if FindLexer(alias) != texLexer {
			t.Errorf("FindLexer(%q) did not resolve to the TeX lexer", alias)
		}
	}
	if texLexer.Tag() != "latex" || texLexer.Title() != "TeX" {
		t.Errorf("tag/title = %q/%q, want latex/TeX", texLexer.Tag(), texLexer.Title())
	}
}
