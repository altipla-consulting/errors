package errors

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWrappingNativeStackedErrors(t *testing.T) {
	err := Trace(fmt.Errorf("cannot query: %w", sql.ErrNoRows))

	require.True(t, errors.Is(err, sql.ErrNoRows))
	require.True(t, Is(err, sql.ErrNoRows))
}

func TestStackTraces(t *testing.T) {
	err := wrapper1()
	err = Trace(err)

	require.Equal(t, err.Error(), "added message: new error here!")

	fmt.Println(Stack(err))
	fmt.Println(Details(err))
}

//go:noinline
func wrapper1() error {
	return Trace(wrapper2())
}

//go:noinline
func wrapper2() error {
	if err := wrapper3(); err != nil {
		return Trace(err)
	}
	return nil
}

//go:noinline
func wrapper3() error {
	return Errorf("added message: %w", wrapper4())
}

//go:noinline
func wrapper4() error {
	return Trace(wrapper5())
}

func wrapper5() error {
	return Trace(unwrappedError())
}

//go:noinline
func unwrappedError() error {
	return New("new error here!")
}

func TestCauseRecoverWithUnwrap(t *testing.T) {
	err := fmt.Errorf("example: %w", wrapper1())
	fmt.Println(Stack(err))
}
