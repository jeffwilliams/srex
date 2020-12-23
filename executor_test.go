package main

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

func TestPrintCommand(t *testing.T) {

	buf := bytes.NewBuffer([]byte("test!"))

	var out bytes.Buffer
	p := PrintCommand{&out}
	p.Do(buf, func(start, end int) {})

	if out.String() != "test!" {
		t.Fatalf("Actual does not match expected: '%s'\n", out.String())
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

			buf := bytes.NewBuffer([]byte(tc.input))

			c.Do(buf, func(start, end int) {
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
			name:  "simple",
			input: "line1\ntest",
			cmds:  []Command{NewRegexpCommand('x', regexp.MustCompile("ine")), PrintCommand{output}},
			expected: "ine",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			buf := strings.NewReader(tc.input)

			output.Reset()
			ex := Executor{commands: tc.cmds}
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
