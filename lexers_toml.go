// Copyright (c) the go-rouge/rouge authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rouge

// --- TOML ---
// A faithful transcription of Rouge::Lexers::TOML (rouge 5.0.0,
// lib/rouge/lexers/toml.rb).
//
// Two spellings differ from the gem's source without differing in meaning:
//
//   - the gem writes `rule %r//, Token, :value` in :array. `Token` there is
//     Rouge::Token itself, and yield_token drops a nil/empty value, so the rule
//     emits NOTHING and only pushes :value. A nil token here is that same
//     "emit nothing".
//   - capture groups the gem never reads (`(\.\d+)?` in the time rule) are
//     written non-capturing, which cannot change what the pattern matches.
var tomlLexer = func() *RegexLexer {
	b := newRegexLexer("toml", "TOML")
	b.filenames("*.toml", "Pipfile", "poetry.lock")

	b.state("root").
		mixin("whitespace").
		mixin("key").
		groupsRuleT(`(=)(\s*)`, []transition{push("value")}, Operator, TextWhitespace).
		rule(`\[\[?`, Keyword, push("table_key"))

	b.state("key").
		rule(`[A-Za-z0-9_-]+`, Name).
		rule(`"`, LiteralString, push("dq")).
		rule(`'`, LiteralString, push("sq")).
		rule(`\.`, Punctuation)

	b.state("table_key").
		rule(`[A-Za-z0-9_-]+`, Name).
		rule(`"`, LiteralString, push("dq")).
		rule(`'`, LiteralString, push("sq")).
		rule(`\.`, Keyword).
		rule(`\]\]?`, Keyword, pop()).
		rule(`[ \t]+`, TextWhitespace)

	b.state("value").
		rule(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z`, LiteralDate, pop()).
		rule(`\d\d:\d\d:\d\d(?:\.\d+)?`, LiteralDate, pop()).
		rule(`[+-]?\d+(?:_\d+)*\.\d+(?:_\d+)*(?:[eE][+-]?\d+(?:_\d+)*)?`, LiteralNumberFloat, pop()).
		rule(`[+-]?\d+(?:_\d+)*[eE][+-]?\d+(?:_\d+)*`, LiteralNumberFloat, pop()).
		rule(`[+-]?(?:nan|inf)`, LiteralNumberFloat, pop()).
		rule(`0x\h+(?:_\h+)*`, LiteralNumberHex, pop()).
		rule(`0o[0-7]+(?:_[0-7]+)*`, LiteralNumberOct, pop()).
		rule(`0b[01]+(?:_[01]+)*`, LiteralNumberBin, pop()).
		rule(`[+-]?\d+(?:_\d+)*`, LiteralNumberInteger, pop()).
		rule(`"""`, LiteralString, pop(), push("mdq")).
		rule(`"`, LiteralString, pop(), push("dq")).
		rule(`'''`, LiteralString, pop(), push("msq")).
		rule(`'`, LiteralString, pop(), push("sq")).
		rule(`(?:true|false)`, KeywordConstant, pop()).
		rule(`\[`, Punctuation, pop(), push("array")).
		rule(`\{`, Punctuation, pop(), push("inline"))

	b.state("dq").
		rule(`"`, LiteralString, pop()).
		rule(`\n`, Error, pop()).
		mixin("esc_str").
		rule(`[^\\"\n]+`, LiteralString)

	b.state("mdq").
		rule(`"""`, LiteralString, pop()).
		mixin("esc_str").
		rule(`[^\\"]+`, LiteralString).
		rule(`"+`, LiteralString)

	b.state("sq").
		rule(`'`, LiteralString, pop()).
		rule(`\n`, Error, pop()).
		rule(`[^'\n]+`, LiteralString)

	b.state("msq").
		rule(`'''`, LiteralString, pop()).
		rule(`[^']+`, LiteralString).
		rule(`'+`, LiteralString)

	b.state("esc_str").
		rule(`\\[0t\tn\n "\\r]`, LiteralStringEscape)

	b.state("array").
		mixin("whitespace").
		rule(`,`, Punctuation).
		rule(`\]`, Punctuation, pop()).
		rule(``, nil, push("value"))

	b.state("inline").
		rule(`[ \t]+`, TextWhitespace).
		mixin("key").
		groupsRuleT(`(=)(\s*)`, []transition{push("value")}, Punctuation, TextWhitespace).
		rule(`,`, Punctuation).
		rule(`\}`, Punctuation, pop())

	b.state("whitespace").
		rule(`\s+`, Text).
		rule(`#.*?$`, Comment)

	return b.done()
}()
