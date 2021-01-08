package main

import (
	"testing"
)

func TestReplaceEscapes(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output string
	}{
		{
			name:   "empty",
			input:  "",
			output: "",
		},
		{
			name:   "noescapes",
			input:  "test",
			output: "test",
		},
		{
			name:   "newline at end",
			input:  `line\n`,
			output: "line\n",
		},
		{
			name:   "newline at middle",
			input:  `line\nline`,
			output: "line\nline",
		},
		{
			name:   "backslash",
			input:  `line\\a`,
			output: `line\a`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o, err := replaceEscapes(tc.input)
			if err != nil {
				t.Fatalf("Error in escape: %v", err)
			}
			
			if o != tc.output {
				t.Fatalf("Actual '%s' does not match expected '%s'", o, tc.output)
			}
		})
	}

}
