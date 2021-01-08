package main

import (
	"github.com/ogier/pflag"
)

var (
	optDebug = pflag.BoolP("debug", "d", false, "Print debug info")
	optSep   = pflag.StringP("separator", "s", "", "String to print between matches")
)
