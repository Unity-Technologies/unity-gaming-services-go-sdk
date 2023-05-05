package server

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func Test_WithQueryWriteBuffer(t *testing.T) {
	t.Parallel()
	s := &Server{}
	WithQueryWriteBuffer(1024)(s)
	require.Equal(t, 1024, s.queryWriteBufferSizeBytes)
}

func Test_WithQueryReadBuffer(t *testing.T) {
	t.Parallel()
	s := &Server{}
	WithQueryReadBuffer(1024)(s)
	require.Equal(t, 1024, s.queryReadBufferSizeBytes)
}

func Test_WithQueryWriteDeadlineDuration(t *testing.T) {
	t.Parallel()
	s := &Server{}
	WithQueryWriteDeadlineDuration(1 * time.Second)(s)
	require.Equal(t, 1*time.Second, s.queryWriteDeadlineDuration)
}
