package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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

func Test_WithConfigPath(t *testing.T) {
	t.Parallel()
	s := &Server{}
	WithConfigPath("foo")(s)
	require.Equal(t, "foo", s.cfgFile)
}
