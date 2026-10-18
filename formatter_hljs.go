package rouge

import "strings"

// HLJSFormatter renders the token stream with highlight.js class names
// (<span class="hljs-keyword">), so that stylesheets written for
// highlight.js — Marp's themes, for one — colour Rouge's output. This is not
// a Rouge formatter; the gem has no equivalent.
//
// Tokens map to highlight.js scopes as its scope reference defines them
// (https://highlightjs.readthedocs.io/en/latest/css-classes-reference.html).
// A sub-scope becomes extra classes with one trailing underscore per level of
// depth, as highlight.js writes them: title.function is
// class="hljs-title function_". A token with no counterpart (plain text,
// names without a role, errors) is written unwrapped.
type HLJSFormatter struct{}

// Tag returns "hljs".
func (HLJSFormatter) Tag() string { return "hljs" }

// Format renders the stream.
func (HLJSFormatter) Format(tokens []TokenValue) string {
	var b strings.Builder
	for _, tv := range tokens {
		if tv.Token == Escape {
			b.WriteString(tv.Value)
			continue
		}
		scope := HLJSScope(tv.Token)
		if scope == "" {
			b.WriteString(escapeHTML(tv.Value))
			continue
		}
		b.WriteString(`<span class="` + hljsClasses(scope) + `">` + escapeHTML(tv.Value) + `</span>`)
	}
	return b.String()
}

// hljsScopes maps a token to its highlight.js scope. A token absent from the
// map takes its nearest mapped ancestor's scope.
var hljsScopes = map[*Token]string{
	Keyword:         "keyword",
	KeywordConstant: "literal",
	KeywordType:     "type",

	NameAttribute:     "attr",
	NameBuiltin:       "built_in",
	NameClass:         "title.class",
	NameConstant:      "variable.constant",
	NameDecorator:     "meta",
	NameEntity:        "symbol",
	NameException:     "title.class",
	NameFunction:      "title.function",
	NameLabel:         "symbol",
	NameNamespace:     "title.class",
	NameProperty:      "property",
	NameTag:           "name",
	NameVariable:      "variable",
	NameBuiltinPseudo: "variable.language",

	Literal:               "literal",
	LiteralDate:           "number",
	LiteralNumber:         "number",
	LiteralString:         "string",
	LiteralStringEscape:   "char.escape",
	LiteralStringInterpol: "subst",
	LiteralStringRegex:    "regexp",
	LiteralStringSymbol:   "symbol",

	Operator:     "operator",
	OperatorWord: "keyword",
	Punctuation:  "punctuation",

	Comment:            "comment",
	CommentHashbang:    "meta",
	CommentPreproc:     "meta",
	CommentPreprocFile: "meta",

	GenericDeleted:    "deletion",
	GenericEmph:       "emphasis",
	GenericHeading:    "section",
	GenericInserted:   "addition",
	GenericPrompt:     "meta.prompt",
	GenericStrong:     "strong",
	GenericSubheading: "section",
}

// HLJSScope returns the highlight.js scope of a token ("title.function"),
// or "" when it has none.
func HLJSScope(t *Token) string {
	for c := t; c != nil; c = c.Parent {
		if s, ok := hljsScopes[c]; ok {
			return s
		}
	}
	return ""
}

// hljsClasses turns a scope into class names: the first part prefixed with
// "hljs-", each deeper part suffixed with one underscore per level.
func hljsClasses(scope string) string {
	parts := strings.Split(scope, ".")
	out := "hljs-" + parts[0]
	for i, p := range parts[1:] {
		out += " " + p + strings.Repeat("_", i+1)
	}
	return out
}
