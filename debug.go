package main

import "fmt"

var debug = true

func dbg(msg string, args ...interface{}) {
	if debug {
		fmt.Printf(msg, args...)
	}
}
