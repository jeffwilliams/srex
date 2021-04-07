package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Command represents a single stage in the pipeline of commands. It processes
// `data` between `start` and `end` and if it finds a match calls `match` with the
// start and end of the match.
type Command interface {
	Do(data io.ReaderAt, start, end int64, match func(start, end int64)) error
}

type Doner interface {
	Done() error
}

type RegexpCommand struct {
	regexp         *regexp.Regexp
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
	case 'z':
		return &ZCommand{RegexpCommand{regexp: re}, -1}
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

// YCommand is like the sam editor's y command, but instead of omitting the matching part, it is included
// as part of the following match.
type ZCommand struct {
	RegexpCommand
	matchStart int64
}

func (c ZCommand) Do(data io.ReaderAt, start, end int64, match func(start, end int64)) error {
	if emptyRange(start, end) {
		return nil
	}

	c.matchStart = -1

	rdr := c.reader(data, start, end)
	dbg("ZCommand.Do: section reader from %d len %d\n", start, end-start)

	for {
		locs := c.RegexpCommand.regexp.FindReaderSubmatchIndex(rdr)
		if locs == nil {
			break
		}

		dbg("ZCommand.Do: match starting at %d\n", locs[0])
		if c.matchStart >= 0 {
			dbg("ZCommand.Do: match at %d-%d. offset=%d\n", c.matchStart, c.offset()+int64(locs[0]), c.offset())
			match(c.matchStart, c.offset()+int64(locs[0]))
			c.matchStart = int64(locs[0])
		}
		c.matchStart = c.offset() + int64(locs[0])
		c.updateOffset(c.offset() + int64(locs[1]))
	}

	if c.matchStart >= 0 && c.offset() != end {
		match(c.matchStart, end)
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

// PrintLineCommand is like the sam editor's = command.
type PrintLineCommand struct {
	fname string
	out   io.Writer
}

func NewPrintLineCommand(fname string, out io.Writer) *PrintLineCommand {
	return &PrintLineCommand{fname: fname, out: out}
}

func (p *PrintLineCommand) Do(data io.ReaderAt, start, end int64, match func(start, end int64)) error {
	dbg("PrintLineCommand.Do for %d-%d\n", start, end)

	nl := 1
	var (
		err error
		r   rune
	)

	sr := io.NewSectionReader(data, 0, start)
	rdr := bufio.NewReader(sr)

	readAndCount := func() {
		for {
			r, _, err = rdr.ReadRune()
			if err != nil {
				break
			}

			if r == '\n' {
				nl++
			}
		}
	}

	// Re-read data and count the number of lines
	readAndCount()

	if err != io.EOF {
		return err
	}
	p.out.Write([]byte(fmt.Sprintf("%s:%d", p.fname, nl)))

	scnt := nl

	sr = io.NewSectionReader(data, start, end-start)
	rdr.Reset(sr)

	readAndCount()

	if err != io.EOF {
		return err
	}

	if nl != scnt {
		p.out.Write([]byte(fmt.Sprintf(",%d", nl)))
	}
	p.out.Write([]byte("\n"))

	return nil
}

// NCommand only allows ranges in the range [first,last] to pass. Ranges
// are counted starting from 0.
// Syntax:
// 	5  		sixth range
//     5:6 		sixth and seventh ranges
//	5:		sixth range to last
//	0:-2		sixth range to second-last
//     -1		last
type NCommand struct {
	// end == -1 means end is the last possible range.
	// end == -2 means the second last range
	start, end int
	ranges     []Range
	match      func(start, end int64)
}

func NewNCommand(s string) (*NCommand, error) {
	cmd := &NCommand{}
	var err error

	parts := strings.Split(s, ":")
	cmd.start, err = strconv.Atoi(parts[0])
	if err != nil {
		return nil, err
	}

	if len(parts) == 1 {
		// a single number
		cmd.end = cmd.start
	} else {
		if len(parts[1]) == 0 {
			cmd.end = -1
		} else {
			cmd.end, err = strconv.Atoi(parts[1])
			if err != nil {
				return nil, err
			}
		}
	}

	return cmd, err
}

func MustNCommand(s string) *NCommand {
	c, err := NewNCommand(s)
	if err != nil {
		panic(err)
	}
	return c
}

func (p *NCommand) Do(data io.ReaderAt, start, end int64, match func(start, end int64)) error {
	p.saveRange(start, end)
	p.match = match
	return nil
}

func (p *NCommand) saveRange(start, end int64) {
	if p.ranges == nil {
		p.ranges = make([]Range, 0, 20)
	}

	p.ranges = append(p.ranges, Range{start, end})
}

func (p *NCommand) Done() error {
	p.computeActualStart()
	p.computeActualEnd()

	if p.start > p.end {
		// Treat this as the empty set
		return nil
	}

	if p.start < 0 || p.end > len(p.ranges) {
		return nil
	}

	for _, r := range p.ranges[p.start:p.end] {
		p.match(r.Start, r.End)
	}
	return nil
}

func (p *NCommand) computeActualStart() {
	if p.start < 0 {
		p.start = len(p.ranges) + p.start
	}
}

func (p *NCommand) computeActualEnd() {
	if p.end >= 0 {
		p.end += 1
	} else {
		p.end = len(p.ranges) + p.end + 1
	}
}
