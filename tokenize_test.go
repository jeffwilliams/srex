package main

import (
	"testing"
)

func TestTokenizeCommands(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output []string
	}{
		{
			name:   "empty",
			input:  "",
			output: []string{},
		},
		{
			name:   "single",
			input:  "x/test/",
			output: []string{"x/test/"},
		},
		{
			name:   "double",
			input:  "x/test/ y/blort/",
			output: []string{"x/test/", "y/blort/"},
		},		
		{
			name:   "x, y then p",
			input:  "x/test/ y/blort/ p",
			output: []string{"x/test/", "y/blort/", "p"},
		},
		{
			name:   "no space between",
			input:  "x/test/y/blort/",
			output: []string{"x/test/", "y/blort/"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			cmds := tokenizeCommands(tc.input)

			if len(cmds) != len(tc.output) {
				t.Fatalf("Expected %#v but got %#v'", tc.output, cmds)
			}

			for i, c := range tc.output {
				if c != cmds[i] {
					t.Fatalf("Expected %#v but got %#v'", tc.output, cmds)
				}
			}
		})
	}

}
