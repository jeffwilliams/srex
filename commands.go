package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
)

// Command represents a single stage in the pipeline of commands. It processes
// `data` between `start` and `end` and if it finds a match calls `match` with the
// start and end of the match.
type Command interface {
	Do(data io.ReaderAt, start, end int64, match func(start, end int64)) error
}

type RegexpCommand struct {
	regexp *regexp.Regexp
	secRdr         *io.SectionReader
	rdr            *bufio.Reader
	start, _offset int64
}

// NewRegexpCommand returns a new Command that uses the specified Regexp.
// The `label` chooses which Command to build; i.e. 'x' creates an XCommand.
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

func (r *RegexpCommand) reader(data io.ReaderAt, start, end int64) io.RuneReader {
	r.secRdr = io.NewSectionReader(data, start, end-start)
	r.rdr = bufio.NewReader(r.secRdr)
	r.start = start
	r._offset = start
	return r.rdr
}

func (r *RegexpCommand) offset() int64 {
	return r._offset
}

func (r *RegexpCommand) updateOffset(o int64) {
	r._offset = o
	r.secRdr.Seek(r._offset-r.start, io.SeekStart)
	r.rdr.Reset(r.secRdr)
}

// XCommand is like the sam editor's x command: loop over matches of this regexp
type XCommand struct {
	RegexpCommand
}

func (c XCommand) Do(data io.ReaderAt, start, end int64, match func(start, end int64)) error {
	if emptyRange(start, end) {
		return nil
	}

	rdr := c.reader(data, start, end)
	dbg("XCommand.Do: section reader from %d len %d\n", start, end-start)

	for {
		locs := c.RegexpCommand.regexp.FindReaderSubmatchIndex(rdr)
		if locs == nil {
			break
		}

		dbg("XCommand.Do: match at %d-%d\n", locs[0], locs[1])
		match(c.offset()+int64(locs[0]), c.offset()+int64(locs[1]))

		c.updateOffset(c.offset() + int64(locs[1]))
	}

	return nil
}

// YCommand is like the sam editor's y command: loop over strings before, between, and after matches of this regexp
type YCommand struct {
	RegexpCommand
}

func (c YCommand) Do(data io.ReaderAt, start, end int64, match func(start, end int64)) error {
	if emptyRange(start, end) {
		return nil
	}

	rdr := c.reader(data, start, end)

	dbg("YCommand.Do: section reader from %d len %d\n", start, end-start)

	for {
		locs := c.RegexpCommand.regexp.FindReaderSubmatchIndex(rdr)
		if locs == nil {
			break
		}

		dbg("YCommand.Do: re match at %d-%d\n", locs[0], locs[1])
		dbg("YCommand.Do: sending match %d-%d\n", c.offset(), c.offset()+int64(locs[0]))

		match(c.offset(), c.offset()+int64(locs[0]))

		c.updateOffset(c.offset() + int64(locs[1]))
	}

	if c.offset() != end {
		match(c.offset(), end)
	}

	return nil
}

// GCommand is like the sam editor's g command: if the regexp matches the range, output the range, otherwise output no range.
type GCommand struct {
	RegexpCommand
}

func (c GCommand) Do(data io.ReaderAt, start, end int64, match func(start, end int64)) error {
	if emptyRange(start, end) {
		return nil
	}

	rdr := c.reader(data, start, end)
	dbg("GCommand.Do: section reader from %d len %d\n", start, end-start)

	if c.RegexpCommand.regexp.MatchReader(rdr) {
		dbg("GCommand.Do: match\n")
		match(start, end)
		return nil
	}

	return nil
}

// VCommand is like the sam editor's y command: if the regexp doesn't match the range, output the range, otherwise output no range.
type VCommand struct {
	RegexpCommand
}

func (c VCommand) Do(data io.ReaderAt, start, end int64, match func(start, end int64)) error {
	if emptyRange(start, end) {
		return nil
	}

	rdr := c.reader(data, start, end)
	dbg("GCommand.Do: section reader from %d len %d\n", start, end-start)

	if c.RegexpCommand.regexp.MatchReader(rdr) {
		dbg("GCommand.Do: match\n")
		return nil
	}

	match(start, end)

	return nil

}

// PrintCommand is like the sam editor's p command.
type PrintCommand struct {
	out      io.Writer
	sep      []byte
	printSep bool
}

func (p *PrintCommand) Do(data io.ReaderAt, start, end int64, match func(start, end int64)) error {
	buf, err := readRange(data, start, end)

	dbg("PrintCommand.Do(%s)\n", string(buf))

	if err != nil {
		return err
	}

	if p.out == nil {
		p.out = os.Stdout
	}

	if p.printSep && len(p.sep) > 0 {
		p.out.Write(p.sep)
	}

	p.out.Write(buf)

	p.printSep = true

	return nil
}

// NewPrintCommand returns a new PrintCommand that writes to `out` and prints the separator `sep` between each match.
func NewPrintCommand(out io.Writer, sep string) *PrintCommand {
	return &PrintCommand{out: out, sep: []byte(sep)}
}
