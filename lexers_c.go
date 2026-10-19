package rouge

import (
	onigmo "github.com/go-ruby-regexp/regexp"
)

// --- C, and the family built on it ---
// A faithful transcription of Rouge::Lexers::C (rouge 5.0.0,
// lib/rouge/lexers/c.rb).
//
// The gem expresses C++ as `class Cpp < C`, overriding the keyword sets and
// using `prepend :root` / `prepend :statements` to put extra rules IN FRONT of
// the inherited ones. Go has no such inheritance, so the shared states are
// built here by buildCFamily, which takes the keyword sets and the hooks a
// subclass would use. C passes no hooks; C++ (lexers_cpp.go) passes all of
// them. Keeping one builder is what makes "C++ is C plus these rules" a fact
// about the code rather than a claim in a comment: a rule added to C is a rule
// C++ gets.

const (
	cID = `[a-zA-Z_][a-zA-Z0-9_]*`
	// cWS is the gem's `ws`: optional comment or whitespace. It is interpolated
	// into the function-signature rule, where Ruby splices a Regexp as
	// `(?-mix:...)`, so it is wrapped at its use site.
	cWS = `(?:\s|//.*?\n|/[*].*?[*]/)+`
)

var cKeywords = stringSet(`
alignas alignof auto break case const constexpr continue
default do else enum extern for goto if register return
sizeof static static_assert struct switch typedef typeof
typeof_unqual union volatile while
_Alignas _Alignof _Atomic _Generic _Imaginary
_Noreturn _Static_assert _Thread_local`)

var cKeywordsType = stringSet(`
int long float short double char unsigned signed void
jmp_buf FILE DIR div_t ldiv_t mbstate_t sig_atomic_t fpos_t
clock_t time_t va_list size_t ssize_t off_t wchar_t ptrdiff_t
wctrans_t wint_t wctype_t
_Bool _Complex int8_t int16_t int32_t int64_t
uint8_t uint16_t uint32_t uint64_t int_least8_t
int_least16_t int_least32_t int_least64_t
uint_least8_t uint_least16_t uint_least32_t
uint_least64_t int_fast8_t int_fast16_t int_fast32_t
int_fast64_t uint_fast8_t uint_fast16_t uint_fast32_t
uint_fast64_t intptr_t uintptr_t intmax_t
uintmax_t
char16_t char32_t
_BitInt _Decimal128 _Decimal32 _Decimal64 bool nullptr_t`)

var cReserved = stringSet(`
__asm __int8 __based __except __int16 __stdcall __cdecl
__fastcall __int32 __declspec __finally __int61 __try __leave
inline _inline __inline naked _naked __naked restrict _restrict
__restrict thread _thread __thread thread_local
typename _typename __typename`)

// unionSet returns a new set holding every word of base plus the words of
// extra, mirroring the gem's `super + Set.new(%w(...))`. It copies rather than
// mutating, so C's sets cannot be changed by defining C++.
func unionSet(base map[string]bool, extra string) map[string]bool {
	out := make(map[string]bool, len(base))
	for k := range base {
		out[k] = true
	}
	for k := range stringSet(extra) {
		out[k] = true
	}
	return out
}

// cFamilyOpts carries what a C-family lexer differs in.
type cFamilyOpts struct {
	tag      string
	title    string
	aliases  []string
	filename []string

	keywords     map[string]bool
	keywordsType map[string]bool
	reserved     map[string]bool
	builtins     map[string]bool

	// prependRoot and prependStatements add rules at the FRONT of those
	// states, which is what the gem's `prepend` does.
	prependRoot       func(b *lexerBuilder)
	prependStatements func(b *lexerBuilder)
	// caseColon is the pattern ending a `case` label. C uses `:`; C++ uses
	// `:(?!:)` so that `case Foo::Bar:` is not ended by the scope operator.
	caseColon string
	// extraStates defines states only the subclass has.
	extraStates func(b *lexerBuilder)
}

