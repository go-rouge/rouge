package rouge

import (
	onigmo "github.com/go-ruby-regexp/regexp"
)

// --- C++ ---
// A faithful transcription of Rouge::Lexers::Cpp (rouge 5.0.0,
// lib/rouge/lexers/cpp.rb), which the gem defines as `class Cpp < C`.
//
// Everything shared lives in buildCFamily (lexers_c.go). What is here is
// exactly what the gem adds: three larger keyword sets, two rules prepended to
// :root and :statements, two new states, and a :case that does not end on the
// scope operator.

var cppKeywords = unionSet(cKeywords, `
and and_eq asm bitand bitor catch compl concept consteval
constinit const_cast co_await co_return co_yield decltype
delete dynamic_cast explicit export final friend import
module mutable namespace new noexcept not not_eq operator or
or_eq override private protected public reinterpret_cast
requires size_of static_cast this throw throws try typeid
typename using virtual xor xor_eq`)

var cppKeywordsType = unionSet(cKeywordsType, `char8_t`)

var cppReserved = unionSet(cReserved, `
__virtual_inheritance __uuidof __super __single_inheritance
__multiple_inheritance __interface __event`)

// cppDQ is the gem's `dq`: a digit run with optional inner quotes, from
// n3781. Ruby interpolates it as `(?-mix:...)`, so it is wrapped at each use.
const cppDQ = `\d(?:'?\d)*`

var cppLexer = buildCFamily(cFamilyOpts{
	tag:     "cpp",
	title:   "C++",
	aliases: []string{"c++"},
	filename: []string{
		"*.cpp", "*.hpp", "*.c++", "*.h++", "*.cc", "*.hh",
		"*.cxx", "*.hxx", "*.pde", "*.ino", "*.tpp", "*.h",
	},
	keywords:     cppKeywords,
	keywordsType: cppKeywordsType,
	reserved:     cppReserved,
	builtins:     map[string]bool{},

	// C++ ends a case label on a colon that is NOT the scope operator, so
	// `case Foo::Bar:` ends at the last colon rather than inside the name.
	caseColon: `:(?!:)`,

	prependRoot: func(b *lexerBuilder) {
		// Offload C++ extensions, http://offload.codeplay.com/
		b.rule(`(?:__offload|__blockingoffload|__outer)\b`, KeywordPseudo)
	},

	prependStatements: func(b *lexerBuilder) {
		b.rule(`(?:class|struct)\b`, Keyword, push("classname")).
			rule(`template\b`, Keyword, push("template")).
			rule(`(?:`+cppDQ+`)(?:\.(?:`+cppDQ+`))?(?:y|d|h|(?:min)|s|(?:ms)|(?:us)|(?:ns)|i|(?:if)|(?:il))\b`, LiteralNumberOther).
			rule(`(?i)(?:(?:`+cppDQ+`)[.](?:`+cppDQ+`)?|[.](?:`+cppDQ+`))(?:[ep][+-]?(?:`+cppDQ+`))?[luf]*`, LiteralNumberFloat).
			rule(`(?i)(?:`+cppDQ+`)[ep][+-]?(?:`+cppDQ+`)[luf]*`, LiteralNumberFloat).
			rule(`(?i)0x\h(?:'?\h)*(?:[ep][+-]?(?:`+cppDQ+`))?[lu]*`, LiteralNumberHex).
			rule(`(?i)0x\h(?:'?\h)*[.]\h+(?:[ep][+-]?(?:`+cppDQ+`))[luf]*`, LiteralNumberHex).
			rule(`0b[01]+(?:'[01]+)*`, LiteralNumberBin).
			rule(`(?i)0[0-7](?:'?[0-7])*[lu]*`, LiteralNumberOct).
			rule(`(?i)(?:`+cppDQ+`)[lu]*`, LiteralNumberInteger).
			rule(`\bnullptr\b`, NameBuiltin).
			// A raw string literal: the delimiter captured in group 1 is matched
			// again by the backreference, so R"xy(...)xy" ends only on its own
			// delimiter.
			rule(`(?m)(?:u8|u|U|L)?R"([a-zA-Z0-9_{}\[\]#<>%:;.?*\+\-\/\^&|~!=,"']{0,16})\(.*?\)\1"`, LiteralString).
			rule(`(?:::|<=>)`, Operator).
			rule(`[{]`, Punctuation).
			// A closing brace ends a function body only when one is open: the
			// gem checks in_state?(:function) before popping.
			cb(`\}`, func(l *lexState, m *onigmo.MatchData) {
				l.emit(Punctuation, m.Str(0))
				l.pos = m.End(0)
				if l.inState("function") {
					l.pop(1)
				}
			})
	},

	extraStates: func(b *lexerBuilder) {
		b.state("classname").
			rule(cID, NameClass, pop()).
			// template specification
			mixin("whitespace").
			rule(`[.]{3}`, Operator).
			rule(`,`, Punctuation, pop()).
			rule(``, nil, pop())

		b.state("template").
			rule(`[>;]`, Punctuation, pop()).
			rule(`typename\b`, Keyword, push("classname")).
			mixin("statements")
	},
})
