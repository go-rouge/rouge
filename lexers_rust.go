package rouge

import (
	"strings"

	onigmo "github.com/go-ruby-regexp/regexp"
)

// --- Rust ---
// A faithful transcription of Rouge::Lexers::Rust (rouge 5.0.0,
// lib/rouge/lexers/rust.rb).

// ⛔ XID_Start / XID_Continue ARE NOT IN THIS ENGINE. The gem writes
// `id = /[\p{XID_Start}_]\p{XID_Continue}*/`, and onigmo here answers
// "invalid character property name {XID_Start}". The nearest properties it
// does have are used instead:
//
//	XID_Start     -> [\p{L}\p{Nl}_]
//	XID_Continue  -> \p{Word}
//
// \p{Mn}, \p{Mc} and \p{Pc} are not available either, so XID_Continue uses
// \p{Word} -- which Ruby defines as \p{L}\p{M}\p{N}\p{Pc}, a SUPERSET of
// XID_Continue, and the same set for every identifier a Rust program can
// actually carry. \p{Word} is what the Ruby, PHP and JavaScript lexers in this
// port already use for the same purpose. The corpus carries non-ASCII
// identifiers so the substitution is measured against the gem rather than
// assumed; see the README's documented simplifications.
const (
	rustIDStart = `[\p{L}\p{Nl}_]`
	rustIDCont  = `\p{Word}`
	rustID      = rustIDStart + rustIDCont + `*`
	rustHex     = `(?i:[0-9a-f])`
	// rustEscapes is the gem's `escapes`, written with /x there (whitespace in
	// the pattern is not part of it).
	rustEscapes = `\\(?:[nrt'"\\0]|x` + rustHex + `{2}|u\{(?:` + rustHex + `_*){1,6}\})`
	// rustSize, rustDot, rustExp and rustFlt are the gem's local fragments.
	// Ruby splices a Regexp as `(?-mix:...)`, so each is wrapped where used.
	rustSize = `8|16|32|64|128|size`
	rustDot  = `[.][0-9][0-9_]*`
	rustExp  = `[eE][-+]?[0-9_]+`
	rustFlt  = `f32|f64`
)

var rustKeywords = `as|async|await|break|const|continue|crate|dyn|else|enum|extern|false|` +
	`fn|for|if|impl|in|let|log|loop|match|mod|move|mut|pub|ref|return|self|` +
	`Self|static|struct|super|trait|true|type|unsafe|use|where|while|` +
	`abstract|become|box|do|final|macro|` +
	`override|priv|typeof|unsized|virtual|` +
	`yield|try|` +
	`union`

var rustBuiltins = stringSet(`
Add BitAnd BitOr BitXor bool c_char c_double c_float char
c_int clock_t c_long c_longlong Copy c_schar c_short
c_uchar c_uint c_ulong c_ulonglong c_ushort c_void dev_t DIR
dirent Div Eq Err f32 f64 FILE float fpos_t
i16 i32 i64 i8 isize Index ino_t int intptr_t mode_t Mul
Neg None off_t Ok Option Ord Owned pid_t ptrdiff_t
Send Shl Shr size_t Some ssize_t str Sub time_t
u16 u32 u64 u8 usize uint uintptr_t
Box Vec String Rc Arc
u128 i128 Result Sync Pin Unpin Sized Drop drop Fn FnMut FnOnce
Clone PartialEq PartialOrd AsMut AsRef From Into Default
DoubleEndedIterator ExactSizeIterator Extend IntoIterator Iterator
FromIterator ToOwned ToString TryFrom TryInto`)

// rustDelimFor maps an opening macro delimiter to the closing one whose
// counter it increments (the gem's delim_map).
var rustDelimFor = map[string]string{"[": "]", "(": ")", "{": "}"}

