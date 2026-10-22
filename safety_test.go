// Copyright (c) the go-rouge/rouge authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rouge

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"
)

// registeredTags returns one canonical tag per registered lexer.
//
// lexerRegistry maps every tag AND alias to its lexer, so it has far more
// entries than there are lexers; Tag() is the canonical one, and that is what
// is counted here. Reading the registry is the point: a hand-kept list of
// lexers is a list that falls behind the library it describes, which is
// exactly how "every lexer is fuzzed" came to mean eight of thirty-one.
func registeredTags() []string {
	seen := map[string]bool{}
	for _, l := range lexerRegistry {
		seen[l.Tag()] = true
	}
	tags := make([]string, 0, len(seen))
	for t := range seen {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	return tags
}

// TestFuzzTagsCoverEveryLexer is what makes the README's "every lexer is
// fuzzed" a fact rather than a sentence.
//
// ⛔ It did not hold when this test was written. fuzzTags listed only the
// eight lexers added in v0.7.0, so the twenty-three that had been exposed to
// untrusted input the longest were the ones nobody fuzzed. Nothing failed: a
// fuzz list is a list, and a short one fuzzes happily.
//
// Both directions are checked. A lexer missing from fuzzTags is the defect
// that was here; a tag in fuzzTags that no longer resolves is the other one —
// it would fuzz nothing while still lengthening the list, which reads like
// coverage.
func TestFuzzTagsCoverEveryLexer(t *testing.T) {
	inFuzz := map[string]bool{}
	for _, tag := range fuzzTags {
		l := FindLexer(tag)
		if l == nil {
			t.Errorf("fuzzTags names %q, which no lexer registers: it fuzzes nothing", tag)
			continue
		}
		inFuzz[l.Tag()] = true
	}

	var missing []string
	for _, tag := range registeredTags() {
		if !inFuzz[tag] {
			missing = append(missing, tag)
		}
	}
	if len(missing) > 0 {
		t.Errorf("%d lexer(s) are registered but never fuzzed: %s\n"+
			"A lexer is a parser fed untrusted input. Add the tag to fuzzTags in "+
			"fuzz_test.go, or stop claiming in README.md that every lexer is fuzzed.",
			len(missing), strings.Join(missing, ", "))
	}
}

// adversarial builds an input of about n bytes designed to defeat a lazy
// quantifier: an opening construct the lexer will look for the end of, which
// never arrives. These are the shapes the gem's rules scan for lazily —
// comments, strings, tags, macros and template literals — so they are where a
// rule that backtracks would show it.
func adversarial(open, filler string, n int) string {
	var b strings.Builder
	b.Grow(n + len(open))
	b.WriteString(open)
	for b.Len() < n {
		b.WriteString(filler)
	}
	return b.String()
}

// timeLex returns the SMALLEST time taken to lex src over several attempts.
//
// The minimum is the right statistic under contention: other load can only
// make a run slower, never faster, so the smallest sample is the one least
// polluted by it. A mean would track the machine's mood — and this test ran
// first on a machine at load 43, where it would have measured nothing else.
func timeLex(tb testing.TB, l Lexer, src string) time.Duration {
	tb.Helper()
	// The same repetition count at every size, so it cancels in the ratio
	// while lifting each measurement clear of the clock's resolution.
	const reps = 5
	best := time.Duration(1<<63 - 1)
	for trial := 0; trial < 3; trial++ {
		start := time.Now()
		for i := 0; i < reps; i++ {
			l.Lex(src)
		}
		if d := time.Since(start); d < best {
			best = d
		}
	}
	return best
}

// quadraticLexers are the lexers known to be super-linear on an unterminated
// block comment. See https://github.com/go-rouge/rouge/issues — the C-family
// function-signature rule scans forward for a `{` or `;` from every scan
// position, so O(n) attempts of O(n) work is O(n^2).
//
// ⛔ They are named here, not silently skipped. An exclusion list a reader can
// see is a defect being tracked; an exclusion folded into the loop condition
// is a defect being hidden. When the rule is fixed this list goes to nil and
// the test immediately covers them.
//
// Measured 2026-10-22 at 1k/2k/4k: c 4.00x, java 3.97x, cpp 3.96x, against
// <= 2.19x for the other 28 lexers.
var quadraticLexers = map[string]string{
	"c":    "function-signature rule scans to a `{`/`;` that never arrives",
	"cpp":  "shares C's function-signature rule",
	"java": "same rule shape as the C family",
}

// TestLexingIsLinearInTheInput measures what the README claims: that lexing is
// linear in the input, including against inputs built to defeat the lazy
// quantifiers the gem's rules use.
//
// That sentence was published with nothing in the repository reproducing it —
// and when the measurement was finally written, it turned out to be false for
// three of the thirty-one lexers. 28 are linear; C, C++ and Java are
// quadratic, and a 32 KiB unterminated comment takes them about 26 seconds.
//
// What this guards against is catastrophic backtracking on untrusted input: a
// highlighted README that hangs the process rendering it. That failure is not
// subtle — it produces ratios of 4x and up per doubling, not 2.1 — so the
// threshold is the quadratic line itself and the test stays a regression
// guard rather than a tripwire for ordinary timing noise.
func TestLexingIsLinearInTheInput(t *testing.T) {
	if testing.Short() {
		t.Skip("timing measurement; skipped under -short")
	}

	cases := []struct {
		tag, open, filler string
	}{
		{"c", "/*", "aaaa "},                  // unterminated block comment
		{"cpp", "/*", "aaaa "},                //
		{"java", "/*", "aaaa "},               //
		{"css", "/*", "aaaa "},                //
		{"javascript", "`${", "aaaa "},        // unterminated template literal
		{"typescript", "`${", "aaaa "},        //
		{"jsx", "{/*", "aaaa "},               //
		{"tsx", "`${", "aaaa "},               //
		{"xml", "<tag attr=\"", "aaaa "},      // unterminated attribute
		{"html", "<div class=\"", "aaaa "},    //
		{"make", "$(shell ", "aaaa "},         // unterminated function call
		{"docker", "RUN foo \\", "aaaa \\\n"}, // continuation that never ends
		{"toml", "s = \"\"\"", "aaaa "},       // unterminated multi-line string
		{"protobuf", "/*", "aaaa "},           //
		{"python", "'''", "aaaa "},            // unterminated triple-quote
		{"ruby", "=begin", "aaaa "},           // unterminated block comment
		{"sql", "/*", "aaaa "},                //
		{"shell", "$(", "aaaa "},              // unterminated substitution
		{"hcl", "<<EOF", "aaaa "},             // here-document with no terminator
		{"latex", "\\begin{verbatim}", "aa "}, // unterminated environment
	}

	// 2.8 sits clear of linear (2.0, with noise) and well under quadratic
	// (4.0). The census that produced it found nothing between 2.19x and
	// 3.96x, so the gap is real rather than a threshold chosen to make the
	// current numbers pass.
	const superLinear = 2.8
	// Two doublings is enough to tell 2x from 4x, and keeps the test to
	// seconds. Measuring up to 32 KiB is what made the first version of this
	// test hit the 10-minute panic — on the quadratic lexer it was built to
	// find, which it could not report because it never finished.
	sizes := []int{1 << 10, 2 << 10, 4 << 10}

	for _, c := range cases {
		t.Run(c.tag, func(t *testing.T) {
			if why, known := quadraticLexers[c.tag]; known {
				t.Skipf("known super-linear: %s", why)
			}
			l := FindLexer(c.tag)
			if l == nil {
				t.Fatalf("no lexer registers %q", c.tag)
			}
			times := make([]time.Duration, len(sizes))
			for i, n := range sizes {
				times[i] = timeLex(t, l, adversarial(c.open, c.filler, n))
			}
			for i := 1; i < len(sizes); i++ {
				if times[i-1] <= 0 {
					t.Fatalf("%d bytes measured as zero time: the clock cannot see this, "+
						"raise reps in timeLex", sizes[i-1])
				}
				ratio := float64(times[i]) / float64(times[i-1])
				if ratio >= superLinear {
					t.Errorf("doubling %d->%d bytes multiplied the time by %.2fx "+
						"(%v -> %v); linear is 2x and 4x is quadratic. A %s rule is "+
						"scanning the whole remaining input from every position.",
						sizes[i-1], sizes[i], ratio, times[i-1], times[i], c.tag)
				}
				t.Logf("%6d -> %6d bytes: %.2fx (%v -> %v)",
					sizes[i-1], sizes[i], ratio, times[i-1], times[i])
			}
		})
	}
}

// TestAdversarialInputIsStillAccountedFor is the correctness half of the test
// above: a lexer that stayed fast by DROPPING the input it could not match
// would pass the timing check. Every byte must still come back.
func TestAdversarialInputIsStillAccountedFor(t *testing.T) {
	src := adversarial("/*", "aaaa ", 4<<10)
	for _, tag := range []string{"c", "cpp", "java", "css", "sql", "protobuf"} {
		l := FindLexer(tag)
		if l == nil {
			t.Fatalf("no lexer registers %q", tag)
		}
		var got strings.Builder
		for _, tok := range l.Lex(src) {
			got.WriteString(tok.Value)
		}
		if got.String() != src {
			t.Errorf("%s: lexing %d bytes of unterminated comment returned %d bytes",
				tag, len(src), got.Len())
		}
	}
	// A planted positive: the comparison must be able to see a difference at
	// all. Without it, a Lex that returned the input verbatim as one token
	// would pass, and so would a broken comparison.
	if fmt.Sprint(src) == fmt.Sprint(src+"x") {
		t.Fatal("the comparison cannot distinguish two different strings")
	}
}