func buildCFamily(o cFamilyOpts) *RegexLexer {
	b := newRegexLexer(o.tag, o.title, o.aliases...)
	b.filenames(o.filename...)
	// The gem's `start { push :bol }`: a lex begins at the beginning of a line,
	// where a label or a preprocessor directive may appear.
	b.start("bol")

	b.state("inline_whitespace").
		rule(`[ \t\r]+`, Text).
		rule(`\\\n`, Text). // line continuation
		rule(`(?m)/(?:\\\n)?[*].*?[*](?:\\\n)?/`, CommentMultiline)

	b.state("expr_bol").
		mixin("inline_whitespace").
		rule(`#if\s0`, Comment, push("if_0")).
		rule(`#`, CommentPreproc, push("macro")).
		// An empty pattern that pops: the gem's `rule(//) { pop! }`. It emits
		// nothing, so it is spelled with a nil token here.
		rule(``, nil, pop())

	// :bol is :expr_bol plus labels, because a label can only begin a statement.
	b.state("bol").
		rule(cID+`:(?!:)`, NameLabel).
		mixin("expr_bol")

	b.state("whitespace").
		rule(`(?m)\n+`, Text, push("bol")).
		rule(`//(?:\\.|.)*?$`, CommentSingle, push("bol")).
		mixin("inline_whitespace")

	b.state("expr_whitespace").
		rule(`(?m)\n+`, Text, push("expr_bol")).
		mixin("whitespace")

	b.state("statements")
	if o.prependStatements != nil {
		o.prependStatements(b)
	}
	b.mixin("whitespace").
		rule(`(?:u8|u|U|L)?"`, LiteralString, push("string")).
		rule(`(?i)(?:u8|u|U|L)?'(?:\\.|\\[0-7]{1,3}|\\x[a-f0-9]{1,2}|[^\\'\n])'`, LiteralStringChar).
		rule(`(?i)(?:\d+[.]\d*|[.]?\d+)e[+-]?\d+[lu]*`, LiteralNumberFloat).
		rule(`(?i)\d+e[+-]?\d+[lu]*`, LiteralNumberFloat).
		rule(`(?i)0x[0-9a-f]+[lu]*`, LiteralNumberHex).
		rule(`(?i)0[0-7]+[lu]*`, LiteralNumberOct).
		rule(`(?i)\d+[lu]*`, LiteralNumberInteger).
		rule(`\*/`, Error).
		rule(`[~!%^&*+=\|?:<>/-]`, Operator).
		rule(`[()\[\],.;]`, Punctuation).
		rule(`\bcase\b`, Keyword, push("case")).
		rule(`(?:true|false|NULL|nullptr)\b`, NameBuiltin).
		cb(cID, func(l *lexState, m *onigmo.MatchData) {
			name := m.Str(0)
			switch {
			case o.keywords[name]:
				l.emit(Keyword, name)
			case o.keywordsType[name]:
				l.emit(KeywordType, name)
			case o.reserved[name]:
				l.emit(KeywordReserved, name)
			case o.builtins[name]:
				l.emit(NameBuiltin, name)
			default:
				l.emit(Name, name)
			}
			l.pos = m.End(0)
		})

	caseColon := o.caseColon
	if caseColon == "" {
		caseColon = `:`
	}
	b.state("case").
		rule(caseColon, Punctuation, pop()).
		mixin("statements")

	b.state("root")
	if o.prependRoot != nil {
		o.prependRoot(b)
	}
	b.mixin("expr_whitespace").
		// The function-signature rule. The gem writes it with /x (whitespace
		// and comments in the pattern are stripped) and /m, and its block
		// RECURSES on three of the five groups, so the return type, the
		// argument list and any comment before the brace keep their own tokens
		// instead of becoming one blob.
		cb(`(?m)([\w*\s]+?[\s*])(`+cID+`)(\s*\([^;]*?\))((?:`+cWS+`)?)(\{|;)`,
			func(l *lexState, m *onigmo.MatchData) {
				l.recurse(m.Str(1))
				l.emit(NameFunction, m.Str(2))
				l.recurse(m.Str(3))
				l.recurse(m.Str(4))
				l.emit(Punctuation, m.Str(5))
				l.pos = m.End(0)
				if m.Str(5) == "{" {
					l.push("function")
				}
			}).
		rule(`\{`, Punctuation, push("function")).
		mixin("statements")

	b.state("function").
		mixin("whitespace").
		mixin("statements").
		rule(`;`, Punctuation).
		rule(`\{`, Punctuation, push("function")).
		rule(`\}`, Punctuation, pop())

	b.state("string").
		rule(`"`, LiteralString, pop()).
		rule(`\\(?:[\\abfnrtv"']|x[a-fA-F0-9]{2,4}|[0-7]{1,3})`, LiteralStringEscape).
		rule(`[^\\"\n]+`, LiteralString).
		rule(`\\\n`, LiteralString).
		rule(`\\`, LiteralString) // stray backslash

	b.state("include").
		groupsRule(`(include)(\s*)(<[^>]+>)([^\n]*)`, CommentPreproc, Text, CommentPreprocFile, CommentSingle).
		groupsRule(`(include)(\s*)("[^"]+")([^\n]*)`, CommentPreproc, Text, CommentPreprocFile, CommentSingle)

	b.state("macro").
		mixin("include").
		rule(`[^/\n\\]+`, CommentPreproc).
		rule(`(?m)\\.`, CommentPreproc).
		mixin("inline_whitespace").
		rule(`/`, CommentPreproc).
		// NB: this pop goes back to :bol.
		rule(`\n`, CommentPreproc, pop())

	b.state("if_0").
		// NB: no \b here, so #ifdef and #ifndef are covered too.
		rule(`^\s*#if`, Comment, push("if_0")).
		rule(`^\s*#\s*el(?:se|if)`, Comment, pop()).
		rule(`(?m)^\s*#\s*endif\b.*?(?<!\\)\n`, Comment, pop()).
		rule(`(?m).*?\n`, Comment)

	if o.extraStates != nil {
		o.extraStates(b)
	}
	return b.done()
}

var cLexer = buildCFamily(cFamilyOpts{
	tag:          "c",
	title:        "C",
	filename:     []string{"*.c", "*.h", "*.idc"},
	keywords:     cKeywords,
	keywordsType: cKeywordsType,
	reserved:     cReserved,
	builtins:     map[string]bool{},
})
