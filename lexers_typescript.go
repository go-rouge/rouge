// Copyright (c) the go-rouge/rouge authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rouge

import (
	onigmo "github.com/go-ruby-regexp/regexp"
)

// --- TypeScript, JSX and TSX ---
// A faithful transcription of Rouge::Lexers::Typescript,
// Rouge::Lexers::TypescriptCommon, Rouge::Lexers::JSX and Rouge::Lexers::TSX
// (rouge 5.0.0: typescript.rb, typescript/common.rb, jsx.rb, tsx.rb).
//
// The gem's inheritance is
//
//	Javascript <- JSX   <- TSX  (TSX also extends TypescriptCommon)
//	Javascript <- Typescript    (        extends TypescriptCommon)
//
// so all four share one machine. buildJavascript (lexers_javascript.go) is that
// machine; TypescriptCommon is tsCommon below, applied to whichever options
// struct is being built, exactly as `extend TypescriptCommon` is applied to
// whichever class names it.

// --- TypescriptCommon: the widened word sets and the two prepends ---

var tsKeywords = unionSet(jsKeywords, `is namespace static private protected
public implements readonly`)

var tsDeclarations = unionSet(jsDeclarations, `type abstract`)

var tsReserved = unionSet(jsReserved, `string any void number namespace module
declare default interface keyof`)

var tsBuiltins = unionSet(jsBuiltins, `Capitalize ConstructorParameters Exclude
Extract InstanceType Lowercase NonNullable Omit OmitThisParameter Parameters
Partial Pick Readonly Record Required ReturnType ThisParameterType ThisType
Uncapitalize Uppercase`)

// tsPrependRoot is TypescriptCommon's `base.prepend :root`.
//
// The `as` / `satisfies` rule emits ONE Keyword::Declaration for the whole
// match, trailing whitespace included: the gem passes a token, not a groups
// block, even though the pattern has two capture groups.
//
// ⛔ ITS THREE LOOKBEHINDS ARE DROPPED, BECAUSE THEY ASSERT NOTHING WHERE THE
// GEM EVALUATES THEM. The gem writes
//
//	%r{(?<![_$[:alnum:]])(?:(?<=\.\.\.)|(?<!\.))(?:(as)|(satisfies))\s+}
//
// and Rouge tests it with StringScanner#skip at the scan pointer, where a
// lookbehind cannot see the bytes before that pointer: a negative lookbehind
// always succeeds and a positive one always fails. So the guard reduces to
// "true", and the only thing that keeps `.as(` out is the trailing `\s+`.
//
// Measured, not assumed. The reference gem, fed
//
//	x.as y;  1as b;  ...as c;  z as d;
//
// marks `as ` as Keyword::Declaration on ALL FOUR lines -- including the one
// after a dot, which the comment in the gem's own source says is the negative
// example. Transcribing the lookbehinds literally would refuse three of the
// four and produce different HTML.
func tsPrependRoot(b *lexerBuilder) {
	b.rule(`[?][.]`, Punctuation).
		rule(`[?]{2}`, Operator).
		// Positive examples:
		//   const cat = { name: "Garfield" } satisfies Person;
		//   import {something as thingy} from 'module'
		//   export { foo as default }
		//   ...spreadOperator as const;
		rule(`(?:as|satisfies)\s+`, KeywordDeclaration)
}

// tsPrependStatement is TypescriptCommon's `base.prepend :statement`: an
// optional-property label, `name?: type`.
func tsPrependStatement(b *lexerBuilder) {
	b.groupsRuleT(`(`+jsID+`)(\??)(\s*)(:)`, []transition{push("expr_start")},
		NameLabel, Punctuation, Text, Punctuation)
}

// tsCommon applies TypescriptCommon to an options struct, the way
// `extend TypescriptCommon` applies it to a class: it widens four of the five
// word sets (the constants are left alone) and installs the two prepends.
//
// It ASSIGNS the two hooks rather than composing with whatever was there,
// because neither class that extends TypescriptCommon brings its own: JSX, the
// one class TSX also inherits from, prepends to :expr_start and nothing else.
func tsCommon(o jsOpts) jsOpts {
	o.keywords = tsKeywords
	o.declarations = tsDeclarations
	o.reserved = tsReserved
	o.builtins = tsBuiltins
	o.prependRoot = tsPrependRoot
	o.prependStatement = tsPrependStatement
	return o
}

var typescriptLexer = buildJavascript(tsCommon(jsOpts{
	tag:       "typescript",
	title:     "TypeScript",
	aliases:   []string{"ts"},
	filename:  []string{"*.ts", "*.d.ts", "*.cts", "*.mts"},
	constants: jsConstants,
}))

// --- JSX, and TSX on top of it ---

