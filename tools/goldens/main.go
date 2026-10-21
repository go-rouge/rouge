// Copyright (c) the go-rouge/rouge authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Command goldens captures reference HTML from the Ruby rouge gem.
//
// Every lexer in go-rouge is validated byte-for-byte against the gem's own
// output (see golden_test.go). New lexers therefore need new goldens, and only
// the gem can produce them: it is the oracle being ported, so there is nothing
// to write in Go that would answer the same question.
//
//	goldens capture --lib <gem lib dir> --tag <lexer tag> --in <file> [--out <file>]
//	goldens verify  --lib <gem lib dir> --tag <lexer tag> --in <file> --want <golden>
//
// `verify` is the control: it reproduces an ALREADY COMMITTED golden and says
// whether the bytes match. Run it before trusting a capture for a language
// that has no golden yet -- a capture tool that cannot reproduce a known
// answer cannot be trusted to produce an unknown one.
//
// The gem is NOT installed as a gem here. Point --lib at the `lib` directory
// of an extracted rouge release (the one holding rouge.rb); that directory is
// put on Ruby's load path for one `ruby -e` run.
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

// script is the Ruby driver. It reads the source from a file named on the
// command line rather than from an interpolated string, so no part of the
// corpus can be read as Ruby.
//
// ⛔ The source MUST be read as UTF-8. File.binread hands back ASCII-8BIT, and
// the gem's lexers are UTF-8 regexps, so a corpus containing any non-ASCII
// byte dies with "incompatible encoding regexp match (UTF-8 regexp with
// ASCII-8BIT string)". With an all-ASCII corpus the two are indistinguishable,
// which is why this went unnoticed until a Rust identifier carried an accent.
const script = `
require "rouge"
tag = ARGV[0]
src = File.read(ARGV[1], encoding: "UTF-8")
lexer = Rouge::Lexer.find(tag)
abort("no such lexer: #{tag}") if lexer.nil?
$stdout.binmode
$stdout.write(Rouge::Formatters::HTML.new.format(lexer.lex(src)))
`

// rubyBin is the interpreter to run. It is a variable so a test can point it
// at something that does not exist and drive the failure arm.
var rubyBin = "ruby"

// osExit is os.Exit, replaceable in tests.
var osExit = os.Exit

func main() {
	if err := RootCmd().Execute(); err != nil {
		osExit(1)
	}
}

// RootCmd builds the command tree. Tests drive it with SetArgs/SetOut/SetErr.
func RootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:          "goldens",
		Short:        "Capture reference HTML from the Ruby rouge gem",
		SilenceUsage: true,
	}
	root.AddCommand(captureCmd(), verifyCmd())
	return root
}

// commonFlags declares the three flags both subcommands need and returns
// pointers to their values.
func commonFlags(cmd *cobra.Command) (lib, tag, in *string) {
	lib = cmd.Flags().String("lib", "", "the gem's lib directory (the one holding rouge.rb)")
	tag = cmd.Flags().String("tag", "", "lexer tag, e.g. toml")
	in = cmd.Flags().String("in", "", "source file to highlight")
	cobra.CheckErr(cmd.MarkFlagRequired("lib"))
	cobra.CheckErr(cmd.MarkFlagRequired("tag"))
	cobra.CheckErr(cmd.MarkFlagRequired("in"))
	return lib, tag, in
}

func captureCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "capture",
		Short:        "Write the gem's HTML for one corpus file",
		SilenceUsage: true,
	}
	lib, tag, in := commonFlags(cmd)
	out := cmd.Flags().String("out", "", "write the HTML here (default: <in>.html)")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		html, err := capture(*lib, *tag, *in)
		if err != nil {
			return err
		}
		dst := *out
		if dst == "" {
			dst = *in + ".html"
		}
		if err := os.WriteFile(dst, html, 0o644); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "wrote %s (%d bytes)\n", dst, len(html))
		return nil
	}
	return cmd
}

func verifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "verify",
		Short:        "Check that the gem still reproduces a committed golden",
		SilenceUsage: true,
	}
	lib, tag, in := commonFlags(cmd)
	want := cmd.Flags().String("want", "", "the committed golden to compare against")
	cobra.CheckErr(cmd.MarkFlagRequired("want"))
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		html, err := capture(*lib, *tag, *in)
		if err != nil {
			return err
		}
		golden, err := os.ReadFile(*want)
		if err != nil {
			return err
		}
		if !bytes.Equal(html, golden) {
			return fmt.Errorf("MISMATCH %s: captured %d bytes, committed %d bytes",
				filepath.Base(*in), len(html), len(golden))
		}
		fmt.Fprintf(cmd.OutOrStdout(), "reproduced %s exactly (%d bytes)\n",
			filepath.Base(*want), len(golden))
		return nil
	}
	return cmd
}

// capture runs the gem once and returns the HTML it produced. Anything on
// stderr is an error: a warning from the gem means the capture was made under
// conditions nobody checked, and a golden is only worth what the run that
// produced it is worth.
func capture(lib, tag, in string) ([]byte, error) {
	cmd := exec.Command(rubyBin, "-I", lib, "-e", script, "--", tag, in)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ruby: %v: %s", err, stderr.String())
	}
	if stderr.Len() > 0 {
		return nil, fmt.Errorf("ruby wrote to stderr: %s", stderr.String())
	}
	return stdout.Bytes(), nil
}
