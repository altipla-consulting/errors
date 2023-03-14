package errors

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
)

var (
	New    = errors.New
	Is     = errors.Is
	As     = errors.As
	Unwrap = errors.Unwrap
)

type altiplaError struct {
	cause error
	stack []uintptr
}

func (e *altiplaError) Error() string {
	return e.cause.Error()
}

func (e *altiplaError) Cause() error {
	return e.cause
}

func (e *altiplaError) StackTrace() []uintptr {
	return e.stack
}

func (e *altiplaError) Unwrap() error {
	return e.cause
}

func (e *altiplaError) writeStackTrace(w io.Writer) {
	fmt.Fprintf(w, "%s\n\n", e.cause.Error())

	for _, frame := range Frames(e) {
		fmt.Fprintf(w, "%s\n", frame.Function)
		fmt.Fprintf(w, "\t%s:%d\n", frame.File, frame.Line)
	}
}

// A Frame represents a Frame in an altipla callstack.
type Frame struct {
	File     string
	Function string
	Line     int
}

// Frames extracts all frames from an altipla error. If err is not an altipla error,
// nil is returned.
func Frames(err error) []Frame {
	e, ok := err.(*altiplaError)
	if !ok {
		return nil
	}

	frames := make([]Frame, 0, 8)
	iter := runtime.CallersFrames(e.stack)
	for {
		frame, ok := iter.Next()
		if !ok {
			break
		}

		frames = append(frames, Frame{
			File:     frame.File,
			Function: frame.Function,
			Line:     frame.Line,
		})
	}
	return frames
}

// Details returns the stacktrace in a succinct format to print a one-line error.
func Details(err error) string {
	e, ok := err.(*altiplaError)
	if !ok {
		return "{" + err.Error() + "}"
	}

	result := []string{
		"{" + e.cause.Error() + "}",
	}
	for _, frame := range Frames(e) {
		result = append(result, fmt.Sprintf("{%s:%d:%s}", frame.File, frame.Line, frame.Function))
	}
	return strings.Join(result, " ")
}

func internalWrapf(err error) error {
	if _, ok := err.(*altiplaError); ok {
		return err
	}

	var buffer [256]uintptr
	// 0 is the frame of Callers, 1 is us, 2 is the public wrapper, 3 is its caller.
	n := runtime.Callers(3, buffer[:])
	frames := make([]uintptr, n)
	copy(frames, buffer[:n])

	return &altiplaError{
		cause: err,
		stack: frames,
	}
}

// Errorf creates a new error with a reason and a stacktrace.
//
// Use Errorf in places where you would otherwise return an error using
// fmt.Errorf or errors.New.
//
// Note that the result of Errorf includes a stacktrace. This means
// that Errorf is not suitable for storing in global variables. For
// such errors, keep using errors.New.
func Errorf(format string, a ...interface{}) error {
	return internalWrapf(fmt.Errorf(format, a...))
}

// Trace annotates an error with a stacktrace.
//
// Use Trace in places where you would otherwise return an error directly. If
// the error passed to Trace is nil, Trace will also return nil. This makes it
// safe to use in one-line return statements.
func Trace(err error) error {
	if err == nil {
		return nil
	}
	return internalWrapf(err)
}

// Cause extracts the cause error of an altipla error. If err is not an altipla
// error, err itself is returned.
func Cause(err error) error {
	if e, ok := err.(*altiplaError); ok {
		return e.cause
	}
	return err
}

// Recover recovers from a panic in a defer. If there is no panic, Recover()
// returns nil. To use, call error.Recover(recover()) and compare the result to nil.
func Recover(p interface{}) error {
	if p == nil {
		return nil
	}
	if err, ok := p.(error); ok {
		return Trace(err)
	}
	return internalWrapf(fmt.Errorf("panic: %v", p))
}

// LogFields returns fields to properly log an error.
func LogFields(err error) log.Fields {
	return log.Fields{
		"error":   err.Error(),
		"details": Details(err),
	}
}

// Stack returns the stacktrace of an error.
func Stack(err error) string {
	e, ok := err.(*altiplaError)
	if !ok {
		return err.Error()
	}

	var buf bytes.Buffer
	e.writeStackTrace(&buf)
	return buf.String()
}
