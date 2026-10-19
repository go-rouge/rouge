package rouge

import (
	onigmo "github.com/go-ruby-regexp/regexp"
)

// --- HCL, and Terraform on top of it ---
// A faithful transcription of Rouge::Lexers::Hcl and Rouge::Lexers::Terraform
// (rouge 5.0.0, lib/rouge/lexers/hcl.rb and terraform.rb), where the gem
// writes `class Terraform < Hcl`.
//
// Terraform does not prepend rules the way C++ does; it REPLACES :dq, :sq and
// :heredoc and adds four states of its own. Defining a state again here
// replaces it, which is the same thing, so both lexers share one builder and
// Terraform's hook runs last.

// hclID is the gem's `id`, which carries /i.
const hclID = `(?i)[$a-z_\-][a-z0-9_\-]*`

type hclOpts struct {
	tag      string
	title    string
	aliases  []string
	filename []string

	keywords     map[string]bool
	declarations map[string]bool
	reserved     map[string]bool
	constants    map[string]bool
	builtins     map[string]bool

	// extraStates redefines or adds states, as a subclass would.
	extraStates func(b *lexerBuilder)
}

func buildHCL(o hclOpts) *RegexLexer {
	b := newRegexLexer(o.tag, o.title, o.aliases...)
	b.filenames(o.filename...)

	b.state("multiline_comment").
		rule(`[*]/`, CommentMultiline, pop()).
		rule(`[^*/]+`, CommentMultiline).
		rule(`[*/]`, CommentMultiline)

	b.state("comments_and_whitespace").
		rule(`\s+`, Text).
		rule(`//.*?$`, CommentSingle).
		rule(`#.*?$`, CommentSingle).
		rule(`/[*]`, CommentMultiline, push("multiline_comment"))

	b.state("primitives").
		rule(`[0-9][0-9]*\.[0-9]+(?:[eE][0-9]+)?[fd]?(?:[kKmMgG]b?)?`, LiteralNumberFloat).
		rule(`[0-9]+(?:[kKmMgG]b?)?`, LiteralNumberInteger).
		rule(`[-+*\/!%&<>|=:?]`, Operator).
		rule(`"`, LiteralStringDouble, push("dq")).
		rule(`'`, LiteralStringSingle, push("sq"))
		// ⛔ THE GEM'S HERE-DOCUMENT RULE IS NOT TRANSCRIBED, BECAUSE THE GEM
		// CANNOT REACH IT. hcl.rb puts
		//
		//	rule %r/(<<-?)(\s*)(\'?)(\\?)(\w+)(\3)/   -> Str::Heredoc, :heredoc
		//
		// AFTER the operator rule three lines above, and that operator class
		// contains `<`. Rouge takes the first rule that matches, so `<<EOF`
		// is consumed as two Operators and a Name, and :heredoc and
		// :heredoc_nl are never entered.
		//
		// Measured, not assumed: testdata/hcl.hcl carries two here-documents
		// (<<EOF and <<-'END'), and the gem's own golden for it contains ZERO
		// spans of class "sh" (Str::Heredoc) and zero of class "no"
		// (Name::Constant). Transcribing the rule would ship three states that
		// no input can enter.
		//
		// If upstream ever moves that rule above the operator, this is where
		// it goes back, together with :heredoc and :heredoc_nl. Recorded in
		// the README's documented simplifications.

	// nameRule is the gem's id-dispatch block. In :root a name that is not
	// reserved, constant or builtin opens a composite (a block header such as
	// `resource "aws_instance" "x" {`), so the lexer knows a brace may follow.
	nameRule := func(l *lexState, m *onigmo.MatchData, pushComposite bool) {
		name := m.Str(0)
		switch {
		case o.keywords[name]:
			l.emit(Keyword, name)
			l.pos = m.End(0)
			if pushComposite {
				l.push("composite")
			}
			return
		case o.declarations[name]:
			l.emit(KeywordDeclaration, name)
			l.pos = m.End(0)
			if pushComposite {
				l.push("composite")
			}
			return
		case o.reserved[name]:
			l.emit(KeywordReserved, name)
		case o.constants[name]:
			l.emit(KeywordConstant, name)
		case o.builtins[name]:
			l.emit(NameBuiltin, name)
		default:
			l.emit(NameOther, name)
			l.pos = m.End(0)
			if pushComposite {
				l.push("composite")
			}
			return
		}
		l.pos = m.End(0)
	}

	b.state("root").
		mixin("comments_and_whitespace").
		mixin("primitives").
		cb(`\{`, func(l *lexState, m *onigmo.MatchData) {
			l.emit(Punctuation, m.Str(0))
			l.pos = m.End(0)
			l.push("hash")
		}).
		cb(`\[`, func(l *lexState, m *onigmo.MatchData) {
			l.emit(Punctuation, m.Str(0))
			l.pos = m.End(0)
			l.push("array")
		}).
		cb(hclID, func(l *lexState, m *onigmo.MatchData) { nameRule(l, m, true) })

	b.state("composite").
		mixin("comments_and_whitespace").
		cb(`[{]`, func(l *lexState, m *onigmo.MatchData) {
			l.emit(Punctuation, m.Str(0))
			l.pos = m.End(0)
			l.pop(1)
			l.push("hash")
		}).
		cb(`[\[]`, func(l *lexState, m *onigmo.MatchData) {
			l.emit(Punctuation, m.Str(0))
			l.pos = m.End(0)
			l.pop(1)
			l.push("array")
		}).
		mixin("root").
		rule(``, Text, pop())

	b.state("hash").
		mixin("comments_and_whitespace").
		rule(`[.,()\\\/*]`, Punctuation).
		rule(`\=`, Punctuation).
		rule(`\}`, Punctuation, pop()).
		mixin("root")

	b.state("array").
		mixin("comments_and_whitespace").
		rule(`[.,()\\\/*]`, Punctuation).
		rule(`\]`, Punctuation, pop()).
		mixin("root")

	b.state("dq").
		rule(`[^\\"]+`, LiteralStringDouble).
		rule(`\\"`, LiteralStringEscape).
		rule(`"`, LiteralStringDouble, pop())

	b.state("sq").
		rule(`[^\\']+`, LiteralStringSingle).
		rule(`\\'`, LiteralStringEscape).
		rule(`'`, LiteralStringSingle, pop())

	// :heredoc and :heredoc_nl are deliberately absent; see the note on the
	// here-document opener in :primitives.

	if o.extraStates != nil {
		o.extraStates(b)
	}
	return b.done()
}

