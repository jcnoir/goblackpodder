// test.go
package main

import (
	"log"
	"os"
)

type Logger struct {
	Debug   *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
}

func NewLogger() Logger {
	logger := Logger{
		Debug:   log.New(os.Stdout, "TRACE    : ", log.Ldate|log.Ltime|log.Lshortfile),
		Info:    log.New(os.Stdout, "", 0),
		Warning: log.New(os.Stdout, "WARNING : ", log.Ldate|log.Ltime|log.Lshortfile),
		Error:   log.New(os.Stderr, "ERROR   : ", log.Ldate|log.Ltime|log.Lshortfile),
	}
	return logger
}
