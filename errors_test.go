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
	err := Trace(foo())

	require.Equal(t, err.Error(), "new error here!")

	fmt.Println(Stack(err))
	fmt.Println(Details(err))
}

func foo() error {
	return Trace(bar())
}

func bar() error {
	return Trace(baz())
}

func baz() error {
	return Errorf("new error here!")
}