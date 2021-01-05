package main

import (
	"io"
	"sync"
)

// Executor executes an ordered sequence of commands
type Executor struct {
	commands []Command
	wg       sync.WaitGroup
	// Channels between goroutines in the command pipeline
	chans       []chan Range
	input       io.ReaderAt
	inputLength int64
	Output      io.Writer
}

func NewExecutor(commands []Command) *Executor {
	return &Executor{commands: commands}
}

// Idea: have two pipelines that are connected:
// filtering pipeline -> action pipeline
//
// filtering pipeline is connected using a series of channels that pass successively more restrictive ranges
//
// action pipeline is connected using a series of channels that pass byte buffers for more manipulation
//
// Between the two is a connector that reads a range and converts it to a buffer.

func (ex *Executor) Go(input io.ReaderAt) error {
	err := ex.prepareToGo(input)
	if err != nil {
		return err
	}

	dbg("%d commands\n", len(ex.commands))

	ex.wg.Add(len(ex.commands))

	for stage := 1; stage < len(ex.commands); stage++ {
		go ex.doCommandForStage(stage)
	}

	ex.doCommandForStage(0)

	ex.wg.Wait()

	return nil
}

func (ex *Executor) prepareToGo(input io.ReaderAt) error {
	var err error
	ex.inputLength, err = lengthOfReaderAt(input)
	if err != nil {
		return err
	}

	ex.commands = ex.addPrintCommandIfNeeded(ex.commands)
	ex.input = input

	// Setup a pipeline for the commands
	ex.makeChans(len(ex.commands) - 1)

	return nil
}

func (ex *Executor) doCommandForStage(stage int) {
	defer ex.wg.Done()

	dbg("Starting stage %d\n", stage)

	if stage == 0 {
		// First stage reads from the reader directly
		ex.commands[stage].Do(ex.input, 0, ex.inputLength, ex.writeRangeToChan(ex.firstChan()))
	} else {
		// Later stages read from a pipe
		for rnge := range ex.chans[stage-1] {
			dbg("Stage %d is reading range %d-%d\n", stage, rnge.Start, rnge.End)

			fn := nop
			if stage < len(ex.commands)-1 {
				fn = ex.writeRangeToChan(ex.chans[stage])
			}
			ex.commands[stage].Do(ex.input, rnge.Start, rnge.End, fn)
		}
	}

	if stage < len(ex.chans) {
		close(ex.chans[stage])
	}
}

func (ex *Executor) firstChan() chan Range {
	if len(ex.chans) > 0 {
		return ex.chans[0]
	}
	return nil
}

func (ex *Executor) makeChans(count int) {
	// Setup a pipeline for the commands

	ex.chans = make([]chan Range, count)
	for i := range ex.chans {
		ex.chans[i] = make(chan Range)
	}
}

func (ex Executor) writeRangeToChan(c chan Range) func(start, end int64) {
	if c == nil {
		return nop
	}

	return func(start, end int64) {
		dbg("Stage is sending range %d-%d\n", start, end)
		c <- Range{start, end}
	}
}

func nop(start, end int64) {
}

func (ex Executor) addPrintCommandIfNeeded(commands []Command) (result []Command) {
	add := false

	if len(commands) == 0 {
		add = true
	} else {
		if _, ok := commands[len(commands)-1].(PrintCommand); !ok {
			add = true
		}
	}

	result = commands
	if add {
		result = append(commands, PrintCommand{ex.Output})
	}
	return
}
