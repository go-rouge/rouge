// Copyright (c) the go-rouge/rouge authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rouge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fuzzTags are the lexers FuzzLexers drives: one canonical tag per registered
// lexer. A lexer is a parser fed untrusted input -- a README, a pasted
// snippet, a file from a repository nobody here controls -- so each one is
// fuzzed.
//
// ⛔ This list held only the eight lexers added in v0.7.0, while the README
// said "every lexer is fuzzed". That was 8 of 31: the twenty-three older
// lexers -- the ones that had been in front of untrusted input the longest --
// were the ones nobody fuzzed. The claim was written when the list was new and
// stopped being true the moment it stopped being the whole list.
//
// TestFuzzTagsCoverEveryLexer below is what keeps the sentence honest: adding
// a lexer without adding it here now fails the suite.
var fuzzTags = []string{
	"c", "cpp", "cobol", "css", "diff", "docker", "fortran", "go", "hcl",
	"html", "java", "javascript", "json", "jsx", "make", "markdown", "php",
	"plaintext", "protobuf", "python", "ruby", "rust", "shell", "sql",
	"terraform", "latex", "toml", "tsx", "typescript", "xml", "yaml",
}

// fuzzSeedFiles are corpus files that make good starting points: each is real
// input for one of the lexers above.
var fuzzSeedFiles = []string{
	"toml.toml", "xml.xml", "xml2.svg", "make.mk", "typescript.ts",
	"typescript2.ts", "jsx.jsx", "jsx2.jsx", "tsx.tsx", "tsx2.tsx",
	"docker.docker", "protobuf.proto",
}

// FuzzLexers asserts the one invariant that holds for every lexer here,
// whatever the input: lexing ACCOUNTS FOR EVERY BYTE. The concatenated token
// values must equal the input exactly -- no byte dropped (which would silently
// delete source from a rendered page) and none invented (which would be an
// injection). A rule whose capture groups do not cover its whole match fails
// this, and so does an infinite loop, a panic, or a slice out of range.
func FuzzLexers(f *testing.F) {
	for i := range fuzzTags {
		for _, name := range fuzzSeedFiles {
			src, err := os.ReadFile(filepath.Join("testdata", name))
			if err != nil {
				f.Fatal(err)
			}
			f.Add(uint8(i), string(src))
		}
	}
	// Shapes a corpus of valid files does not contain: unterminated
	// delimiters, delimiters that nest far past anything a human writes, and
	// the state-machine edges (an interpolation that never closes, a
	// here-document opener with no body, a tag that never ends).
	for _, s := range []string{
		"", "\n", "\x00", "\xff\xfe", "é", "\r\n\r\n",
		"\"", "'", "`", "'''", "\"\"\"", "<!--", "<![CDATA[", "<a", "</",
		"${", "$(", "#{", "{{{{{{{{{{{{{{{{", "[[[[[[[[[[[[[[[[",
		"t@\\", "\\\n\\\n\\\n", "RUN \"", "FROM a AS",
		"a = [[[[", "[[t]]", "a=\"\"\"", "x:\n\t$(shell (((",
		"<x " + strings.Repeat("a", 64) + "=", "/*", "//\\\n", "'''a",
		"<T,>", "<a<b<c<d<e", "{foo {bar", "as ", "satisfies\t",
	} {
		for i := range fuzzTags {
			f.Add(uint8(i), s)
		}
	}

	f.Fuzz(func(t *testing.T, which uint8, src string) {
		tag := fuzzTags[int(which)%len(fuzzTags)]
		lx := FindLexer(tag)
		var b strings.Builder
		for _, tv := range lx.Lex(src) {
			if tv.Value == "" {
				t.Fatalf("%s: emitted an empty value for %s", tag, tv.Token.Qualname())
			}
			b.WriteString(tv.Value)
		}
		if b.String() != src {
			t.Fatalf("%s: lexing did not account for every byte\n got: %q\nwant: %q",
				tag, b.String(), src)
		}
	})
}
