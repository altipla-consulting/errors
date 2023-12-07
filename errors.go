package errors

import (
	"bytes"
	stderrors "errors"
	"fmt"
	"log/slog"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
)

// New calls errors.New.
func New(text string) error {
	return stderrors.New(text)
}

// Is calls errors.Is.
func Is(err, target error) bool {
	return stderrors.Is(err, target)
}

// As calls errors.As.
func As(err error, target any) bool {
	return stderrors.As(err, target)
}

// Unwrap calls errors.Unwrap.
func Unwrap(err error) error {
	return stderrors.Unwrap(err)
}

type altiplaError struct {
	cause  error
	stack  []uintptr
	frames []Frame
}

func (e *altiplaError) Error() string {
	return e.cause.Error()
}

func (e *altiplaError) Unwrap() error {
	return e.cause
}

func (e *altiplaError) Cause() error {
	return e.cause
}

// StackTrace returns the stack pointers without calling runtime.CallersFrames.
// It implements an interface that Sentry needs to extract the call.
func (e *altiplaError) StackTrace() []uintptr {
	return e.stack
}

// Frame stores information about a call stack frame.
type Frame struct {
	File     string
	Function string
	Line     int
}

// Frames extracts all frames from the first altipla error of the chain.
func Frames(err error) []Frame {
	e := unwrapPrev(err)
	if e == nil {
		return nil
	}
	return e.frames
}

// Details returns the stacktrace in a succinct format to print a one-line error.
func Details(err error) string {
	result := []string{
		"{" + err.Error() + "}",
	}
	for _, frame := range Frames(err) {
		result = append(result, fmt.Sprintf("{%s:%d:%s}", frame.File, frame.Line, frame.Function))
	}
	return strings.Join(result, " ")
}

func internalWrap(err error) error {
	if prev := unwrapPrev(err); prev != nil {
		return &altiplaError{
			cause:  err,
			stack:  prev.stack,
			frames: prev.frames,
		}
	}

	var buffer [256]uintptr
	// 0 is the frame of Callers, 1 is us, 2 is the public wrapper, 3 is its caller.
	n := runtime.Callers(3, buffer[:])
	stack := make([]uintptr, n)
	copy(stack, buffer[:n])

	var frames []Frame
	iter := runtime.CallersFrames(stack)
	for {
		frame, more := iter.Next()
		frames = append(frames, Frame{
			File:     frame.File,
			Function: frame.Function,
			Line:     frame.Line,
		})
		if !more {
			break
		}
	}
	return &altiplaError{
		cause:  err,
		stack:  stack,
		frames: frames,
	}
}

func unwrapPrev(err error) *altiplaError {
	for err != nil {
		e, ok := err.(*altiplaError)
		if ok {
			return e
		}
		err = Unwrap(err)
	}
	return nil
}

// Errorf creates a new error with a formatted message.
//
// Use Errorf in places where you would otherwise return an error using
// fmt.Errorf.
//
// Note that the result of Errorf includes a stacktrace. This means
// that Errorf is not suitable for storing in global variables. For
// such errors, keep using errors.New.
func Errorf(format string, a ...interface{}) error {
	return internalWrap(fmt.Errorf(format, a...))
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
	return internalWrap(err)
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
	return internalWrap(fmt.Errorf("panic: %v", p))
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
	e := unwrapPrev(err)
	if e == nil {
		return err.Error()
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%s\n\n", err.Error())
	for _, frame := range Frames(e) {
		fmt.Fprintf(&buf, "%s\n", frame.Function)
		fmt.Fprintf(&buf, "\t%s:%d\n", frame.File, frame.Line)
	}
	return buf.String()
}

// LogValue returns the standard log value to emit the error in a structured way.
func LogValue(err error) slog.Value {
	return slog.GroupValue(slog.String("error", err.Error()), slog.String("details", Details(err)))
}
