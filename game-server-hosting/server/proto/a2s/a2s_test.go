package a2s

import (
	"bytes"
	"encoding/binary"
	"io"
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

	clientAddr := "my-client:1234"

	// Query packet to receive challenge
	qResp, err := q.Respond(clientAddr, a2sInfoRequest)
	require.NoError(t, err)
	require.Equal(t, s2cChallengeResponse, qResp[0:5])

	// Expect challenge response
	var challenge uint32
	require.NoError(t, binary.Read(bytes.NewBuffer(qResp), binary.LittleEndian, &challenge))
	require.True(t, challenge != 0)

	// Query packet with challenge included
	req := bytes.Join(
		[][]byte{
			a2sInfoRequest,
			{0x0},      // payload
			qResp[5:9], // challenge
		},
		nil,
	)
	resp, err := q.Respond(clientAddr, req)
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
				{'d'},
				{environmentFromRuntime(runtime.GOOS)},
				{0},
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

func Test_parseInfoRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         []byte
		expected      *infoRequest
		expectedError error
	}{
		{
			name: "golden path",
			input: bytes.Join(
				[][]byte{
					{0xFF, 0xFF, 0xFF, 0xFF, 0x54},
					[]byte("hello payload\x00"),
					{1, 0, 0, 0},
				},
				nil,
			),
			expected: &infoRequest{
				Payload:   "hello payload",
				Challenge: 1,
			},
		},
		{
			name: "no payload",
			input: bytes.Join(
				[][]byte{
					{0xFF, 0xFF, 0xFF, 0xFF, 0x54},
				},
				nil,
			),
			expected: &infoRequest{},
		},
		{
			name: "one byte payload",
			input: bytes.Join(
				[][]byte{
					{0xFF, 0xFF, 0xFF, 0xFF, 0x54},
					[]byte("\x00"),
				},
				nil,
			),
			expected: &infoRequest{},
		},
		{
			name: "no challenge",
			input: bytes.Join(
				[][]byte{
					{0xFF, 0xFF, 0xFF, 0xFF, 0x54},
					[]byte("hello payload\x00"),
				},
				nil,
			),
			expected: &infoRequest{
				Payload: "hello payload",
			},
		},
		{
			name: "no nil terminator",
			input: bytes.Join(
				[][]byte{
					{0xFF, 0xFF, 0xFF, 0xFF, 0x54},
					[]byte("hello payload"),
				},
				nil,
			),
			expected: &infoRequest{
				Payload: "hello payload",
			},
		},
		{
			name: "insufficient challenge bytes",
			input: bytes.Join(
				[][]byte{
					{0xFF, 0xFF, 0xFF, 0xFF, 0x54},
					[]byte("hello payload\x00"),
					{1, 0, 0},
				},
				nil,
			),
			expectedError: io.ErrUnexpectedEOF,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := parseInfoRequest(tt.input)
			require.ErrorIs(t, err, tt.expectedError)
			require.Equal(t, tt.expected, info)
		})
	}
}
