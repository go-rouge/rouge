// Copyright (c) the go-rouge/rouge authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rouge

import (
	onigmo "github.com/go-ruby-regexp/regexp"
)

// --- Make ---
// A faithful transcription of Rouge::Lexers::Make (rouge 5.0.0,
// lib/rouge/lexers/make.rb).
//
// Everything a recipe or a macro expansion contains is handed to the Shell
// lexer. The gem keeps ONE Shell instance per Makefile and delegates to it
// with `delegate @shell`, which calls continue_lex rather than lex: the shell
// lexer's state stack survives between chunks, so a quote opened in one chunk
// is still open in the next. subDelegate is that; `@shell.reset!` -- run at the
// start of a lex and at the end of every recipe body -- is subReset.
//
// makeFunctions is the gem's `self.functions`, joined with "|" in the order the
// gem lists them. The order is load-bearing: the alternation is tried left to
// right, so `filter` before `filter-out` means `$(filter-out ...)` matches
// `filter` and leaves `-out`, and `word` before `wordlist`/`words` does the
// same. That is what the gem does, so it is what this does.
const makeFunctions = `abspath|addprefix|addsuffix|and|basename|call|dir|error|eval|file|` +
	`filter|filter-out|findstring|firstword|flavor|foreach|if|join|lastword|` +
	`notdir|or|origin|patsubst|realpath|shell|sort|strip|subst|suffix|value|` +
	`warning|wildcard|word|wordlist|words`

// makeStop is the gem's local `stop`: the set of places a delegated shell run
// must be cut, so that macro expansions are tokenized by Make and not by the
// shell. Ruby interpolates it into the lookahead as a Regexp, which wraps it in
// a non-capturing group; that wrapping is written out here.
const makeStop = `[\$]{1,2}\(|[\$]{1,2}\{|\(|\)|\}|\\|$`

var makeLexer = func() *RegexLexer {
	b := newRegexLexer("make", "Make", "makefile", "mf", "gnumake", "bsdmake")
	b.filenames("*.make", "*.mak", "*.mk", "Makefile", "makefile", "Makefile.*",
		"GNUmakefile", "*,fe1")

	b.state("root").
		rule(`\s+`, Text).
		rule(`#.*?\n`, Comment).
		groupsRule(`([-s]?include)((?:[\t ]+[^\t\n #]+)+)`, Keyword, LiteralStringOther).
		groupsRule(`((?:ifn?def|ifn?eq|unexport)\b)([\t ]+)([^#\n]+)`, Keyword, Text, NameVariable).
		groupsRule(`(else\b)([\t ]+)((?:ifn?def|ifn?eq)\b)([\t ]+)([^#\n]+)`,
			Keyword, Text, Keyword, Text, NameVariable).
		rule(`(?:else|endif|endef|endfor)[\t ]*(?=[#\n])`, Keyword).
		groupsRuleT(`(export)([\t ]+)(?=[\w\${}()\t -]+\n)`, []transition{push("export")}, Keyword, Text).
		rule(`export[\t ]+`, Keyword).
		// assignment
		groupsRuleT(`(?m)(override\b)*([\t ]*)([\w${}().-]+)([\t ]*)([!?:+]?=)`,
			[]transition{push("shell_line")}, NameBuiltin, Text, NameVariable, Text, Operator).
		rule(`"(?:\\\\|\\.|[^"\\])*"`, LiteralStringDouble).
		rule(`'(?:\\\\|\\.|[^'\\])*'`, LiteralStringSingle).
		groupsRuleT(`([^\n:]+)(:+)([ \t]*)`, []transition{push("block_header")},
			NameLabel, Operator, Text).
		groupsRule(`(override\b)*([\t ])*(define)([\t ]+)([^#\n]+)`,
			NameBuiltin, Text, Keyword, Text, NameVariable).
		groupsRuleT(`(?m)(\$[({])([\t ]*)(`+makeFunctions+`)([\t ]+)`,
			[]transition{push("shell_expr")}, NameFunction, Text, NameBuiltin, Text)

	b.state("export").
		rule(`[\w\${}()-]`, NameVariable).
		rule(`\n`, Text, pop()).
		rule(`[\t ]+`, Text)

	b.state("block_header").
		rule(`[^,\\\n#]+`, NameFunction).
		rule(`,`, Punctuation).
		rule(`#.*?`, Comment).
		rule(`\\\n`, Text).
		rule(`\\.`, Text).
		rule(`\n`, Text, gotoState("block_body"))

	b.state("block_body").
		groupsRule(`(ifn?def|ifn?eq)([\t ]+)([^#\n]+)(#.*)?(\n)`,
			Keyword, Text, NameVariable, Comment, Text).
		groupsRule(`(else|endif)([\t ]*)(#.*)?(\n)`, Keyword, Text, Comment, Text).
		groupsRuleT(`(\t[\t ]*)([@-]?)`, []transition{push("shell_line")}, Text, Punctuation).
		cb(``, func(l *lexState, m *onigmo.MatchData) {
			l.pos = m.End(0)
			l.subReset(shellLexer)
			l.pop(1)
		})

	b.state("shell").
		// macro interpolation
		rule(`[\$]{1,2}[({]`, Punctuation, push("macro_expr")).
		// function invocation
		groupsRuleT(`(?m)(\$[({])([\t ]*)(`+makeFunctions+`)([\t ]+)`,
			[]transition{push("shell_expr")}, Punctuation, Text, NameBuiltin, Text).
		cb(`(?m)\\.`, makeDelegate).
		cb(`(?m).+?(?=(?:`+makeStop+`))`, makeDelegate).
		cb(makeStop, makeDelegate)

	b.state("macro_expr").
		rule(`[)}]`, Punctuation, pop()).
		rule(`\n`, Text, pop()).
		mixin("shell")

	b.state("shell_expr").
		cb(`[({]`, func(l *lexState, m *onigmo.MatchData) {
			l.subDelegate(shellLexer, m.Str(0))
			l.pos = m.End(0)
			l.stack = append(l.stack, l.stack[len(l.stack)-1])
		}).
		rule(`[)}]`, Punctuation, pop()).
		mixin("shell")

	b.state("shell_line").
		rule(`\n`, Text, pop()).
		mixin("shell")

	return b.done()
}()

// makeDelegate hands the whole match to the continuing Shell sub-lexer, which
// is the gem's `delegate @shell` with no explicit text.
func makeDelegate(l *lexState, m *onigmo.MatchData) {
	l.subDelegate(shellLexer, m.Str(0))
	l.pos = m.End(0)
}
