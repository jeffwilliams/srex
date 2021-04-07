package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"unicode"

	"github.com/ogier/pflag"
)

// Using file seek, in case we are processing a large file

func init() {
	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <file> <commands>\n", os.Args[0])
		fmt.Printf("Apply structural regular expressions to the file, like in sam, and print the result to stdout. Supported commands:\n")
		fmt.Printf("x/pattern/ (looping over match)\n")
		fmt.Printf("y/pattern/ (looping over not match)\n")
		fmt.Printf("z/pattern/ (looping over match plus everything after not including next match)\n")
		fmt.Printf("g/pattern/ (selecting matching objects)\n")
		fmt.Printf("v/pattern/ (selecting non-matching objects)\n")
		fmt.Printf("n[indexes] (select only the ranges with the specified indexes. Valid values:)\n")
		fmt.Printf("   N   a single number selects the range N only. Ranges are counted starting from 0. If N is negative it specifies counts from the last element instead\n")
		fmt.Printf("   N:M  select ranges who's index is >= N and <= M. M may be negative.\n")
		fmt.Printf("   N:     select ranges who's index is >= N\n")
		fmt.Printf("p (print the range. This is the default behaviour. This command is terminal.)\n")
		fmt.Printf("= (print the file and line numbers of ranges. This command is terminal.)\n")
		fmt.Printf("\n")
		fmt.Printf("Commands can be composed into a pipeline of commands like so:")
		fmt.Printf("x/pattern/ g/pattern/ n[5]")

		pflag.PrintDefaults()
	}
}

func main() {
	var err error

	pflag.Parse()
	debug = *optDebug

	dbg("Command line positional arguments after parsing: %#v\n", pflag.Args())

	*optSep, err = replaceEscapes(*optSep)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid escape character in separator\n")
		os.Exit(1)
	}

	if len(pflag.Args()) < 1 {
		fmt.Fprintf(os.Stderr, "The first argument must be a filename\n")
		os.Exit(1)
	}
	fname := pflag.Args()[0]

	if len(pflag.Args()) < 2 {
		fmt.Fprintf(os.Stderr, "There must be a command specified\n")
		os.Exit(1)
	}
	commands := pflag.Args()[1]

	file, err := os.Open(fname)
	if err != nil {
		fmt.Fprintf(os.Stderr, "The first argument must be a filename\n")
		os.Exit(1)
	}

	err = processFile(fname, file, commands, *optSep)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}

func processFile(fname string, file *os.File, commands, sep string) error {
	cmds, err := parseCommands(fname, commands)
	if err != nil {
		return err
	}

	ex := NewExecutor(cmds)
	ex.Sep = sep
	ex.Go(file)

	return nil
}

func lengthOfReaderAt(r io.ReaderAt) (int64, error) {
	if s, ok := r.(io.Seeker); ok {
		return s.Seek(0, io.SeekEnd)
	}

	// Use the readerAtSize function to determine size. First port it.
	return 0, fmt.Errorf("Can't determine length of ReaderAt since it is not a Seeker")
}

type Range struct {
	Start, End int64
}

var EmptyRange = Range{}

func emptyRange(start, end int64) bool {
	return end-start <= 0
}

func (r Range) IsEmpty() bool {
	return r.Start == r.End && r.Start == 0
}

func (r Range) SectionReader(input io.ReaderAt) *io.SectionReader {
	return io.NewSectionReader(input, int64(r.Start), int64(r.End-r.Start))
}

var completeRange = Range{-1, -1}

func parseCommands(fname string, commands string) (result []Command, err error) {
	result = []Command{}
	for _, s := range tokenizeCommands(commands) {
		cmdLabel := []rune(s)[0]
		switch cmdLabel {
		case 'x', 'y', 'g', 'v', 'z':
			if len(s) < 3 {
				err = fmt.Errorf("Command '%s' is malformatted", s)
				return
			}
			var re *regexp.Regexp
			re, err = parseCommandRegexp(s)
			if err != nil {
				return
			}
			cmd := NewRegexpCommand(cmdLabel, re)
			result = append(result, cmd)
		case 'p':
			cmd := NewPrintCommand(os.Stdout, *optSep)
			result = append(result, cmd)
		case '=':
			cmd := NewPrintLineCommand(fname, os.Stdout)
			result = append(result, cmd)
		case 'n':
			p, err := extractArraylikeCommandParameter(s)
			cmd, err := NewNCommand(p)
			if err != nil {
				return nil, err
			}
			result = append(result, cmd)
		default:
			err = fmt.Errorf("Unknown command '%c'", cmdLabel)
			return
		}
	}
	return
}

