package main

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/ogier/pflag"
)

// Structural regular expressions, like in sam.
// Using file seek, in case we are processing a large file
// support:
//		x// (looping over match)
//		y// (looping over not match)
//		g// (selecting matching objects)
//		v// (selecting non-matching objects)

// Usage:
// <file> <commands>...

// To implement:
// I think we can make this work on a stream. We can parse the regex twice:
//	- Once the normal way to make the Regexp we use to parse the text
//	- Once using the syntax/ package to get the Prog that we can use to find the Prefix() and
//		StartCond that can be used to determine whether we even need to start matching in the stream yet.
//
// With that we can read from the stream and keep track of if we have seen the prefix or startcond yet.
// If we haven't we can just drop or pass along those characters (as appropriate) until we see it, and
// at that point start matching the regex. While we are matching the regex we need to store the
// bytes read in memory so that we can index back into it once the match is complete.
//
// Now, will this work for y//

func main() {

	pflag.Parse()

	if len(pflag.Args()) < 1 {
		fmt.Fprintf(os.Stderr, "The first argument must be a filename")
		os.Exit(1)
	}
	fname := pflag.Args()[0]

	if len(pflag.Args()) < 2 {
		fmt.Fprintf(os.Stderr, "There must be a command specified")
		os.Exit(1)
	}
	commands := pflag.Args()[1]

	file, err := os.Open(fname)
	if err != nil {
		fmt.Fprintf(os.Stderr, "The first argument must be a filename")
		os.Exit(1)
	}

	processFile(file, commands)
}

func processFile(file *os.File, commands string) error {
	cmds, err := parseCommands(commands)
	if err != nil {
		return err
	}

	ex := NewExecutor(cmds)
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

func (r Range) IsEmpty() bool {
	return r.Start == r.End && r.Start == 0
}

func (r Range) SectionReader(input io.ReaderAt) *io.SectionReader {
	return io.NewSectionReader(input, int64(r.Start), int64(r.End-r.Start))
}

var completeRange = Range{-1, -1}

func parseCommands(commands string) (result []Command, err error) {
	result = []Command{}
	for _, s := range strings.Fields(commands) {
		cmdLabel := []rune(s)[0]
		switch cmdLabel {
		case 'x', 'y', 'g':
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
		}
	}
	return
}

func parseCommandRegexp(command string) (re *regexp.Regexp, err error) {
	// First char of the command is the command label, then it must be /.../
	if len(command) < 3 {
		err = fmt.Errorf("Command '%s' is malformatted", command)
		return
	}

	if command[1] != '/' {
		err = fmt.Errorf("Command '%c' must be followed by a forward slash", command[0])
		return
	}

	if command[len(command)-1] != '/' {
		err = fmt.Errorf("Command '%c' must be terminated by a forward slash", command[0])
		return
	}

	reText := command[1 : len(command)-1]
	re, err = regexp.Compile(reText)
	return
}

func readRange(data io.ReaderAt, start, end int64) (buf []byte, err error) {
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