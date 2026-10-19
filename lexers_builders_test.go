// Copyright (c) the go-rouge/rouge authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rouge

import "testing"

// The C-family and HCL builders carry the gem's extension points: a set of
// builtins for C (`def self.builtins`), and sets of reserved words and
// builtins for HCL (`def self.reserved`, `def self.builtins`). Every lexer the
// gem ships leaves those sets EMPTY -- C, C++, HCL and Terraform all do -- so
// no corpus can reach the branches that read them, and a golden cannot cover
// what no input produces.
//
// These tests drive the builders directly with non-empty sets. That covers the
// branches, and it pins the contract the gem states: a word in `builtins` is
// Name::Builtin, a word in `reserved` is Keyword::Reserved, and neither opens
// a composite.

// qualnames reduces a token stream to its token names, which is what these
// tests assert on.
func qualnames(tvs []TokenValue) []string {
	out := make([]string, 0, len(tvs))
	for _, tv := range tvs {
		out = append(out, tv.Token.Qualname())
	}
	return out
}

// find returns the token emitted for the first occurrence of value.
func find(t *testing.T, tvs []TokenValue, value string) *Token {
	t.Helper()
	for _, tv := range tvs {
		if tv.Value == value {
			return tv.Token
		}
	}
	t.Fatalf("no token carried %q; stream was %v", value, qualnames(tvs))
	return nil
}

func TestCFamilyBuiltinsAreDispatched(t *testing.T) {
	lx := buildCFamily(cFamilyOpts{
		tag:          "c-builtins-probe",
		title:        "C (probe)",
		keywords:     cKeywords,
		keywordsType: cKeywordsType,
		reserved:     cReserved,
		builtins:     stringSet(`printf`),
	})
	got := lx.Lex("printf;\n")
	if tok := find(t, got, "printf"); tok != NameBuiltin {
		t.Errorf("a word in builtins gave %s, want Name.Builtin", tok.Qualname())
	}
	// A word in none of the sets is still a plain Name, so the dispatch has
	// not simply been widened.
	if tok := find(t, lx.Lex("notabuiltin;\n"), "notabuiltin"); tok != Name {
		t.Errorf("an ordinary word gave %s, want Name", tok.Qualname())
	}
}

func TestHCLReservedAndBuiltinsAreDispatched(t *testing.T) {
	lx := buildHCL(hclOpts{
		tag:          "hcl-sets-probe",
		title:        "HCL (probe)",
		keywords:     map[string]bool{},
		declarations: map[string]bool{},
		reserved:     stringSet(`forbidden`),
		constants:    stringSet(`true false null`),
		builtins:     stringSet(`jsonencode`),
	})
	if tok := find(t, lx.Lex("forbidden\n"), "forbidden"); tok != KeywordReserved {
		t.Errorf("a reserved word gave %s, want Keyword.Reserved", tok.Qualname())
	}
	if tok := find(t, lx.Lex("jsonencode\n"), "jsonencode"); tok != NameBuiltin {
		t.Errorf("a builtin gave %s, want Name.Builtin", tok.Qualname())
	}
	if tok := find(t, lx.Lex("true\n"), "true"); tok != KeywordConstant {
		t.Errorf("a constant gave %s, want Keyword.Constant", tok.Qualname())
	}
}

// TestHCLArrayAtTopLevel covers the bracket rule of :root. Every array in a
// real configuration follows a name, which has already pushed :composite, so
// it is :composite's bracket that fires there; :root's own is reached only
// when a document opens with one.
func TestHCLArrayAtTopLevel(t *testing.T) {
	got := FindLexer("hcl").Lex("[1, 2]\n")
	names := qualnames(got)
	if len(names) == 0 || names[0] != "Punctuation" {
		t.Fatalf("a leading [ gave %v, want Punctuation first", names)
	}
	if tok := find(t, got, "1"); tok != LiteralNumberInteger {
		t.Errorf("the array element gave %s, want Literal.Number.Integer", tok.Qualname())
	}
}