func tokenizeCommands(commands string) []string {
	var t tokenizer
	return t.tokenize(commands)
}

type tokenizer struct {
	runes []rune
	cmd   bytes.Buffer
	cmds  []string
}

func (t *tokenizer) tokenize(commands string) []string {
	t.runes = []rune(commands)
	t.innerTokenize()
	return t.cmds
}

func (t *tokenizer) innerTokenize() {
	const (
		Default = iota
		WaitingForTerminator
	)

	var state = Default
	var terminator rune
	for _, r := range t.runes {
		switch state {
		case Default:
			if unicode.IsSpace(r) {
				continue
			}
			t.addRuneToCurrentCommand(r)
			switch r {
			case '/':
				state = WaitingForTerminator
				terminator = '/'
			case '[':
				state = WaitingForTerminator
				terminator = ']'
			}
		case WaitingForTerminator:
			t.addRuneToCurrentCommand(r)
			if r == terminator {
				state = Default
				t.addCommand()
			}
		}
	}

	if t.cmd.Len() != 0 {
		t.addCommand()
	}
}

func (t *tokenizer) addRuneToCurrentCommand(r rune) {
	t.cmd.WriteRune(r)
}

func (t *tokenizer) addCommand() {
	t.cmds = append(t.cmds, t.cmd.String())
	t.cmd.Reset()
}

func parseCommandRegexp(command string) (re *regexp.Regexp, err error) {
	reText, err := extractRegexpCommandParameter(command)
	if err != nil {
		return
	}
	re, err = regexp.Compile(reText)
	return
}

func extractRegexpCommandParameter(command string) (param string, err error) {
	return extractCommandParameter(command, '/', '/')
}

func extractArraylikeCommandParameter(command string) (param string, err error) {
	return extractCommandParameter(command, '[', ']')
}

func extractCommandParameter(command string, lmark, rmark rune) (param string, err error) {
	// First char of the command is the command label, then it must be /.../
	if len(command) < 3 {
		err = fmt.Errorf("Command '%s' is malformatted", command)
		return
	}

	if command[1] != byte(lmark) {
		err = fmt.Errorf("Command '%c' must be followed by a forward slash (the complete command is: '%s')",
			command[0], command)
		return
	}

	if command[len(command)-1] != byte(rmark) {
		err = fmt.Errorf("Command '%c' must be terminated by a forward slash (the complete command is: '%s')",
			command[0], command)
		return
	}

	param = command[2 : len(command)-1]
	return
}

func readRange(data io.ReaderAt, start, end int64) (buf []byte, err error) {
	if end < start {
		panic(fmt.Sprintf("readRange: can't read range %d-%d", start, end))
	}
	buf = make([]byte, end-start)
	_, err = data.ReadAt(buf, start)
	if err == io.EOF {
		err = nil
	}

	return
}

/*
// Determine the size of a ReaderAt using a binary search. Given that file
// offsets are no larger than int64, there is an upper limit of 64 iterations
// before the EOF is found.
func readerAtSize(rd io.ReaderAt) (pos int64, err error) {
	defer errs.Recover(&err)

	// Function to check if the given position is at EOF
	buf := make([]byte, 2)
	checkEOF := func(pos int64) int {
		if pos > 0 {
			cnt, err := rd.ReadAt(buf[:2], pos-1)
			errs.Panic(errs.Ignore(err, io.EOF))
			return 1 - cnt // RetVal[Cnt] = {0: +1, 1: 0, 2: -1}
		} else { // Special case where position is zero
			cnt, err := rd.ReadAt(buf[:1], pos-0)
			errs.Panic(errs.Ignore(err, io.EOF))
			return 0 - cnt // RetVal[Cnt] = {0: 0, 1: -1}
		}
	}

	// Obtain the size via binary search O(log n) => 64 iterations
	posMin, posMax := int64(0), int64(math.MaxInt64)
	for posMax >= posMin {
		pos = (posMax + posMin) / 2
		switch checkEOF(pos) {
		case -1: // Below EOF
			posMin = pos + 1
		case 0: // At EOF
			return pos, nil
		case +1: // Above EOF
			posMax = pos - 1
		}
	}
	panic(errs.New("EOF is in a transient state"))
}
*/

func replaceEscapes(s string) (string, error) {
	var b bytes.Buffer

	esc := false
	for _, r := range []rune(s) {
		if !esc {
			if r == '\\' {
				esc = true
			} else {
				b.WriteRune(r)
			}
		} else {
			switch r {
			case 'n':
				b.WriteRune('\n')
			case '\\':
				b.WriteRune('\\')
			default:
				return "", fmt.Errorf("Invalid escape character")
			}
			esc = false
		}
	}
	return b.String(), nil
}
