package rouge

import (
	"strings"
	"testing"
)

func TestHLJSClasses(t *testing.T) {
	cases := map[string]string{
		"keyword":               "hljs-keyword",
		"title.function":        "hljs-title function_",
		"title.class.inherited": "hljs-title class_ inherited__",
	}
	for scope, want := range cases {
		if got := hljsClasses(scope); got != want {
			t.Errorf("%s: %q, want %q", scope, got, want)
		}
	}
}

func TestHLJSScopeInherits(t *testing.T) {
	cases := map[*Token]string{
		Keyword:              "keyword",
		KeywordDeclaration:   "keyword",
		KeywordConstant:      "literal",
		LiteralStringDouble:  "string",
		LiteralStringEscape:  "char.escape",
		LiteralNumberInteger: "number",
		NameFunctionMagic:    "title.function",
		NameVariableInstance: "variable",
		CommentSingle:        "comment",
		CommentPreprocFile:   "meta",
		NameBuiltinPseudo:    "variable.language",
		Text:                 "",
		TextWhitespace:       "",
		Name:                 "",
		Error:                "",
	}
	for tok, want := range cases {
		if got := HLJSScope(tok); got != want {
			t.Errorf("%s: %q, want %q", tok, got, want)
		}
	}
}

func TestHLJSFormatter(t *testing.T) {
	out, err := Highlight("// c\nfunc main() { s := \"<x>\"; n := 42 }\n", "go", "hljs")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<span class="hljs-comment">// c</span>`,
		`<span class="hljs-keyword">func</span>`,
		// Rouge's escaping, kept from the gem, leaves quotes alone.
		`<span class="hljs-string">"&lt;x&gt;"</span>`,
		`<span class="hljs-number">42</span>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in\n%s", want, out)
		}
	}
	if strings.Contains(out, `class="hljs-"`) || strings.Contains(out, "<x>") {
		t.Errorf("empty class or unescaped text in\n%s", out)
	}
	if FindFormatter("hljs") == nil {
		t.Error("the formatter is registered as hljs")
	}
	if got := (HLJSFormatter{}).Format([]TokenValue{{Token: Escape, Value: "<b>"}}); got != "<b>" {
		t.Errorf("an Escape token passes through: %q", got)
	}
}
