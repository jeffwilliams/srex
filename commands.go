package main

import (
	"fmt"
	"io"
	"os"
	"regexp"
)

type Command interface {
	Do(data io.ReaderAt, start, end int64, match func(start, end int64)) error
}

type RegexpCommand struct {
	regexp *regexp.Regexp
	//label  rune // "name" of the command: x, y, g..
}

func NewRegexpCommand(label rune, re *regexp.Regexp) Command {
	switch label {
	case 'x':
		return &XCommand{RegexpCommand{regexp: re}}
	case 'g':
		return &GCommand{RegexpCommand{regexp: re}}
	default:
		panic(fmt.Sprintf("NewRegexpCommand: called with invalid command rune %c", label))
	}

	return nil
}

// XCommand is like the sam editor's x command: loop over matches of this regexp
type XCommand struct {
	RegexpCommand
}

func (c XCommand) Do(data io.ReaderAt, start, end int64, match func(start, end int64)) error {
	buf, err := readRange(data, start, end)

	if err != nil {
		return err
	}

	matches := c.RegexpCommand.regexp.FindAllSubmatchIndex(buf, -1)
	if matches == nil {
		fmt.Printf("XCommand.Do: no matches\n")
		return nil
	}
	
	for _, locs := range matches {
		fmt.Printf("XCommand.Do: match at %d-%d\n", locs[0], locs[1])
		match(start+int64(locs[0]), start+int64(locs[1]))
	} 
	
	return nil
}

// GCommand is like the sam editor's g command: if the regexp matches the range, output the range, otherwise output no range.
type GCommand struct {
	RegexpCommand
}

func (c GCommand) Do(data io.ReaderAt, start, end int64, match func(start, end int64)) error {
	buf, err := readRange(data, start, end)

	if err != nil {
		fmt.Printf("GCommand.Do(%s): error %v\n", string(buf), err)
		return err
	}

	if c.RegexpCommand.regexp.Match(buf) {
		fmt.Printf("GCommand.Do(%s): match at %d-%d\n", string(buf), start, end)
		match(start, end)
	}

	match(EmptyRange.Start, EmptyRange.End)

	return nil
}

// PrintCommand is like the sam editors p command.
type PrintCommand struct{ Out io.Writer }

func (p PrintCommand) Do(data io.ReaderAt, start, end int64, match func(start, end int64)) error {
	buf, err := readRange(data, start, end)

	if err != nil {
		return err
	}

	if p.Out == nil {
		p.Out = os.Stdout
	}

	p.Out.Write(buf)

	return nil
}
