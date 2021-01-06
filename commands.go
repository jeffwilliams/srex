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
	case 'y':
		return &YCommand{RegexpCommand{regexp: re}}
	case 'v':
		return &VCommand{RegexpCommand{regexp: re}}
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
		dbg("XCommand.Do: no matches\n")
		return nil
	}

	for _, locs := range matches {
		dbg("XCommand.Do: match at %d-%d\n", locs[0], locs[1])
		match(start+int64(locs[0]), start+int64(locs[1]))
	}

	return nil
}

// YCommand is like the sam editor's y command: loop over strings before, between, and after matches of this regexp
type YCommand struct {
	RegexpCommand
}

func (c YCommand) Do(data io.ReaderAt, start, end int64, match func(start, end int64)) error {
	buf, err := readRange(data, start, end)

	if err != nil {
		return err
	}

	matches := c.RegexpCommand.regexp.FindAllSubmatchIndex(buf, -1)
	if matches == nil {
		dbg("YCommand.Do: no matches\n")
		match(start, end)
		return nil
	}
	
	dbg("YCommand.Do: %d matches\n", len(matches))

	lastIndex := start
	for _, locs := range matches {
		dbg("YCommand.Do: re match at %d-%d\n", locs[0], locs[1])
		dbg("YCommand.Do: sending match %d-%d\n", lastIndex, lastIndex+int64(locs[0]))

		match(start+lastIndex, start+int64(locs[0]))
		lastIndex = int64(locs[1])
	}

	if lastIndex != end {
		match(lastIndex, end)
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
		dbg("GCommand.Do(%s): error %v\n", string(buf), err)
		return err
	}

	if c.RegexpCommand.regexp.Match(buf) {
		dbg("GCommand.Do(%s): match at %d-%d\n", string(buf), start, end)
		match(start, end)
		return nil
	}

	match(EmptyRange.Start, EmptyRange.End)

	return nil
}

// VCommand is like the sam editor's y command: if the regexp doesn't match the range, output the range, otherwise output no range.
type VCommand struct {
	RegexpCommand
}

func (c VCommand) Do(data io.ReaderAt, start, end int64, match func(start, end int64)) error {
	buf, err := readRange(data, start, end)
	
	if err != nil {
		dbg("VCommand.Do(%s): error %v\n", string(buf), err)
		return err
	}
	
	if c.RegexpCommand.regexp.Match(buf) {
		dbg("VCommand.Do(%s): match at %d-%d\n", string(buf), start, end)
		match(EmptyRange.Start, EmptyRange.End)
		return nil
	}
	
	match(start, end)
	
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
