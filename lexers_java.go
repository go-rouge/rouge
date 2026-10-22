package rouge

import (
	onigmo "github.com/go-ruby-regexp/regexp"
)

// --- Java ---
// A faithful transcription of Rouge::Lexers::Java (rouge 5.0.0,
// lib/rouge/lexers/java.rb).
//
// ⛔ INTERPOLATING A REGEXP IS NOT INTERPOLATING ITS SOURCE. The gem writes
// `digit = /[0-9]_+[0-9]|[0-9]/` and then `%r/#{digit}+/`, and Ruby splices a
// Regexp in as `(?-mix:...)` -- so the `+` applies to the whole alternation,
// not to its last branch. Transcribing `#{digit}+` as `[0-9]_+[0-9]|[0-9]+`
// would quietly change what matches. Every interpolated fragment below is
// therefore wrapped in `(?:...)` at its use site, which is what Ruby does.
const (
	javaID        = `[[:alpha:]_][[:word:]]*`
	javaConstName = `[[:upper:]][[:upper:][:digit:]_]*\b`
	javaClassName = `[[:upper:]][[:alnum:]]*\b`

	javaKeywords = `assert|break|case|catch|continue|default|do|else|finally|for|` +
		`if|goto|instanceof|new|return|switch|this|throw|try|while`

	javaDeclarations = `abstract|const|enum|extends|final|implements|native|private|protected|` +
		`public|static|strictfp|super|synchronized|throws|transient|volatile`

	javaTypes = `boolean|byte|char|double|float|int|long|short|var|void`

	// The gem's digit fragments, each used as `(?:...)` + quantifier below.
	javaDigit    = `[0-9]_+[0-9]|[0-9]`
	javaBinDigit = `[01]_+[01]|[01]`
	javaOctDigit = `[0-7]_+[0-7]|[0-7]`
	javaHexDigit = `[0-9a-f]_+[0-9a-f]|[0-9a-f]`
)

var javaLexer = func() *RegexLexer {
	b := newRegexLexer("java", "Java")
	b.filenames("*.java")

	b.state("root").
		rule(`[^\S\n]+`, Text).
		rule(`//.*?$`, CommentSingle).
		rule(`(?m)/\*.*?\*/`, CommentMultiline).
		// Keywords go before method names so "throw new XYZ" is not lexed as a
		// method signature (the gem says so in a comment here).
		rule(`(?:`+javaKeywords+`)\b`, Keyword).
		// The method-signature rule. The gem writes it with /x (whitespace and
		// comments stripped) and a block that DELEGATES THE RETURN ARGUMENTS TO
		// JAVA ITSELF -- `delegate Java, m[1]` -- which is a self-recursion, so
		// the modifiers and the type in "public static int foo(" keep their own
		// tokens instead of becoming one blob. recurse is this port's name for
		// re-lexing from the lexer's own root.
		// needAny "(": the pattern's last group is (\(), so no match can exist
		// unless an opening parenthesis lies at or after the scan position.
		// Same defect as C's signature rule (#17): without this the lazy `+?`
		// scans to end of input from EVERY position on an unterminated block
		// comment, which is O(n^2).
		cbNeedAny(`(`, `(\s*(?:[a-zA-Z_][a-zA-Z0-9_.\[\]<>]*\s+)+?)([a-zA-Z_][a-zA-Z0-9_]*)(\s*)(\()`,
			func(l *lexState, m *onigmo.MatchData) {
				l.recurse(m.Str(1))
				l.emit(NameFunction, m.Str(2))
				l.emit(Text, m.Str(3))
				l.emit(Operator, m.Str(4))
				l.pos = m.End(0)
			}).
		rule(`@`+javaID, NameDecorator).
		rule(`(?:`+javaDeclarations+`)\b`, KeywordDeclaration).
		rule(`(?:`+javaTypes+`)\b`, KeywordType).
		rule(`(?:true|false|null)\b`, KeywordConstant).
		rule(`(?:class|interface|record)\b`, KeywordDeclaration, push("class")).
		rule(`(?:import|package)\b`, KeywordNamespace, push("import")).
		// A text block: """ then a newline, up to an unescaped """.
		rule(`(?m)"""\s*\n.*?(?<!\\)"""`, LiteralStringHeredoc).
		rule(`"(?:\\\\|\\"|[^"])*"`, LiteralString).
		rule(`'(?:\\.|[^\\]|\\u[0-9a-f]{4})'`, LiteralStringChar).
		groupsRule(`(\.)(`+javaID+`)`, Operator, NameAttribute).
		rule(javaID+`:`, NameLabel).
		rule(javaConstName, NameConstant).
		rule(javaClassName, NameClass).
		rule(`\$?`+javaID, Name).
		rule(`[~^*!%&\[\](){}<>\|+=:;,.\/?-]`, Operator).
		rule(`(?:`+javaDigit+`)+\.(?:`+javaDigit+`)+(?:[eE](?:`+javaDigit+`)+)?[fd]?`, LiteralNumberFloat).
		rule(`(?i)0b(?:`+javaBinDigit+`)+`, LiteralNumberBin).
		rule(`(?i)0x(?:`+javaHexDigit+`)+`, LiteralNumberHex).
		rule(`0(?:`+javaOctDigit+`)+`, LiteralNumberOct).
		rule(`(?:`+javaDigit+`)+L?`, LiteralNumberInteger).
		rule(`\n`, Text)

	b.state("class").
		rule(`(?m)\s+`, Text).
		rule(javaID, NameClass, pop())

	b.state("import").
		rule(`(?m)\s+`, Text).
		rule(`(?i)[a-z0-9_.]+\*?`, NameNamespace, pop())

	return b.done()
}()