var rustLexer = func() *RegexLexer {
	b := newRegexLexer("rust", "Rust",
		"rs",
		// So that directives from rust-skeptic do not prevent highlighting.
		"rust,no_run", "rs,no_run",
		"rust,ignore", "rs,ignore",
		"rust,should_panic", "rs,should_panic")
	b.filenames("*.rs")
	b.detectWith(func(text string) bool {
		if !strings.HasPrefix(text, "#!") {
			return false
		}
		line := text
		if nl := strings.IndexByte(text, '\n'); nl >= 0 {
			line = text[:nl]
		}
		return strings.Contains(line, "rustc")
	})
	// The gem's start block seeds @macro_delims and pushes :bol; the counters
	// are created on first use instead (see lexState.macroDelims).
	b.start("bol")

	// rustdoc lets code in comments begin with `# `, which is evaluated but not
	// shown. Such a line is marked Comment::Special rather than lexed as code.
	b.state("bol").
		mixin("whitespace").
		rule(`#\s[^\n]*`, CommentSpecial).
		rule(``, nil, pop())

	b.state("attribute").
		mixin("whitespace").
		mixin("has_literals").
		rule(`[(,)=:]`, NameDecorator).
		rule(`\]`, NameDecorator, pop()).
		rule(rustID, NameDecorator)

	b.state("whitespace").
		rule(`\s+`, Text).
		mixin("comments")

	b.state("comments").
		// Only three slashes are doc comments; //// and beyond go back to being
		// ordinary comments, so this is matched first.
		rule(`////+[^\n]*`, CommentSingle).
		// Doc line comments, inner (//!) or outer (///).
		rule(`//[/!][^\n]*`, CommentDoc).
		rule(`//[^\n]*`, CommentSingle).
		// /**/ and /***/ close themselves, so they never enter a nested state.
		rule(`/\*\*\*?/`, CommentMultiline).
		rule(`/\*\*\*+`, CommentMultiline, push("nested_plain_block")).
		rule(`/[*][*!]`, CommentDoc, push("nested_doc_block")).
		rule(`/[*]`, CommentMultiline, push("nested_plain_block"))

	// Block comments nest, and the OUTERMOST kind wins: everything inside a doc
	// block stays documentation even if it contains a plain block comment, and
	// everything inside a plain block stays plain even if it contains a doc
	// block. Two states say that without tracking it in a variable.
	b.state("nested_plain_block").
		rule(`\*/`, CommentMultiline, pop()).
		rule(`/\*`, CommentMultiline, push("nested_plain_block")).
		// At most one [*/] at a time, but anything else can be eaten in bulk.
		rule(`[^*/]+|[*/]`, CommentMultiline)

	b.state("nested_doc_block").
		rule(`\*/`, CommentDoc, pop()).
		rule(`/\*`, CommentDoc, push("nested_doc_block")).
		rule(`[^*/]+|[*/]`, CommentDoc)

	b.state("root").
		rule(`\n`, Text, push("bol")).
		mixin("whitespace").
		rule(`#!?\[`, NameDecorator, push("attribute")).
		rule(`\b(?:`+rustKeywords+`)\b`, Keyword).
		mixin("has_literals").
		rule(`[=-]>`, Keyword).
		rule(`<->`, Keyword).
		rule(`[()\[\]{}|,:;]`, Punctuation).
		rule(`[*\/!@~&+%^<>=\?-]|\.{2,3}`, Operator).
		rule(`(?m)(?:[.]\s*)?`+rustID+`(?=\s*[(])`, NameFunction).
		rule(`[.]\s*await\b`, Keyword).
		rule(`[.]\s*`+rustID, NameProperty).
		rule(`[.]\s*\d+`, NameAttribute).
		groupsRule(`(?m)(`+rustID+`)(::)`, NameNamespace, Punctuation).
		// macros
		rule(`\bmacro_rules!`, NameDecorator, push("macro_rules")).
		rule(rustID+`!`, NameDecorator, push("macro")).
		rule(`'static\b`, Keyword).
		rule(`'`+rustID, NameVariable).
		cb(rustID, func(l *lexState, m *onigmo.MatchData) {
			name := m.Str(0)
			if rustBuiltins[name] {
				l.emit(NameBuiltin, name)
			} else {
				l.emit(Name, name)
			}
			l.pos = m.End(0)
		})

	b.state("macro").
		mixin("has_literals").
		cb(`[\[{(]`, func(l *lexState, m *onigmo.MatchData) {
			l.macroDelim()[rustDelimFor[m.Str(0)]]++
			l.emit(Punctuation, m.Str(0))
			l.pos = m.End(0)
		}).
		cb(`[\]})]`, func(l *lexState, m *onigmo.MatchData) {
			l.macroDelim()[m.Str(0)]--
			l.emit(Punctuation, m.Str(0))
			l.pos = m.End(0)
			if l.macroClosed() {
				l.pop(1)
			}
		}).
		// Same as the rule in root, but it does not push another macro state.
		rule(rustID+`!`, NameDecorator).
		mixin("root").
		// There are no syntax errors inside a macro.
		rule(`.`, Text)

	b.state("macro_rules").
		rule(`[$]`+rustID+`(?::`+rustID+`)?`, NameVariable).
		rule(`[$]`, NameVariable).
		mixin("macro")

	b.state("has_literals").
		// constants
		rule(`\b(?:true|false)\b`, KeywordConstant).
		// characters and bytes
		rule(`b?'(?:`+rustEscapes+`|[^\\])'`, LiteralStringChar).
		rule(`b?"`, LiteralString, push("string")).
		// A raw string: the hashes captured in group 1 must appear again to
		// close it, so r#"..."# ends only on its own delimiter.
		rule(`(?m)b?r(#*)".*?"\1`, LiteralString).
		// numbers
		rule(`[0-9][0-9_]*(?:`+
			`(?:`+rustDot+`)(?:`+rustExp+`)?(?:`+rustFlt+`)?`+
			`|(?:`+rustDot+`)?(?:`+rustExp+`)(?:`+rustFlt+`)?`+
			`|(?:`+rustDot+`)?(?:`+rustExp+`)?(?:`+rustFlt+`)`+
			`|[.](?![._\p{L}\p{Nl}])`+
			`)`, LiteralNumberFloat).
		rule(`(?:0b[10_]+|0x[0-9a-fA-F_]+|0o[0-7_]+|[0-9][0-9_]*)`+
			`(?:u(?:`+rustSize+`)?|i(?:`+rustSize+`))?`, LiteralNumberInteger)

	b.state("string").
		rule(`"`, LiteralString, pop()).
		rule(rustEscapes, LiteralStringEscape).
		rule(`\\\n[ \t\r\n]*`, LiteralStringEscape).
		rule(`(?m)[^"\\]+`, LiteralString)

	return b.done()
}()