var hclLexer = buildHCL(hclOpts{
	tag:          "hcl",
	title:        "Hashicorp Configuration Language",
	filename:     []string{"*.hcl", "*.nomad"},
	keywords:     map[string]bool{},
	declarations: map[string]bool{},
	reserved:     map[string]bool{},
	constants:    stringSet(`true false null`),
	builtins:     map[string]bool{},
})

// --- Terraform ---

var terraformKeywords = stringSet(`terraform module provider variable resource data provisioner output`)
var terraformDeclarations = stringSet(`var local`)
var terraformConstants = stringSet(`true false null`)

var terraformLexer = buildHCL(hclOpts{
	tag:          "terraform",
	title:        "Terraform",
	aliases:      []string{"tf"},
	filename:     []string{"*.tf", "*.tfvars", "*.tofu"},
	keywords:     terraformKeywords,
	declarations: terraformDeclarations,
	reserved:     map[string]bool{},
	constants:    terraformConstants,
	builtins:     map[string]bool{},

	extraStates: func(b *lexerBuilder) {
		// Interpolation is what Terraform adds to HCL: ${...} inside a string,
		// a heredoc, or another interpolation.
		b.state("strings").
			rule(`\\.`, LiteralStringEscape).
			rule(`(?:\$[\$]+|%[%]+)(?:\{)`, LiteralString).
			cb(`\$\{`, func(l *lexState, m *onigmo.MatchData) {
				l.emit(Punctuation, m.Str(0))
				l.pos = m.End(0)
				l.push("interpolation")
			})

		// These three replace HCL's versions.
		b.state("dq").
			rule(`[^\\"\$]+`, LiteralStringDouble).
			mixin("strings").
			rule(`"`, LiteralStringDouble, pop())

		b.state("sq").
			rule(`[^\\'\$]+`, LiteralStringSingle).
			mixin("strings").
			rule(`'`, LiteralStringSingle, pop())

		// Terraform's :heredoc override is absent for the same reason HCL's
		// is: the state cannot be entered.

		b.state("interpolation").
			cb(`\}`, func(l *lexState, m *onigmo.MatchData) {
				l.emit(Punctuation, m.Str(0))
				l.pos = m.End(0)
				l.pop(1)
			}).
			mixin("expression")

		b.state("regexps").
			cb(`"\/`, func(l *lexState, m *onigmo.MatchData) {
				l.emit(LiteralStringDelimiter, m.Str(0))
				l.pos = m.End(0)
				l.goTo("regexp_inner")
			})

		b.state("regexp_inner").
			rule(`[^"\/\\]+`, LiteralStringRegex).
			rule(`\\.`, LiteralStringRegex).
			rule(`\/"`, LiteralStringDelimiter, pop()).
			rule(`["\/]`, LiteralStringRegex)

		// An expression inside ${...}. Its name rule does NOT open a
		// composite: there is no block header inside an interpolation.
		b.state("expression").
			mixin("regexps").
			mixin("primitives").
			rule(`\s+`, Text).
			rule(`\+\+|--|~|&&|\|\||\\(?=\n)|<<|>>>?|==|!=`, Operator).
			rule(`[-<>+*%&|\^/!=?:]=?`, Operator).
			rule(`[(\[,]`, Punctuation).
			rule(`[)\].]`, Punctuation).
			cb(hclID, func(l *lexState, m *onigmo.MatchData) {
				name := m.Str(0)
				switch {
				case terraformKeywords[name]:
					l.emit(Keyword, name)
				case terraformDeclarations[name]:
					l.emit(KeywordDeclaration, name)
				case terraformConstants[name]:
					l.emit(KeywordConstant, name)
				default:
					l.emit(NameOther, name)
				}
				l.pos = m.End(0)
			})
	},
})
