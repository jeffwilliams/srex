package main

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

func TestLengthOfReader(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty",
			input: "",
		},
		{
			name:  "single",
			input: "l",
		},
		{
			name:  "normal",
			input: "snarfle",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buf := []byte(tc.input)
			rdr := bytes.NewReader(buf)

			l, err := lengthOfReaderAt(rdr)
			if err != nil {
				t.Fatalf("Error getting length of reader: '%v'", err)
			}

			if l != int64(len(buf)) {
				t.Fatalf("Actual '%d' does not match expected '%d'", l, len(buf))
			}
		})
	}

}
func TestPrintCommand(t *testing.T) {

	buf := []byte("test!")
	rdr := bytes.NewReader(buf)

	var out bytes.Buffer
	p := PrintCommand{&out}

	l, err := lengthOfReaderAt(rdr)
	if err != nil {
		t.Fatalf("Error getting length of reader: '%v'", err)
	}

	p.Do(rdr, 0, l, func(start, end int64) {})

	if out.String() != "test!" {
		t.Fatalf("Actual does not match expected: '%s'", out.String())
	}

}

func TestXCommand(t *testing.T) {
	var c XCommand
	var err error

	tests := []struct {
		name     string
		input    string
		regex    string
		expected Range
		failed   bool
	}{
		{
			name:     "simple",
			input:    "line1\ntest",
			regex:    "ine",
			expected: Range{1, 4},
		},
		{
			name:   "nonmatch",
			input:  "loom1\ntest",
			regex:  "ine",
			failed: true,
		},
		{
			name:     "multiline",
			input:    "loom1\ntest",
			regex:    "1ab",
			expected: Range{4, 7},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c.regexp, err = regexp.Compile(tc.regex)
			if err != nil {
				t.Fatalf("Error making regex: %s\n", err)
			}

			buf := []byte(tc.input)
			rdr := bytes.NewReader(buf)

			l, err := lengthOfReaderAt(rdr)
			if err != nil {
				t.Fatalf("Error getting length of reader: '%v'", err)
			}

			c.Do(rdr, 0, l, func(start, end int64) {
				if tc.failed {
					t.Fatalf("Do called when the match failed\n")
				}
				if start != tc.expected.Start || end != tc.expected.End {
					t.Fatalf("start and end does not match expected: %d, %d\n", start, end)
				}
			})
		})
	}
}

func TestExecutor(t *testing.T) {
	output := &bytes.Buffer{}

	tests := []struct {
		name     string
		input    string
		cmds     []Command
		expected string
	}{
		{
			name:     "simple",
			input:    "line1\ntest",
			cmds:     []Command{NewRegexpCommand('x', regexp.MustCompile("ine")), PrintCommand{output}},
			expected: "ine",
		},
		{
			name:     "simple noprint",
			input:    "line1\ntest",
			cmds:     []Command{NewRegexpCommand('x', regexp.MustCompile("ine"))},
			expected: "ine",
		},
		{
			name:  "x matches multiple lines",
			input: "line1\ntest\nline2",
			cmds: []Command{
				NewRegexpCommand('x', regexp.MustCompile(".*line.*")),
				PrintCommand{output}},
			expected: "line1line2",
		},
		{
			name:  "x then g",
			input: "line1\nline2\nline3",
			cmds: []Command{
				NewRegexpCommand('x', regexp.MustCompile(".*line.*")),
				NewRegexpCommand('g', regexp.MustCompile("1|3")),
				PrintCommand{output}},
			expected: "line1line3",
		},
		{
			name:  "x then x",
			input: "line1\nline2\nline3",
			cmds: []Command{
				NewRegexpCommand('x', regexp.MustCompile(".*line.*")),
				NewRegexpCommand('x', regexp.MustCompile("1|3")),
				PrintCommand{output}},
			expected: "13",
		},
		{
			name:  "no commands",
			input: "line1\nline2\nline3",
			cmds: []Command{},
			expected: "line1\nline2\nline3",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			buf := strings.NewReader(tc.input)

			output.Reset()
			ex := NewExecutor(tc.cmds)
			ex.Output = output
			ex.Go(buf)

			s := output.String()
			t.Logf("Expected '%s' and got '%s'", tc.expected, s)
			t.Logf("Expected '%s' and got '%s'", tc.expected, s)
			if s != tc.expected {
				t.Fatalf("Expected '%s' but got '%s'", tc.expected, s)
			}

		})
	}
}
