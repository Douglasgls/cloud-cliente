package logger

import (
	"fmt"
	"io"
	"os"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelError
)

type Logger struct {
	out   io.Writer
	errOut io.Writer
	debug bool
}

func New(debug bool) *Logger {
	return &Logger{
		out:   os.Stdout,
		errOut: os.Stderr,
		debug: debug,
	}
}

func NewWithWriters(out, errOut io.Writer, debug bool) *Logger {
	return &Logger{
		out:   out,
		errOut: errOut,
		debug: debug,
	}
}

func (l *Logger) Info(msg string, args ...interface{}) {
	if len(args) > 0 {
		fmt.Fprintf(l.out, msg+"\n", args...)
	} else {
		fmt.Fprintln(l.out, msg)
	}
}

func (l *Logger) Error(msg string, args ...interface{}) {
	if len(args) > 0 {
		fmt.Fprintf(l.errOut, msg+"\n", args...)
	} else {
		fmt.Fprintln(l.errOut, msg)
	}
}

func (l *Logger) Debug(msg string, args ...interface{}) {
	if !l.debug {
		return
	}
	prefix := "[DEBUG] "
	if len(args) > 0 {
		fmt.Fprintf(l.out, prefix+msg+"\n", args...)
	} else {
		fmt.Fprintln(l.out, prefix+msg)
	}
}
