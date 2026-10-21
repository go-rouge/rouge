// Copyright (c) the go-rouge/rouge authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rouge

import (
	onigmo "github.com/go-ruby-regexp/regexp"
)

// --- Docker ---
// A faithful transcription of Rouge::Lexers::Docker (rouge 5.0.0,
// lib/rouge/lexers/docker.rb).
//
// The body of a RUN instruction is delegated to the Shell lexer. The gem keeps
// ONE Shell instance per Dockerfile (`start { @shell = Shell.new(@options) }`)
// and delegates to it with `delegate @shell`, which calls continue_lex: a
// quote or a here-document opened on one continued line is still open on the
// next. subDelegate reproduces that; `@shell.reset!` at each RUN is subReset.
var dockerLexer = func() *RegexLexer {
	// The gem's KEYWORDS, joined with | and interpolated as a plain string (so
	// no (?-mix:) wrapper is added around it).
	const keywords = `FROM|MAINTAINER|CMD|LABEL|EXPOSE|ENV|ADD|COPY|ENTRYPOINT|VOLUME|USER|WORKDIR|ARG|STOPSIGNAL|HEALTHCHECK|SHELL`

	b := newRegexLexer("docker", "Docker", "dockerfile", "Dockerfile", "containerfile", "Containerfile")
	b.filenames("Dockerfile", "Dockerfile.*", "*.Dockerfile", "*.docker",
		"Containerfile", "Containerfile.*", "*.Containerfile")

	b.state("root").
		rule(`\s+`, Text).
		groupsRule(`(?i)^(FROM)(\s+)(.*)(\s+)(AS)(\s+)(.*)`,
			Keyword, TextWhitespace, LiteralString, TextWhitespace, Keyword, TextWhitespace, LiteralString).
		groupsRule(`(?i)^(ONBUILD)(\s+)(`+keywords+`)(.*)`, Keyword, TextWhitespace, Keyword, LiteralString).
		groupsRule(`(?i)^(`+keywords+`)\b(.*)`, Keyword, LiteralString).
		rule(`#.*?$`, Comment).
		cb(`(?i)^(?:ONBUILD\s+)?RUN(\s+)`, func(l *lexState, m *onigmo.MatchData) {
			l.emit(Keyword, m.Str(0))
			l.pos = m.End(0)
			l.push("run")
			l.subReset(shellLexer)
		}).
		rule(`\w+`, Text).
		rule(`[^\w]+`, Text).
		rule(`.`, Text)

	b.state("run").
		rule(`\n`, Text, pop()).
		rule(`(?m)\\.`, LiteralStringEscape).
		cb(`(?:\\.|[^\n\\])+`, func(l *lexState, m *onigmo.MatchData) {
			l.subDelegate(shellLexer, m.Str(0))
			l.pos = m.End(0)
		})

	return b.done()
}()
