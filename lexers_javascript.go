// Copyright (c) the go-rouge/rouge authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rouge

import (
	"strings"

	onigmo "github.com/go-ruby-regexp/regexp"
)

// --- JavaScript, and the family built on it ---
// A faithful transcription of Rouge::Lexers::Javascript (rouge 5.0.0),
// including its regex / template-string / object-literal / ternary state
// machine. The keyword / declaration / reserved / constant / builtin word sets
// are the gem's.
//
// The gem writes `class JSX < Javascript`, `class Typescript < Javascript` and
// `class TSX < JSX`, where the subclasses widen the word sets and use
// `prepend :state` to put rules IN FRONT of the inherited ones. Go has no such
// inheritance, so the shared machine is built here by buildJavascript, which
// takes the word sets and the hooks a subclass would use. JavaScript passes no
// hooks; TypeScript, JSX and TSX (lexers_typescript.go) pass the ones the gem
// gives them. One builder is what makes "TSX is JSX plus these rules" a fact
// about the code rather than a claim in a comment.

var jsKeywords = stringSet(`async await break case catch continue debugger
default delete do else export finally from for if import in instanceof new of
return super switch this throw try typeof void while yield`)

var jsDeclarations = stringSet(`var let const with function class extends
constructor get set static`)

var jsReserved = stringSet(`enum implements interface package private protected
public`)

var jsConstants = stringSet(`true false null NaN Infinity undefined`)

var jsBuiltins = stringSet(`Array Boolean Date Error Function Math netscape
Number Object Packages RegExp String sun decodeURI decodeURIComponent encodeURI
encodeURIComponent Error eval isFinite isNaN parseFloat parseInt document window
navigator self global Promise Set Map WeakSet WeakMap Symbol Proxy Reflect
Int8Array Uint8Array Uint8ClampedArray Int16Array Uint16Array Uint16ClampedArray
Int32Array Uint32Array Uint32ClampedArray Float32Array Float64Array DataView
ArrayBuffer`)

// jsID is the gem's id_regex: /[\p{L}\p{Nl}$_][\p{Word}]*/.
const jsID = `[\p{L}\p{Nl}$_][\p{Word}]*`

// jsOpts carries what a JavaScript-family lexer differs in.
type jsOpts struct {
	tag      string
	title    string
	aliases  []string
	filename []string

	keywords     map[string]bool
	declarations map[string]bool
	reserved     map[string]bool
	constants    map[string]bool
	builtins     map[string]bool

	// detect is the content sniffer (the gem's self.detect?).
	detect func(string) bool
	// startPush is the gem's `start do push :x end`.
	startPush []string
	// prependRoot, prependExprStart and prependStatement add rules at the
	// FRONT of those states, which is what the gem's `prepend` does.
	prependRoot      func(b *lexerBuilder)
	prependExprStart func(b *lexerBuilder)
	prependStatement func(b *lexerBuilder)
	// extraStates defines states only a subclass has.
	extraStates func(b *lexerBuilder)
}

