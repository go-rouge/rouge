// Copyright (c) the go-rouge/rouge authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rouge

import (
	"strings"

	onigmo "github.com/go-ruby-regexp/regexp"
)

// --- R ---
// A transcription of Rouge::Lexers::R (lib/rouge/lexers/r.rb, rouge v5.1.0).
// The gem's mimetypes ('text/x-r-source', 'text/x-r', 'text/x-R',
// 'application/x-r') have no counterpart in this library's lexer metadata.
//
// One deviation from the text of r.rb, measured against the gem (see the
// README, "Documented simplifications"): the primitive-function rule is written
// without its leading `\b` and `(?<!.)`. Rouge scans with StringScanner#skip,
// where the scan pointer is the start of the string for both assertions: `\b`
// is true before a word character and `(?<!.)` is always true. The rule's
// alternatives all start with a letter, so `\b` is a no-op there and `(?<!.)`
// constrains nothing -- `x+1abs(2)` gives `abs` as Name::Function in the gem.

var rLexer = func() *RegexLexer {
	words := func(s string) map[string]bool {
		m := map[string]bool{}
		for _, w := range strings.Fields(s) {
			m[w] = true
		}
		return m
	}

	keywords := words(`if else for while repeat in next break function`)

	keywordConstants := words(`
		NULL Inf TRUE FALSE NaN NA
		NA_integer_ NA_real_ NA_complex_ NA_character_`)

	builtinConstants := words(`LETTERS letters month.abb month.name pi T F`)

	// These are all the functions in `base` that are implemented as a
	// `.Primitive`, minus those functions that are also keywords. The gem joins
	// them with `|` WITHOUT escaping, so the dots in `as.call` match any
	// character; that is kept.
	const primitiveFunctions = `
		abs acos acosh all any anyNA Arg as.call as.character
		as.complex as.double as.environment as.integer as.logical
		as.null.default as.numeric as.raw asin asinh atan atanh attr
		attributes baseenv browser c call ceiling class Conj cos cosh
		cospi cummax cummin cumprod cumsum digamma dim dimnames
		emptyenv exp expression floor forceAndCall gamma gc.time
		globalenv Im interactive invisible is.array is.atomic is.call
		is.character is.complex is.double is.environment is.expression
		is.finite is.function is.infinite is.integer is.language
		is.list is.logical is.matrix is.na is.name is.nan is.null
		is.numeric is.object is.pairlist is.raw is.recursive is.single
		is.symbol lazyLoadDBfetch length lgamma list log max min
		missing Mod names nargs nzchar oldClass on.exit pos.to.env
		proc.time prod quote range Re rep retracemem return round
		seq_along seq_len seq.int sign signif sin sinh sinpi sqrt
		standardGeneric substitute sum switch tan tanh tanpi tracemem
		trigamma trunc unclass untracemem UseMethod xtfrm`

	b := newRegexLexer("r", "R", "R", "s", "S")
	b.filenames("*.R", "*.r", ".Rhistory", ".Rprofile")
	b.detectWith(shebangDetector("Rscript"))

	b.state("root").
		rule(`#'.*?$`, CommentDoc).
		rule(`#.*?$`, CommentSingle).
		rule(`(?m:\s+)`, TextWhitespace).
		rule("`[^`]+?`", Name).
		rule(`(?m:'(\\.|.)*?')`, LiteralStringSingle).
		rule(`(?m:"(\\.|.)*?")`, LiteralStringDouble).
		rule(`%[^%]*?%`, Operator).
		rule(`0[xX][a-fA-F0-9]+([pP][0-9]+)?[Li]?`, LiteralNumberHex).
		rule(`[+-]?(\d+([.]\d+)?|[.]\d+)([eE][+-]?\d+)?[Li]?`, LiteralNumber).
		// Only recognize built-in functions when they are actually used as a
		// function call, i.e. followed by an opening parenthesis.
		// `Name::Builtin` would be more logical, but is usually not
		// highlighted specifically; thus use `Name::Function`.
		rule(`(`+strings.Join(strings.Fields(primitiveFunctions), "|")+`)(?=\()`, NameFunction).
		cb(`(?:(?:[[:alpha:]]|[.][._[:alpha:]])[._[:alnum:]]*)|[.]`, func(l *lexState, m *onigmo.MatchData) {
			word := m.Str(0)
			switch {
			case keywords[word]:
				l.emit(Keyword, word)
			case keywordConstants[word]:
				l.emit(KeywordConstant, word)
			case builtinConstants[word]:
				l.emit(NameBuiltin, word)
			default:
				l.emit(Name, word)
			}
			l.pos = m.End(0)
		}).
		rule(`[\[\]{}();,]`, Punctuation).
		rule(`[-<>?*+^/!=~$@:%&|]`, Operator)

	return b.done()
}()