// jsxOpts carries what a JSX-family lexer adds to the JSX states themselves.
type jsxOpts struct {
	// prependElementName is TSX's `prepend :element_name`.
	prependElementName func(b *lexerBuilder)
	// extraStates defines states only the subclass has (TSX's :type).
	extraStates func(b *lexerBuilder)
}

// buildJSX fills in the JSX half of the machine -- the `start` push, the
// `prepend :expr_start` that makes a `<` open an element anywhere an
// expression may start, and the six states the gem's JSX class defines -- and
// hands the result to buildJavascript.
func buildJSX(o jsOpts, jo jsxOpts) *RegexLexer {
	o.startPush = append([]string{"expr_start"}, o.startPush...)
	o.prependExprStart = func(b *lexerBuilder) { b.mixin("tag") }
	o.extraStates = func(b *lexerBuilder) {
		b.state("tag").
			cb(`<`, func(l *lexState, m *onigmo.MatchData) {
				l.emit(Punctuation, m.Str(0))
				l.pos = m.End(0)
				l.push("tag_opening")
				l.push("element")
				l.push("element_name")
			})

		b.state("tag_opening").
			cb(`</`, func(l *lexState, m *onigmo.MatchData) {
				l.emit(Punctuation, m.Str(0))
				l.pos = m.End(0)
				l.goTo("element")
				l.push("element_name")
			}).
			mixin("tag").
			cb(`{`, func(l *lexState, m *onigmo.MatchData) {
				l.emit(LiteralStringInterpol, m.Str(0))
				l.pos = m.End(0)
				l.push("interpol")
				l.push("expr_start")
			}).
			// The literal text between tags is HTML, and the gem delegates it
			// to a CONTINUING HTML lexer instance (`delegate @html`).
			cb(`[^<{]+`, func(l *lexState, m *onigmo.MatchData) {
				l.subDelegate(htmlLexer, m.Str(0))
				l.pos = m.End(0)
			})

		b.state("element").
			mixin("comments_and_whitespace").
			cb(`/>`, func(l *lexState, m *onigmo.MatchData) {
				l.emit(Punctuation, m.Str(0))
				l.pos = m.End(0)
				l.pop(2)
			}).
			rule(`>`, Punctuation, pop()).
			cb(`{`, func(l *lexState, m *onigmo.MatchData) {
				l.emit(LiteralStringInterpol, m.Str(0))
				l.pos = m.End(0)
				l.push("interpol")
				l.push("expr_start")
			}).
			rule(`\w[\w-]*`, NameAttribute).
			rule(`=`, Punctuation).
			// The capture groups are load-bearing here: \1 is a backreference
			// to the opening quote, so the string ends on the SAME quote.
			rule(`(["']).*?(\1)`, LiteralString)

		b.state("element_name")
		if jo.prependElementName != nil {
			jo.prependElementName(b)
		}
		b.rule(`[A-Z]\w*`, NameClass).
			rule(`\w+`, NameTag).
			rule(`\.`, Punctuation).
			cb(``, func(l *lexState, m *onigmo.MatchData) { l.pop(1) })

		b.state("interpol").
			rule(`}`, LiteralStringInterpol, pop()).
			cb(`{`, func(l *lexState, m *onigmo.MatchData) {
				l.emit(Punctuation, m.Str(0))
				l.pos = m.End(0)
				l.push("interpol_inner")
				l.push("statement")
			}).
			mixin("root")

		b.state("interpol_inner").
			cb(`}`, func(l *lexState, m *onigmo.MatchData) {
				l.emit(Punctuation, m.Str(0))
				l.pos = m.End(0)
				l.goTo("statement")
			}).
			mixin("root")

		if jo.extraStates != nil {
			jo.extraStates(b)
		}
	}
	return buildJavascript(o)
}

var jsxLexer = buildJSX(jsOpts{
	tag:          "jsx",
	title:        "JSX",
	aliases:      []string{"react"},
	filename:     []string{"*.jsx"},
	keywords:     jsKeywords,
	declarations: jsDeclarations,
	reserved:     jsReserved,
	constants:    jsConstants,
	builtins:     jsBuiltins,
}, jsxOpts{})

var tsxLexer = buildJSX(tsCommon(jsOpts{
	tag:       "tsx",
	title:     "TSX",
	filename:  []string{"*.tsx"},
	constants: jsConstants,
}), jsxOpts{
	prependElementName: func(b *lexerBuilder) {
		b.groupsRuleT(`(\w+)(,)`, []transition{pop(), pop(), pop()}, NameOther, Punctuation).
			rule(`<`, Punctuation, push("type"))
	},
	extraStates: func(b *lexerBuilder) {
		b.state("type").
			mixin("object").
			rule(`>`, Punctuation, pop())
	},
})
