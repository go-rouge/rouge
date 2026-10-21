<p align="center"><img src="https://raw.githubusercontent.com/go-rouge/brand/main/social/go-rouge-rouge.png" alt="go-rouge/rouge" width="720"></p>

# rouge — go-rouge

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-rouge.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of the Ruby [`rouge`](https://github.com/rouge-ruby/rouge)
syntax highlighter** — the state-machine regex lexers, the token hierarchy, and
the HTML formatters of the gem, emitting exactly the CSS short-codes (`k`, `s`,
`c`, `nf`, …) `rouge` emits. It tokenizes source and renders `<span class>` HTML
that matches the gem **byte-for-byte** on the supported lexers, **without any
Ruby runtime**.

It is a sibling of [go-ruby-regexp](https://github.com/go-ruby-regexp/regexp)
(the Onigmo engine it lexes with), [go-ruby-erb](https://github.com/go-ruby-erb/erb),
and [go-ruby-yaml](https://github.com/go-ruby-yaml/yaml), and is bound into
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby) to give `rbgo` a
CGO-free `Rouge.highlight`.

## API

```go
import "github.com/go-rouge/rouge"

// Rouge.highlight(text, lexer, formatter) -> HTML
html, err := rouge.Highlight(`puts "hello"`, "ruby", "html")
// => <span class="nb">puts</span> <span class="s2">"hello"</span>

// Rouge::Lexer.find / find_fancy / guess
lex := rouge.FindLexer("rb")          // by tag or alias (case-insensitive)
lex = rouge.FindFancy("ruby?opt=1")   // "tag" or "tag?opts"
lex = rouge.Guess(src)                // content sniff; never nil (PlainText fallback)

// Token stream + the HTML / HTMLInline formatters directly
toks := lex.Lex(src)                          // []TokenValue{Token, Value}
out := rouge.HTMLFormatter{}.Format(toks)     // <span class="…"> per token
page := rouge.WrapHTML(out)                   // <pre class="highlight"><code>…</code></pre>
inline := rouge.HTMLInlineFormatter{Theme: rouge.Github}.Format(toks) // inline styles
```

- **Token model** — `rouge.Token` mirrors `Rouge::Token::Tokens`: a tree with a
  dotted `Qualname()` (`Literal.String.Double`), the CSS `Shortname` the HTML
  formatter emits (`s2`), `Matches(ancestor)`, and `TokenByName("Keyword.Constant")`.
- **Formatters** — `HTML` (`<span class="shortcode">`, the gem's exact codes;
  `Text`/`Escape` pass through) and `HTMLInline` (inline `style=` from a `Theme`),
  registered as `"html"` / `"html_inline"`; plus `WrapHTML` for the gem's
  `<pre class="highlight"><code>` page wrapper. Beyond the gem, `HLJS`
  (`"hljs"`) writes [highlight.js class
  names](https://highlightjs.readthedocs.io/en/latest/css-classes-reference.html)
  (`hljs-keyword`, `hljs-title function_`), so stylesheets written for
  highlight.js colour Rouge's tokens; `HLJSScope(token)` gives the mapping.
  go-slides uses it for Marp's themes.
- **Themes** — `Base16`, `Github`, `ThankfulEyes`: their resolved
  `rendered_rules` captured token-by-token, with `StyleFor` walking the ancestor
  chain exactly like `Rouge::Theme.get_style`.

## Lexers

**31 lexers.** Each one is a near-mechanical transcription of the gem's
`state … rule …` definition, validated by a **differential oracle** against the
reference `rouge` 5.0.0 gem: every committed `testdata/*` corpus file is
highlighted and compared **byte-for-byte** to the gem's HTML (the `*.html`
goldens). `tools/goldens` is the capture tool that produces them; its `verify`
subcommand reproduces a golden that is already committed, which is the control
on the capture itself.

Where the gem expresses a lexer as a subclass, so does this port: one builder
carries the shared states and the hooks a subclass uses, so a rule added to C
is a rule C++ gets, and a rule added to JavaScript is a rule TSX gets.

**Byte-faithful** (gem-identical HTML on the benign corpus):

| Lexer | Tag(s) | Notes |
|-------|--------|-------|
| Ruby | `ruby`, `rb` | symbols, `%w`/`%r`/`%(…)` sigils, heredocs (`<<`/`<<-`/`<<~`), interpolation, ternary, method-call disambiguation |
| Go | `go`, `golang` | |
| Python | `python`, `py` | f-strings, `case`/`match`, decorators, doctest |
| JavaScript | `javascript`, `js` | regex/template/object/ternary state machine |
| JSON | `json` | |
| YAML | `yaml`, `yml` | indentation state machine, anchors, block scalars, tags |
| HTML | `html` | delegates `<script>`/`<style>` to JS/CSS |
| CSS | `css` | property/builtin/color/function sets |
| Shell | `shell`, `bash`, `zsh`, `ksh`, `sh` | heredocs, `$()`/`${}`/`$(())`, `case` |
| Diff | `diff`, `patch`, `udiff` | with content detector |
| Markdown | `markdown`, `md`, `mkd` | delegates fenced code to the named lexer, frontmatter to YAML |
| SQL | `sql` | case-insensitive keyword/type sets |
| PHP | `php`, `php3`, `php4`, `php5` | `start_inline` option, `<?php`/`<?=` template delegation to HTML, string interpolation, heredocs, enums, function/property-hook sub-states |
| Java | `java` | text blocks, annotations, method signatures (the gem delegates the return arguments to Java itself), all four numeric bases with `_` separators |
| C | `c` | preprocessor states (`#if 0`, `#include <>`/`""`, macro continuation), labels, function-signature recursion |
| C++ | `cpp`, `c++` | C plus templates, raw strings with a delimiter backreference, `'` digit separators, `<=>`, `::`-aware `case` |
| Rust | `rust`, `rs`, `rust,no_run`, `rs,no_run`, `rust,ignore`, `rs,ignore`, `rust,should_panic`, `rs,should_panic` | nesting doc/plain block comments that keep the OUTER kind, macro delimiter counting, raw strings, `rustc` shebang detector |
| COBOL | `cobol` | divisions, sections and the IBM z/OS reserved-word list; fixed-form and `*>` comments |
| Fortran | `fortran` | Fortran 2008 free form: kind parameters, `B`/`O`/`Z` literals, `.EQ.`-style operator words, two-word keywords, `FORMAT` state |
| HCL | `hcl` | Hashicorp Configuration Language (`*.hcl`, `*.nomad`) |
| Terraform | `terraform`, `tf` | HCL plus `${…}` interpolation, regex literals, `var`/`local` declarations |
| TOML | `toml` | tables and arrays of tables, dotted and quoted keys, dates and times, all four numeric bases with `_` separators, multiline basic and literal strings |
| XML | `xml` | also **SVG**, PLIST, XSL/XSD/WSDL/RSS (`*.svg`, `*.plist`, `*.xsl`, …) — CDATA, processing instructions, entities, attribute states; content detector on the XML declaration and the doctype |
| Make | `make`, `makefile`, `mf`, `gnumake`, `bsdmake` | recipes, macro expansion, the GNU function list, conditionals, `define`; recipe bodies delegated to a **continuing** Shell lexer |
| TypeScript | `typescript`, `ts` | JavaScript plus the TypeScript word sets, `as`/`satisfies`, optional-property labels, `?.` and `??` |
| JSX | `jsx`, `react` | JavaScript plus elements, attributes and `{…}` interpolation; the text between tags delegated to a **continuing** HTML lexer |
| TSX | `tsx` | JSX plus TypeScript, and the `<T,>` / `<Type>` generic-argument rules |
| Dockerfile | `docker`, `dockerfile`, `containerfile` | instruction keywords, `FROM … AS …`, `ONBUILD`; `RUN` bodies delegated to a **continuing** Shell lexer |
| Protobuf | `protobuf`, `proto` | messages, enums, services, `oneof`, extensions, scalar types, all numeric forms |
| PlainText | `plaintext`, `text` | fallback |

**Documented simplifications** (honest deviations from the gem):

- **Markdown** fenced code blocks: the closed-fence form is byte-faithful; the
  gem's anonymous dynamic state for an *unclosed* fence is approximated.
- **HTML** `<script>`/`<style>` delegation is per-chunk (matching the gem); like
  the gem, a token straddling a bare `<` inside an embedded string can split
  (the gem itself emits an `Error` there).
- **Ruby** `%`-sigil string bodies are scanned in one pass (nesting +
  interpolation + escapes preserved); the heredoc `(?<!\p{Word})` guard is done
  against the preceding byte (the engine has no variable-width lookbehind).
- **Guess** uses each lexer's content detector only (a subset of the gem's
  multi-signal guesser) — enough for the unambiguous formats (diff, YAML,
  shebangs, doctype).
- **COBOL and Fortran** drop the leading `^` of three rules, and COBOL drops a
  `(?<![\w-])` guard, because in the gem **neither assertion constrains
  anything**. Rouge scans with `StringScanner#skip`, where both are evaluated
  as though the scan pointer were the start of the string. Measured against
  rouge 5.0.0: `X      * c` gives `Comment::Special` for `      * c`
  immediately after the `X`; `selected_real_kind(15, 307)` gives `Name::Label`
  for both numbers, from Fortran's `^(\s*)([0-9]+)\b` statement-label rule; and
  `= -ABC.` gives `Name` then `Punctuation`, not a declined match. This engine
  anchors every rule with `\G` and evaluates `^` and lookbehind against the
  real buffer, so transcribing them literally makes the rules unreachable. `\G`
  alone already says "at the scan position", which is what the gem's
  assertions amount to.
- **HCL** does not carry the gem's here-document rule or its `:heredoc` /
  `:heredoc_nl` states, because the gem cannot reach them: `hcl.rb` lists the
  operator class `[-+*\/!%&<>|=:?]` — which contains `<` — *before* the
  `(<<-?)…` rule, so `<<EOF` is consumed as two operators and a name. Measured:
  `testdata/hcl.hcl` carries two here-documents and the gem's own golden for it
  contains zero `Str::Heredoc` and zero `Name::Constant` spans. If upstream
  reorders those rules, the states go back.
- **Rust** uses `[\p{L}\p{Nl}_]` and `\p{Word}` where the gem writes
  `\p{XID_Start}` and `\p{XID_Continue}`, which this engine does not provide.
  `\p{Word}` is Ruby's `\p{L}\p{M}\p{N}\p{Pc}`, a superset of XID_Continue and
  the same set for any identifier a program can carry; the Rust corpus includes
  non-ASCII identifiers so the substitution is checked against the gem rather
  than assumed.
- **TypeScript** drops the three lookbehinds of the gem's `as` / `satisfies`
  rule, for the same reason COBOL and Fortran drop theirs: in the gem they
  **constrain nothing**. Measured against rouge 5.0.0 — fed `x.as y`, `1as b`,
  `...as c` and `z as d`, the gem marks `as ` as `Keyword::Declaration` on
  **all four** lines, including the one after a dot that the gem's own source
  comment gives as the negative example. What actually keeps `.as(` out is the
  rule's trailing `\s+`, not its lookbehind.
- **Protobuf** keeps the gem's `[+-=]` as written. It is a character *range*
  from `+` (U+002B) to `=` (U+003D) rather than the three characters it looks
  like, so it also covers `,` `-` `.` `/` `0`-`9` `:` `;` `<`. Every one of
  those that could reach it is claimed by an earlier rule, so the widening is
  inert — but it is the gem's behaviour.
- **Make** keeps the gem's unreachable function rule in `:shell`: the macro
  rule listed above it matches `$(` first, so the function rule can never fire.
  Like HCL's here-document, it is recorded rather than quietly "fixed", so that
  a reordering upstream is a change this port can follow.
- **PHP** does not model `funcnamehighlighting` (the gem's built-in-function
  table promoting known calls to `Name.Builtin`); calls render as `Name.Function`,
  exactly as the gem does with `funcnamehighlighting:false`, the configuration the
  PHP goldens are captured under. Its `start_inline` fancy-spec option
  (`php?start_inline=1`) is honoured. `FindFancy` now applies a lexer's recognised
  options where the lexer defines them (PHP's `start_inline`); other options are
  still accepted and ignored.

## Engine

Lexing runs on the sibling [go-ruby-regexp](https://github.com/go-ruby-regexp/regexp)
Onigmo engine via its `MatchAt(src, pos)` primitive, which anchors `\G` at the
scan cursor while keeping the whole buffer visible — so a rule's `^` matches only
at a true line start and lookbehind sees the real prefix.

That is **stricter than the gem**, and the difference is why several rules are
transcribed without their assertions (see the COBOL/Fortran and TypeScript
notes above). Rouge tests rules with `StringScanner#skip` at the scan pointer,
where a lookbehind cannot read the bytes before that pointer: a negative
lookbehind always succeeds and a positive one always fails. Each such rule is
therefore written from what the gem **does**, measured, rather than from what
its pattern reads like.

A lexer may also **delegate** to another. Two shapes exist, both the gem's:
`delegate` lexes a chunk with a fresh sub-lexer, while `subDelegate` is the
gem's `delegate @sub` — it calls the sub-lexer's `continue_lex`, so its state
stack survives between chunks and a quote opened in one chunk is still open in
the next. Make, Dockerfile and JSX need the second kind.

Building this port upstreamed a
handful of Onigmo-faithfulness fixes to that engine (`\<punct>` and `\f \v \a \e`
escapes, the `\p{Nl}`/`\p{No}`/`\p{Cf}` property subcategories, and `MatchAt`
itself).

## Tests & coverage

Tests are **deterministic and Ruby-free**: the differential goldens were captured
from the `rouge` gem once and committed, so `go test` reproduces gem-faithfulness
without the gem on `PATH` (the gem is not stdlib, and the cross-arch / Windows CI
lanes have no Ruby). Coverage is **100%** including error branches, with **zero**
uncovered blocks rather than a rounded total.

```
go test ./...
```

A lexer is a parser fed untrusted input, so every lexer is **fuzzed**
(`FuzzLexers`, seed corpus committed under `testdata/fuzz/`). The invariant it
asserts is the one that holds whatever the input: lexing **accounts for every
byte** — none dropped, which would delete source from a rendered page, and none
invented, which would be an injection.

```
go test -fuzz FuzzLexers
```

Lexing is **linear in the input**, including against inputs built to defeat the
lazy quantifiers the gem's rules use — unterminated comments, strings, tags,
macros and template literals. Measured at 4k/8k/16k/32k bytes across the
lexers, the worst doubling ratio is 2.36x; 4x would be quadratic.

CI builds and tests on **3 OSes** (linux/macos/windows) and the **six 64-bit Go
targets** (amd64/arm64/riscv64/loong64/ppc64le/s390x).

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the go-rouge/rouge authors.
The bundled themes' colour tables derive from the BSD-3-licensed `rouge` gem.

## WebAssembly

Being pure Go (CGO=0), this library also compiles to **WebAssembly** — both
`GOOS=js GOARCH=wasm` (browser / Node.js) and `GOOS=wasip1 GOARCH=wasm` (WASI).
CI builds both targets on every push, alongside the six 64-bit native/qemu arches.

```sh
GOOS=js     GOARCH=wasm go build ./...   # browser / Node
GOOS=wasip1 GOARCH=wasm go build ./...   # WASI (wasmtime, wasmer, wasmedge, …)
```
