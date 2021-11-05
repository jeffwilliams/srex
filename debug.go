package main

import (
	"fmt"
	"os"
)

var debug = false

func dbg(msg string, args ...interface{}) {
	if debug {
		fmt.Fprintf(os.Stderr, msg, args...)
	}
}
