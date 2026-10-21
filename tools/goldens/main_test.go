// Copyright (c) the go-rouge/rouge authors
//
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// run drives the command tree the way a shell would and returns its output and
// error.
func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd := RootCmd()
	cmd.SetArgs(args)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	err := cmd.Execute()
	return out.String(), err
}

// TestRequiredFlags pins that neither subcommand will run without being told
// which gem, which lexer and which file -- a capture made against an
// unspecified oracle is worthless.
func TestRequiredFlags(t *testing.T) {
	for _, args := range [][]string{
		{"capture"},
		{"capture", "--lib", "x"},
		{"capture", "--lib", "x", "--tag", "toml"},
		{"verify", "--lib", "x", "--tag", "toml", "--in", "y"},
	} {
		if _, err := run(t, args...); err == nil {
			t.Errorf("%v: no error, want a missing-flag error", args)
		}
	}
}

// TestCaptureReportsAFailedInterpreter drives the arm that matters most: when
// the oracle cannot be run, the tool must say so and write nothing. Silence
// here would produce an empty golden that every later test would "pass"
// against.
func TestCaptureReportsAFailedInterpreter(t *testing.T) {
	saved := rubyBin
	rubyBin = filepath.Join(t.TempDir(), "no-such-ruby")
	defer func() { rubyBin = saved }()

	dst := filepath.Join(t.TempDir(), "out.html")
	_, err := run(t, "capture", "--lib", "x", "--tag", "toml", "--in", "y", "--out", dst)
	if err == nil {
		t.Fatal("a missing interpreter produced no error")
	}
	if !strings.Contains(err.Error(), "ruby") {
		t.Errorf("error does not name the interpreter: %v", err)
	}
	if _, statErr := os.Stat(dst); statErr == nil {
		t.Error("a failed capture still wrote an output file")
	}

	// verify fails the same way, rather than reporting a match it never made.
	if _, err := run(t, "verify", "--lib", "x", "--tag", "toml", "--in", "y", "--want", "z"); err == nil {
		t.Error("verify with a missing interpreter produced no error")
	}
}

// TestCaptureAndVerifyAgainstTheGem is the end-to-end control: with the gem
// available, the tool must reproduce a golden this repository already
// committed, byte for byte, and must report a MISMATCH against a golden that
// has been altered. It is skipped when the gem is not on this machine --
// ROUGE_GEM_LIB names the `lib` directory of an extracted rouge release.
func TestCaptureAndVerifyAgainstTheGem(t *testing.T) {
	lib := os.Getenv("ROUGE_GEM_LIB")
	if lib == "" {
		t.Skip("ROUGE_GEM_LIB is not set: no gem to ask")
	}
	corpus := filepath.Join("..", "..", "testdata", "toml.toml")
	golden := corpus + ".html"

	out, err := run(t, "verify", "--lib", lib, "--tag", "toml", "--in", corpus, "--want", golden)
	if err != nil {
		t.Fatalf("verify against the committed golden failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "reproduced") {
		t.Errorf("verify said %q", out)
	}

	// A golden that does not match must be reported, not glossed over.
	bad := filepath.Join(t.TempDir(), "bad.html")
	if err := os.WriteFile(bad, []byte("not the gem's answer"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, "verify", "--lib", lib, "--tag", "toml", "--in", corpus, "--want", bad); err == nil {
		t.Error("verify accepted a golden that does not match")
	}

	// capture writes the same bytes the golden holds.
	dst := filepath.Join(t.TempDir(), "toml.toml.html")
	if _, err := run(t, "capture", "--lib", lib, "--tag", "toml", "--in", corpus, "--out", dst); err != nil {
		t.Fatalf("capture: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("capture wrote %d bytes, the committed golden has %d", len(got), len(want))
	}
}
