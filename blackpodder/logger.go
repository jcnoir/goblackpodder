// test.go
package main

import (
	"io"
	"io/ioutil"
	"log"
	"os"
)

type Logger struct {
	Debug   *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
}

func NewLogger(verbose bool) Logger {

	var verboseOut io.Writer

	if verbose {
		verboseOut = os.Stdout
	} else {
		verboseOut = ioutil.Discard
	}

	logger := Logger{
		Debug:   log.New(verboseOut, "TRACE    : ", log.Ldate|log.Ltime|log.Lshortfile),
		Info:    log.New(os.Stdout, "", 0),
		Warning: log.New(os.Stdout, "WARNING : ", 0),
		Error:   log.New(os.Stderr, "ERROR   : ", 0),
	}
	return logger
}
