package main

import "fmt"

var debug = false

func dbg(msg string, args ...interface{}) {
	if debug {
		fmt.Printf(msg, args...)
	}
}
