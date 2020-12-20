package main

import (
	"bufio"
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

	ex := Executor{cmds}
	ex.Go(file)

	return nil
}

type Executor struct {
	commands []Command
}

type ReaderAt interface {
	ReadAt(p []byte, off int64) (n int, err error)
	Read(p []byte) (n int, err error)
}

// Idea: have two pipelines that are connected:
// filtering pipeline -> action pipeline
//
// filtering pipeline is connected using a series of channels that pass successively more restrictive ranges
//
// action pipeline is connected using a series of channels that pass byte buffers for more manipulation
//
// Between the two is a connector that reads a range and converts it to a buffer.

func (ex Executor) Go(input ReaderAt) {

	ex.commands = ex.addPrintCommandIfNeeded(ex.commands)

	// Need a runereader and
	// also a way to seek
	//bufio.Reader is a RuneReader on a reader
	// io.SectionReader is a reader on a readerat that limits how much to read

	// Setup a pipeline for the commands
	var chans []chan Range
	if len(ex.commands) > 1 {
		chans = make([]chan Range, len(ex.commands)-1)
		for i := range chans {
			chans[i] = make(chan Range)
		}
	}

	fmt.Printf("%d commands\n", len(ex.commands))

	//inputRange is the range of characters that stage i can read
	inputRange := make([]Range, len(ex.commands))

	// Later stages read from a pipe
	for stage := 1; stage < len(ex.commands); stage++ {
		go func(stage int) {
			for rnge := range chans[stage-1] {
				// setup new sectionreader to read starting
				// at what the last range was but also + rnge.start
				srdr := inputRange[stage-1].Subrange(rnge.Start, rnge.End).SectionReader(input)
				rdr := bufio.NewReader(srdr)
				ex.commands[stage].Do(rdr, func(start, end int) {
					chans[stage] <- Range{start, end}
				})
			}
		}(stage)
	}

	// First stage reads from the reader directly

	rdr := bufio.NewReader(input)
	ex.commands[0].Do(rdr, func(start, end int) {
		chans[0] <- Range{start, end}
	})

}

func (ex Executor) addPrintCommandIfNeeded(commands []Command) (result []Command) {
	result = append(commands, PrintCommand{})
	return
}

type Range struct {
	Start, End int
}

func (r Range) IsCompleteRange() bool {
	return r.Start == -1 && r.End == -1
}

func (r Range) Subrange(start, end int) Range {
	if r.IsCompleteRange() {
		return Range{Start: start, End: end}
	}
	return Range{Start: r.Start + start, End: r.End + end}
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
	fmt.Printf("re text: %s\n", reText)

	re, err = regexp.Compile(command[1 : len(command)-1])
	return
}

type Command interface {
	Do(rnge io.RuneReader, match func(start, end int)) error
}

type RegexpCommand struct {
	regexp *regexp.Regexp
	//label  rune // "name" of the command: x, y, g..
}

func NewRegexpCommand(label rune, re *regexp.Regexp) Command {
	switch label {
	case 'x':
		return &XCommand{RegexpCommand{regexp: re}}
	}
	return nil
}

type XCommand struct {
	RegexpCommand
}

// rnge should be a reader that will only read as much as the range that the command
// should operate on. This can be achieved using a LimitReader
// Any matches found will be passed to the match command
func (c XCommand) Do(rnge io.RuneReader, match func(start, end int)) error {
	locs := c.RegexpCommand.regexp.FindReaderSubmatchIndex(rnge)
	if locs == nil {
		return nil
	}

	match(locs[0], locs[1])
	return nil
}

type GCommand struct {
	RegexpCommand
}

// rnge should be a reader that will only read as much as the range that the command
// should operate on. This can be achieved using a LimitReader
// Any matches found will be passed to the match command
func (c GCommand) Do(rnge io.RuneReader, match func(start, end int)) error {
	/*
		if c.RegexpCommand.regexp.Match(rnge) {
			//TODO:  Retuirn the full range. We should just pass a range here instead of the reader.
			return nil
		}

		match(locs[0], locs[1])
	*/
	return nil
}

type PrintCommand struct{ Out io.Writer }

func (p PrintCommand) Do(rnge io.RuneReader, match func(start, end int)) error {
	reader, ok := rnge.(io.Reader)
	if !ok {
		return fmt.Errorf("PrintCommand.Do was passed something that is not an io.Reader")
	}

	if p.Out == nil {
		p.Out = os.Stdout
	}

	io.Copy(p.Out, reader)

	return nil
}