func buildJavascript(o jsOpts) *RegexLexer {
	b := newRegexLexer(o.tag, o.title, o.aliases...)
	b.filenames(o.filename...)
	if o.detect != nil {
		b.detectWith(o.detect)
	}
	b.start(o.startPush...)

	b.state("multiline_comment").
		rule(`[*]/`, CommentMultiline, pop()).
		rule(`[^*/]+`, CommentMultiline).
		rule(`[*/]`, CommentMultiline)

	b.state("comments_and_whitespace").
		rule(`\s+`, Text).
		rule(`<!--`, Comment).
		rule(`//.*?$`, CommentSingle).
		rule(`/[*]`, CommentMultiline, push("multiline_comment"))

	b.state("expr_start")
	if o.prependExprStart != nil {
		o.prependExprStart(b)
	}
	b.mixin("comments_and_whitespace").
		cb(`/`, func(l *lexState, m *onigmo.MatchData) {
			l.emit(LiteralStringRegex, m.Str(0))
			l.pos = m.End(0)
			l.goTo("regex")
		}).
		cb(`[{]`, func(l *lexState, m *onigmo.MatchData) {
			l.emit(Punctuation, m.Str(0))
			l.pos = m.End(0)
			l.goTo("object")
		}).
		rule(``, Text, pop())

	b.state("regex").
		cb(`/`, func(l *lexState, m *onigmo.MatchData) {
			l.emit(LiteralStringRegex, m.Str(0))
			l.pos = m.End(0)
			l.goTo("regex_end")
		}).
		rule(`[^/]\n`, Error, pop()).
		rule(`\n`, Error, pop()).
		rule(`\[\^`, LiteralStringEscape, push("regex_group")).
		rule(`\[`, LiteralStringEscape, push("regex_group")).
		rule(`\\.`, LiteralStringEscape).
		rule(`[(][?][:=<!]`, LiteralStringEscape).
		rule(`[{][\d,]+[}]`, LiteralStringEscape).
		rule(`[()?]`, LiteralStringEscape).
		rule(`.`, LiteralStringRegex)

	b.state("regex_end").
		rule(`[gimuy]+`, LiteralStringRegex, pop()).
		cb(``, func(l *lexState, m *onigmo.MatchData) { l.pop(1) })

	b.state("regex_group").
		rule(`/`, LiteralStringEscape).
		cb(`[^/]\n`, func(l *lexState, m *onigmo.MatchData) {
			l.emit(Error, m.Str(0))
			l.pos = m.End(0)
			l.pop(2)
		}).
		rule(`\]`, LiteralStringEscape, pop()).
		rule(`\\.`, LiteralStringEscape).
		rule(`.`, LiteralStringRegex)

	b.state("bad_regex").
		rule(`[^\n]+`, Error, pop())

	b.state("root")
	if o.prependRoot != nil {
		o.prependRoot(b)
	}
	b.rule(`(?m)\A\s*#!.*?\n`, CommentPreproc, push("statement")).
		rule(`(?<=\n)(?=\s|/|<!--)`, Text, push("expr_start")).
		mixin("comments_and_whitespace").
		rule(`(?x)\+\+ | -- | ~ | \?\?=? | && | \|\| | \\(?=\n) | << | >>>? | === | !== `, Operator, push("expr_start")).
		rule(`[-<>+*%&|^/!=]=?`, Operator, push("expr_start")).
		rule(`[(\[,]`, Punctuation, push("expr_start")).
		rule(`;`, Punctuation, push("statement")).
		rule(`[)\].]`, Punctuation).
		rule("`", LiteralStringDouble, push("template_string")).
		rule(`[?][.]`, Punctuation).
		cb(`[?]`, func(l *lexState, m *onigmo.MatchData) {
			l.emit(Punctuation, m.Str(0))
			l.pos = m.End(0)
			l.push("ternary")
			l.push("expr_start")
		}).
		groupsRuleT(`(@)(\w+)?`, []transition{push("expr_start")}, Punctuation, NameDecorator).
		groupsRuleT(`(class)((?:\s|\\\s)+)`, []transition{push("classname")}, KeywordDeclaration, Text).
		rule(`(?m)[\p{Nl}$_]*\p{Lu}[\p{Word}]*[ \t]*(?=(?:\(.*\)))`, NameClass).
		groupsRule(`(function)((?:\s|\\\s)+)(`+jsID+`)`, KeywordDeclaration, Text, NameFunction).
		rule(`function(?=(?:\(.*\)))`, KeywordDeclaration).
		cb(`(?m)(#?`+jsID+`)[ \t]*(?=(?:\(.*\)))`, func(l *lexState, m *onigmo.MatchData) {
			if o.keywords[m.Str(1)] {
				l.emit(Keyword, m.Str(0))
			} else {
				l.emit(NameFunction, m.Str(0))
			}
			l.pos = m.End(0)
		}).
		rule(`[{}]`, Punctuation, push("statement")).
		cb(`#?`+jsID, func(l *lexState, m *onigmo.MatchData) {
			w := m.Str(0)
			switch {
			case o.keywords[w]:
				l.emit(Keyword, w)
				l.pos = m.End(0)
				l.push("expr_start")
				return
			case o.declarations[w]:
				l.emit(KeywordDeclaration, w)
				l.pos = m.End(0)
				l.push("expr_start")
				return
			case o.reserved[w]:
				l.emit(KeywordReserved, w)
			case o.constants[w]:
				l.emit(KeywordConstant, w)
			case o.builtins[w]:
				l.emit(NameBuiltin, w)
			default:
				l.emit(NameOther, w)
			}
			l.pos = m.End(0)
		}).
		rule(`[0-9][0-9]*\.[0-9]+(?:[eE][0-9]+)?[fd]?`, LiteralNumberFloat).
		rule(`(?i)0x[0-9a-fA-F]+`, LiteralNumberHex).
		rule(`(?i)0o[0-7][0-7_]*`, LiteralNumberOct).
		rule(`(?i)0b[01][01_]*`, LiteralNumberBin).
		rule(`[0-9]+`, LiteralNumberInteger).
		rule(`"`, LiteralStringDelimiter, push("dq")).
		rule(`'`, LiteralStringDelimiter, push("sq")).
		rule(`:`, Punctuation)

	b.state("dq").
		rule(`\\[\\nrt"]?`, LiteralStringEscape).
		rule(`[^\\"]+`, LiteralStringDouble).
		rule(`"`, LiteralStringDelimiter, pop())

	b.state("sq").
		rule(`\\[\\nrt']?`, LiteralStringEscape).
		rule(`[^\\']+`, LiteralStringSingle).
		rule(`'`, LiteralStringDelimiter, pop())

	b.state("classname").
		groupsRule(`(`+jsID+`)((?:\s|\\\s)+)(extends)((?:\s|\\\s)+)`, NameClass, Text, KeywordDeclaration, Text).
		rule(jsID, NameClass, pop())

	// braced parts that aren't object literals
	b.state("statement")
	if o.prependStatement != nil {
		o.prependStatement(b)
	}
	b.cb(`case\b`, func(l *lexState, m *onigmo.MatchData) {
		l.emit(Keyword, m.Str(0))
		l.pos = m.End(0)
		l.goTo("expr_start")
	}).
		groupsRule(`(`+jsID+`)(\s*)(:)`, NameLabel, Text, Punctuation).
		mixin("expr_start")

	// object literals
	b.state("object").
		mixin("comments_and_whitespace").
		cb(`[{]`, func(l *lexState, m *onigmo.MatchData) {
			l.emit(Punctuation, m.Str(0))
			l.pos = m.End(0)
			l.push(l.stack[len(l.stack)-1].name)
		}).
		cb(`[}]`, func(l *lexState, m *onigmo.MatchData) {
			l.emit(Punctuation, m.Str(0))
			l.pos = m.End(0)
			l.goTo("statement")
		}).
		groupsRuleT(`(`+jsID+`)(\s*)(:)`, []transition{push("expr_start")}, NameAttribute, Text, Punctuation).
		rule(`:`, Punctuation).
		mixin("root")

	// ternary expressions, where <id>: is not a label!
	b.state("ternary").
		cb(`:`, func(l *lexState, m *onigmo.MatchData) {
			l.emit(Punctuation, m.Str(0))
			l.pos = m.End(0)
			l.goTo("expr_start")
		}).
		mixin("root")

	// template strings
	b.state("template_string").
		rule(`[$]{`, Punctuation, push("template_string_expr")).
		rule("`", LiteralStringDouble, pop()).
		rule(`\\[$`+"`"+`\\]`, LiteralStringEscape).
		rule("[^$`\\\\]+", LiteralStringDouble).
		rule("[\\\\$]", LiteralStringDouble)

	b.state("template_string_expr").
		rule(`}`, Punctuation, pop()).
		mixin("root")

	if o.extraStates != nil {
		o.extraStates(b)
	}
	return b.done()
}

var javascriptLexer = buildJavascript(jsOpts{
	tag:          "javascript",
	title:        "JavaScript",
	aliases:      []string{"js"},
	filename:     []string{"*.cjs", "*.js", "*.mjs"},
	keywords:     jsKeywords,
	declarations: jsDeclarations,
	reserved:     jsReserved,
	constants:    jsConstants,
	builtins:     jsBuiltins,
	detect: func(text string) bool {
		if strings.HasPrefix(text, "#!") {
			line := text
			if nl := strings.IndexByte(text, '\n'); nl >= 0 {
				line = text[:nl]
			}
			return strings.Contains(line, "node") || strings.Contains(line, "jsc")
		}
		return false
	},
})
