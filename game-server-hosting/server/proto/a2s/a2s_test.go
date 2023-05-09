package a2s

import (
	"bytes"
	"runtime"
	"testing"

	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server/proto"
	"github.com/stretchr/testify/require"
)

func Test_Respond(t *testing.T) {
	t.Parallel()
	q, err := NewQueryResponder(&proto.QueryState{
		CurrentPlayers: 1,
		MaxPlayers:     2,
		ServerName:     "foo",
		Map:            "map",
		GameType:       "type",
	})
	require.NoError(t, err)
	require.NotNil(t, q)

	// Query packet
	resp, err := q.Respond("", a2sInfoRequest)
	require.NoError(t, err)
	require.Equal(
		t,
		bytes.Join(
			[][]byte{
				{0xFF, 0xFF, 0xFF, 0xFF, 0x49},
				{1},
				[]byte("foo\x00"),
				[]byte("map\x00"),
				[]byte("n/a\x00"),
				[]byte("type\x00"),
				{0, 0},
				{1},
				{2},
				{0},
				{0},
				{environmentFromRuntime(runtime.GOOS)},
				{0},
				{0},
			},
			nil,
		),
		resp,
	)
}

func Test_environmentFromRuntime(t *testing.T) {
	t.Parallel()

	require.Equal(t, byte('m'), environmentFromRuntime("darwin"))
	require.Equal(t, byte('w'), environmentFromRuntime("windows"))
	require.Equal(t, byte('l'), environmentFromRuntime("linux"))
	require.Equal(t, byte('l'), environmentFromRuntime("foo"))
}
