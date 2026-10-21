// Copyright (c) the go-rouge/rouge authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rouge

import (
	"strings"

	onigmo "github.com/go-ruby-regexp/regexp"
)

// --- XML (and SVG, which is XML) ---
// A faithful transcription of Rouge::Lexers::XML (rouge 5.0.0,
// lib/rouge/lexers/xml.rb), including its content detector. The gem claims
// *.svg, *.plist, *.xsl and friends with this one lexer, so there is no
// separate SVG lexer to port.

// xmlName is the gem's name class, from
// https://www.w3.org/TR/xml11/#sec-suggested-names. The "·" (U+00B7 middle dot)
// is written as its literal UTF-8 bytes, as in the HTML lexer.
const xmlName = `[\p{L}:_][\p{Word}\p{Cf}:.·-]*`

var xmlLexer = func() *RegexLexer {
	b := newRegexLexer("xml", "XML")
	b.filenames("*.xml", "*.xsl", "*.rss", "*.xslt", "*.xsd", "*.wsdl", "*.svg",
		"*.plist")
	// The gem's detect?: not XML if the doctype names html; XML if the text
	// opens with an <?xml ...?> declaration; XML if it has any doctype at all.
	// TextAnalyzer#doctype allows an <?xml?> prologue before the DOCTYPE, and
	// `doctype?(/html/)` is a substring test on the doctype name, which is what
	// strings.Contains does.
	b.detectWith(func(text string) bool {
		doctype := ""
		if re, err := onigmo.Compile(`(?m)\A\s*(?:<\?.*?\?>\s*)?<!DOCTYPE\s+(.+?)>`); err == nil {
			if m := re.Match(text); m != nil {
				doctype = m.Str(1)
			}
		}
		if strings.Contains(doctype, "html") {
			return false
		}
		if re, err := onigmo.Compile(`\A<\?xml\b`); err == nil && re.MatchString(text) {
			return true
		}
		return doctype != ""
	})

	b.state("root").
		rule(`[^<&]+`, Text).
		rule(`&\S*?;`, NameEntity).
		rule(`<!\[CDATA\[.*?\]\]>`, CommentPreproc).
		rule(`<!--`, Comment, push("comment")).
		rule(`<\?.*?\?>`, CommentPreproc).
		rule(`<![^>]*>`, CommentPreproc).
		// open tags
		rule(`(?m)<\s*`+xmlName, NameTag, push("tag")).
		// self-closing tags
		rule(`(?m)<\s*/\s*`+xmlName+`\s*>`, NameTag)

	b.state("comment").
		rule(`(?m)[^-]+`, Comment).
		rule(`-->`, Comment, pop()).
		rule(`-`, Comment)

	b.state("tag").
		rule(`(?m)\s+`, Text).
		rule(`(?m)`+xmlName+`\s*=`, NameAttribute, push("attr")).
		rule(`/?\s*>`, NameTag, pop())

	b.state("attr").
		rule(`(?m)\s+`, Text).
		rule(`(?m)".*?"|'.*?'|[^\s>]+`, LiteralString, pop())

	return b.done()
}()
