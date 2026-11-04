package rouge

import "testing"

// TestRubySigilBodyKeepsNonASCII: a %-sigil body is scanned byte by byte, and a
// multi-byte character in it must come back unchanged (it used to be re-encoded
// one byte at a time: "é" -> "Ã©").
func TestRubySigilBodyKeepsNonASCII(t *testing.T) {
	for _, src := range []string{"%q(é)", "x = %w[日本 語]\n", "%r{é}i", "~ / + $\né <- 1\n%in% é %",
		"%\xea0\xea0", // a delimiter that is a lone non-ASCII byte (found by fuzzing)
		"%q\xea0\xea0"} {
		reconstruct(t, rubyLexer, src)
	}
}
