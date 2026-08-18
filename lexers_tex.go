// Copyright (c) the go-rouge/rouge authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rouge

import (
	onigmo "github.com/go-ruby-regexp/regexp"
)

// --- TeX / LaTeX ---
// A faithful transcription of Rouge::Lexers::TeX (the gem's lib/rouge/lexers/
// tex.rb), which highlights the TeX/LaTeX typesetting language. The gem tags the
// lexer "tex" with "latex" among its aliases; this port makes "latex" the
// primary tag (so rouge consumers resolve it under that name) and keeps "tex"
// and the case variants as aliases. Both names resolve to this lexer either way,
// because registerLexer registers a lexer under its tag and every alias.
//
// The gem's shared `command` pattern is /\\([a-z]+|\s+|.)/i: a backslash
// followed by either a run of letters (a control word, case-insensitive), a run
// of whitespace, or any single other character (a control symbol such as \\, \%
// or \,). The capture group only exists so the block form can read it; every
// rule here emits the whole match, so the group is inlined as non-capturing.
const texCommand = `\\(?:[a-zA-Z]+|\s+|.)`

var texLexer = func() *RegexLexer {
	b := newRegexLexer("latex", "TeX", "tex", "TeX", "LaTeX")
	b.filenames("*.tex", "*.aux", "*.toc", "*.sty", "*.cls")
	// Rouge's self.detect?: a document that opens (after optional leading
	// whitespace) with one of these preamble control words is TeX/LaTeX.
	b.detectWith(func(text string) bool {
		if re, err := onigmo.Compile(`\A\s*\\(?:documentclass|input|documentstyle|relax|ProvidesPackage|ProvidesClass)`); err == nil && re.MatchString(text) {
			return true
		}
		return false
	})

	// general is mixed into root and math: comments and the bare special
	// characters { } & _ ^.
	b.state("general").
		rule(`%.*$`, Comment).
		rule(`[{}&_^]`, Punctuation)

	b.state("root").
		rule(`\\\[`, Punctuation, push("displaymath")).
		rule(`\\\(`, Punctuation, push("inlinemath")).
		rule(`\$\$`, Punctuation, push("displaymath")).
		rule(`\$`, Punctuation, push("inlinemath")).
		rule(`\\(?:begin|end)\{.*?\}`, NameTag).
		groupsRule(`(\\verb)\b(\S)(.*?)(\2)`, NameBuiltin, KeywordPseudo, LiteralStringOther, KeywordPseudo).
		rule(texCommand, Keyword, push("command")).
		mixin("general").
		rule(`[^\\$%&_^{}]+`, Text)

	// math is mixed into inlinemath and displaymath. A control sequence inside
	// math is a Name::Variable; digits are numbers; the listed characters are
	// operators; everything else is a Name::Builtin run.
	b.state("math").
		rule(texCommand, NameVariable).
		mixin("general").
		rule(`[0-9]+`, LiteralNumber).
		rule(`[-=!+*\/()\[\]]`, Operator).
		rule(`[^=!+*\/()\[\]\\$%&_^{}0-9-]+`, NameBuiltin)

	b.state("inlinemath").
		rule(`\\\)`, Punctuation, pop()).
		rule(`\$`, Punctuation, pop()).
		mixin("math")

	b.state("displaymath").
		rule(`\\\]`, Punctuation, pop()).
		rule(`\$\$`, Punctuation, pop()).
		rule(`\$`, NameBuiltin).
		mixin("math")

	// command handles a control word's trailing optional argument ([...]) and
	// star form (*); anything else pops straight back via the empty rule.
	b.state("command").
		rule(`\[.*?\]`, NameAttribute).
		rule(`\*`, Keyword).
		rule(``, nil, pop())

	return b.done()
}()
