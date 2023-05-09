package server

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_pushError(t *testing.T) {
	t.Parallel()

	s := &Server{
		chanError: make(chan error, 1),
	}

	a := errors.New("a")
	b := errors.New("b")
	c := errors.New("c")

	s.PushError(a)
	s.PushError(b)
	s.PushError(c)

	require.Len(t, s.chanError, 1)
	require.Equal(t, a, <-s.chanError)
}
