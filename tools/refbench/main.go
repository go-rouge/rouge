// Copyright (c) the go-rouge/rouge authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Command refbench times go-rouge's lexers against the reference, Ruby Rouge,
// on the same input, single core, in interleaved rounds.
//
//	refbench --lib <rouge gem lib dir> [--kib 64] [--rounds 5]
//
// Per round and language it runs Ruby once (a fresh process that warms up, then
// times the best of three lexes inside the process, so interpreter start-up and
// gem loading are excluded) and then go-rouge in-process (warm-up, best of
// five). Alternating the two within a round means a load spike hits both. It
// refuses to run when the one-minute load average is above --max-load, because
// a time taken on a busy machine measures the machine.
//
// The input for a language is its committed corpus file(s) in testdata/,
// repeated to at least --kib KiB, exactly as BenchmarkLexJuliaRPython builds it.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-rouge/rouge"
	"github.com/spf13/cobra"
)

var osExit = os.Exit

// corpus names the testdata files each language is built from.
var corpus = []struct {
	tag   string
	files []string
}{
	{"python", []string{"python.py", "python2.py", "python3.py", "python4.py", "python5.py"}},
	{"julia", []string{"julia.jl"}},
	{"r", []string{"r.r"}},
}

// script lexes the file in ARGV[1] with lexer ARGV[0]: one warm-up, then three
// timed lexes whose best time in nanoseconds is printed. The token stream is
// forced (lex is lazy) and counted so the work cannot be skipped.
const script = `
require "rouge"
lexer = Rouge::Lexer.find(ARGV[0])
abort("no such lexer: #{ARGV[0]}") if lexer.nil?
src = File.read(ARGV[1], encoding: "UTF-8")
lexer.lex(src).count
best = nil
3.times do
  t = Process.clock_gettime(Process::CLOCK_MONOTONIC, :nanosecond)
  lexer.lex(src).count
  d = Process.clock_gettime(Process::CLOCK_MONOTONIC, :nanosecond) - t
  best = d if best.nil? || d < best
end
puts best
`

func main() {
	if err := RootCmd().Execute(); err != nil {
		osExit(1)
	}
}

// RootCmd builds the command.
func RootCmd() *cobra.Command {
	var lib, testdata string
	var kib, rounds int
	var maxLoad float64
	cmd := &cobra.Command{
		Use:          "refbench",
		Short:        "Time go-rouge against Ruby Rouge on the same input",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(cmd, lib, testdata, kib, rounds, maxLoad)
		},
	}
	cmd.Flags().StringVar(&lib, "lib", "", "the gem's lib directory (the one holding rouge.rb)")
	cmd.Flags().StringVar(&testdata, "testdata", "testdata", "directory holding the corpus files")
	cmd.Flags().IntVar(&kib, "kib", 64, "input size in KiB")
	cmd.Flags().IntVar(&rounds, "rounds", 5, "interleaved rounds")
	cmd.Flags().Float64Var(&maxLoad, "max-load", 8, "refuse to run above this one-minute load average")
	cobra.CheckErr(cmd.MarkFlagRequired("lib"))
	return cmd
}

func run(cmd *cobra.Command, lib, testdata string, kib, rounds int, maxLoad float64) error {
	runtime.GOMAXPROCS(1)
	load, err := loadAverage()
	if err != nil {
		return err
	}
	if load > maxLoad {
		return fmt.Errorf("one-minute load %.2f is above %.2f: a timing taken now measures the machine; deferred", load, maxLoad)
	}
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "1-min load %.2f, %d KiB, %d interleaved rounds, GOMAXPROCS=1\n", load, kib, rounds)
	fmt.Fprintf(out, "%-8s %12s %12s %8s\n", "lexer", "rougify ms", "go-rouge ms", "ratio")
	dir, err := os.MkdirTemp("", "refbench")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	for _, c := range corpus {
		src, err := build(testdata, c.files, kib<<10)
		if err != nil {
			return err
		}
		in := filepath.Join(dir, c.tag+".in")
		if err := os.WriteFile(in, []byte(src), 0o644); err != nil {
			return err
		}
		lx := rouge.FindLexer(c.tag)
		var ruby, goT []time.Duration
		for r := 0; r < rounds; r++ {
			d, err := rubyTime(lib, c.tag, in)
			if err != nil {
				return err
			}
			ruby = append(ruby, d)
			goT = append(goT, goTime(lx, src))
		}
		rm, gm := median(ruby), median(goT)
		fmt.Fprintf(out, "%-8s %12.2f %12.2f %7.2fx\n", c.tag,
			float64(rm)/1e6, float64(gm)/1e6, float64(gm)/float64(rm))
	}
	return nil
}

// build repeats the files until the result holds at least n bytes.
func build(testdata string, files []string, n int) (string, error) {
	var b strings.Builder
	for b.Len() < n {
		for _, f := range files {
			src, err := os.ReadFile(filepath.Join(testdata, f))
			if err != nil {
				return "", err
			}
			b.Write(src)
		}
	}
	return b.String(), nil
}

// rubyTime runs the Ruby driver once and returns the best lex time it reports.
func rubyTime(lib, tag, in string) (time.Duration, error) {
	c := exec.Command("ruby", "-I", lib, "-e", script, "--", tag, in)
	var stderr strings.Builder
	c.Stderr = &stderr
	o, err := c.Output()
	if err != nil {
		return 0, fmt.Errorf("ruby: %v: %s", err, stderr.String())
	}
	ns, err := strconv.ParseInt(strings.TrimSpace(string(o)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("ruby printed %q: %v", o, err)
	}
	return time.Duration(ns), nil
}

// goTime is one warm-up lex, then the best of five.
func goTime(lx rouge.Lexer, src string) time.Duration {
	lx.Lex(src)
	best := time.Duration(1<<63 - 1)
	for i := 0; i < 5; i++ {
		runtime.GC()
		t := time.Now()
		lx.Lex(src)
		if d := time.Since(t); d < best {
			best = d
		}
	}
	return best
}

func median(ds []time.Duration) time.Duration {
	s := append([]time.Duration(nil), ds...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	return s[len(s)/2]
}

// loadAverage returns the one-minute load average, from `sysctl -n vm.loadavg`
// on macOS and /proc/loadavg elsewhere.
func loadAverage() (float64, error) {
	if b, err := os.ReadFile("/proc/loadavg"); err == nil {
		return strconv.ParseFloat(strings.Fields(string(b))[0], 64)
	}
	o, err := exec.Command("sysctl", "-n", "vm.loadavg").Output()
	if err != nil {
		return 0, err
	}
	f := strings.Fields(strings.Trim(string(o), "{} \n"))
	if len(f) == 0 {
		return 0, fmt.Errorf("unreadable load average %q", o)
	}
	return strconv.ParseFloat(f[0], 64)
}
