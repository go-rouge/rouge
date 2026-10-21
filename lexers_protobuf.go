// Copyright (c) the go-rouge/rouge authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rouge

// --- Protobuf ---
// A faithful transcription of Rouge::Lexers::Protobuf (rouge 5.0.0,
// lib/rouge/lexers/protobuf.rb).
//
// Two of the gem's patterns look like typos and are transcribed as written,
// because the gem's output is the oracle:
//
//   - `%r/[+-=]/` is a RANGE from '+' (U+002B) to '=' (U+003D), not the three
//     characters + - =. It therefore also covers , - . / 0-9 : ; <. Every one
//     of those that could reach it is claimed by an earlier rule (punctuation,
//     the number rules, the comment rules), so the widening is mostly inert --
//     but it is the gem's behaviour and is kept.
//   - `%r/(\-?(inf|nan))\b/` has no \b on the left, so `-inf` inside a longer
//     word still matches. Kept.
var protobufLexer = func() *RegexLexer {
	const (
		kw       = `\b(?:ctype|default|extensions|import|max|oneof|option|optional|packed|repeated|required|returns|rpc|to)\b`
		datatype = `\b(?:bool|bytes|double|fixed32|fixed64|float|int32|int64|sfixed32|sfixed64|sint32|sint64|string|uint32|uint64)\b`
	)

	b := newRegexLexer("protobuf", "Protobuf", "proto")
	b.filenames("*.proto")

	b.state("root").
		rule(`[\s]+`, Text).
		rule(`[,;{}\[\]()]`, Punctuation).
		rule(`\/(?:\\\n)?\/(?:$|(?:.|\n)*?[^\\]$)`, CommentSingle).
		rule(`\/(?:\\\n)?\*(?:.|\n)*?\*(?:\\\n)?\/`, CommentMultiline).
		rule(kw, Keyword).
		rule(datatype, KeywordType).
		rule(`true|false`, KeywordConstant).
		groupsRuleT(`(package)(\s+)`, []transition{push("package")}, KeywordNamespace, Text).
		groupsRuleT(`(message|extend)(\s+)`, []transition{push("message")}, KeywordDeclaration, Text).
		groupsRuleT(`(enum|group|service)(\s+)`, []transition{push("type")}, KeywordDeclaration, Text).
		rule(`".*?"`, LiteralString).
		rule(`'.*?'`, LiteralString).
		rule(`(?:\d+\.\d*|\.\d+|\d+)[eE][+-]?\d+[LlUu]*`, LiteralNumberFloat).
		rule(`(?:\d+\.\d*|\.\d+|\d+[fF])[fF]?`, LiteralNumberFloat).
		rule(`(?:\-?(?:inf|nan))\b`, LiteralNumberFloat).
		rule(`0x[0-9a-fA-F]+[LlUu]*`, LiteralNumberHex).
		rule(`0[0-7]+[LlUu]*`, LiteralNumberOct).
		rule(`\d+[LlUu]*`, LiteralNumberInteger).
		rule(`[+-=]`, Operator).
		groupsRule(`([a-zA-Z_][\w.]*)([ \t]*)(=)`, NameAttribute, Text, Operator).
		rule(`[a-zA-Z_][\w.]*`, Name)

	b.state("package").
		rule(`[a-zA-Z_]\w*`, NameNamespace, pop()).
		rule(``, nil, pop())

	b.state("message").
		rule(`[a-zA-Z_]\w*`, NameClass, pop()).
		rule(``, nil, pop())

	b.state("type").
		rule(`[a-zA-Z_]\w*`, Name, pop()).
		rule(``, nil, pop())

	return b.done()
}()
