package sqp

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server/proto"
	"github.com/stretchr/testify/require"
)

const addrKey = "client-addr:65534"

func Test_Respond(t *testing.T) {
	t.Parallel()
	q, err := NewQueryResponder(&proto.QueryState{
		CurrentPlayers: 1,
		MaxPlayers:     2,
		Metrics:        make([]float32, 1, MaxMetrics),
	})
	require.NoError(t, err)
	require.NotNil(t, q)

	// Set a metric value.
	metricBytes := bytes.NewBuffer(nil)
	q.State.Metrics[0] = 1.234
	require.NoError(t, binary.Write(metricBytes, binary.BigEndian, q.State.Metrics[0]))

	addr := addrKey

	// Challenge packet
	resp, err := q.Respond(addr, []byte{0, 0, 0, 0, 0})
	require.NoError(t, err)
	require.Equal(t, byte(0), resp[0])

	// Query packet
	resp, err = q.Respond(
		addr,
		bytes.Join(
			[][]byte{
				{1},
				resp[1:5],    // challenge
				{0, 2},       // SQP version
				{0b00010001}, // Request chunks (server and metrics info)
			},
			nil,
		),
	)
	require.NoError(t, err)
	require.Equal(
		t,
		bytes.Join(
			[][]byte{
				{1},
				resp[1:5],
				resp[5:7],
				{0},
				{0},
				{0x0, 0x17},
				// Server Info chunk
				{0x0, 0x0, 0x0, 0xa, 0x0, 0x1, 0x0, 0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
				// Metrics chunk
				{0x0, 0x0, 0x0, 0x5, 0x1},
				metricBytes.Bytes(),
			},
			nil,
		),
		resp,
	)
}

func Test_Respond_ServerInfoOnly(t *testing.T) {
	t.Parallel()
	q, err := NewQueryResponder(&proto.QueryState{
		CurrentPlayers: 1,
		MaxPlayers:     2,
	})
	require.NoError(t, err)
	require.NotNil(t, q)

	addr := addrKey

	// Challenge packet
	resp, err := q.Respond(addr, []byte{0, 0, 0, 0, 0})
	require.NoError(t, err)
	require.Equal(t, byte(0), resp[0])

	// Query packet
	resp, err = q.Respond(
		addr,
		bytes.Join(
			[][]byte{
				{1},
				resp[1:5],    // challenge
				{0, 1},       // SQP version
				{0b00000001}, // Request chunks (server info only)
			},
			nil,
		),
	)
	require.NoError(t, err)
	require.Equal(
		t,
		bytes.Join(
			[][]byte{
				{1},
				resp[1:5],
				resp[5:7],
				{0},
				{0},
				{0x0, 0xe},
				// Server Info chunk
				{0x0, 0x0, 0x0, 0xa, 0x0, 0x1, 0x0, 0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
			},
			nil,
		),
		resp,
	)
}

func Test_Respond_MetricsOnly(t *testing.T) {
	t.Parallel()
	q, err := NewQueryResponder(&proto.QueryState{
		CurrentPlayers: 1,
		MaxPlayers:     2,
		Metrics:        make([]float32, 1, MaxMetrics),
	})
	require.NoError(t, err)
	require.NotNil(t, q)

	// Set a metric value.
	metricBytes := bytes.NewBuffer(nil)
	q.State.Metrics[0] = 1.234
	require.NoError(t, binary.Write(metricBytes, binary.BigEndian, q.State.Metrics[0]))

	addr := addrKey

	// Challenge packet
	resp, err := q.Respond(addr, []byte{0, 0, 0, 0, 0})
	require.NoError(t, err)
	require.Equal(t, byte(0), resp[0])

	// Query packet
	resp, err = q.Respond(
		addr,
		bytes.Join(
			[][]byte{
				{1},
				resp[1:5],    // challenge
				{0, 2},       // SQP version
				{0b00010000}, // Request chunks (metrics info only)
			},
			nil,
		),
	)
	require.NoError(t, err)
	require.Equal(
		t,
		bytes.Join(
			[][]byte{
				{1},
				resp[1:5],
				resp[5:7],
				{0},
				{0},
				{0x0, 0x9},
				// Metrics chunk
				{0x0, 0x0, 0x0, 0x5, 0x1},
				metricBytes.Bytes(),
			},
			nil,
		),
		resp,
	)
}

func Test_Respond_NoMetricsInVersion1(t *testing.T) {
	t.Parallel()
	q, err := NewQueryResponder(&proto.QueryState{
		CurrentPlayers: 1,
		MaxPlayers:     2,
		Metrics:        make([]float32, 1, MaxMetrics),
	})
	require.NoError(t, err)
	require.NotNil(t, q)

	addr := addrKey

	// Challenge packet
	resp, err := q.Respond(addr, []byte{0, 0, 0, 0, 0})
	require.NoError(t, err)
	require.Equal(t, byte(0), resp[0])

	// Query packet
	resp, err = q.Respond(
		addr,
		bytes.Join(
			[][]byte{
				{1},
				resp[1:5],    // challenge
				{0, 1},       // SQP version
				{0b00010000}, // Request chunks (metrics info only)
			},
			nil,
		),
	)
	require.NoError(t, err)
	require.Equal(
		t,
		bytes.Join(
			[][]byte{
				{1},
				resp[1:5],
				resp[5:7],
				{0},
				{0},
				{0x0, 0x0},
			},
			nil,
		),
		resp,
	)
}

func Test_Respond_noChallenge(t *testing.T) {
	t.Parallel()
	q, err := NewQueryResponder(&proto.QueryState{
		CurrentPlayers: 1,
		MaxPlayers:     2,
	})
	require.NoError(t, err)
	require.NotNil(t, q)

	resp, err := q.Respond(
		"my-addr",
		bytes.Join(
			[][]byte{
				{1},
				{0, 0, 0, 0}, // challenge
				{0, 1},       // SQP version
				{1},          // Request chunks (server info only)
			},
			nil,
		),
	)
	require.Nil(t, resp)
	require.ErrorIs(t, err, proto.ErrNoChallenge)
}

func Test_Respond_mismatchedChallenge(t *testing.T) {
	t.Parallel()
	q, err := NewQueryResponder(&proto.QueryState{
		CurrentPlayers: 1,
		MaxPlayers:     2,
	})
	require.NoError(t, err)
	require.NotNil(t, q)

	addr := addrKey

	// Challenge packet
	resp, err := q.Respond(addr, []byte{0, 0, 0, 0, 0})
	require.NoError(t, err)
	require.Equal(t, byte(0), resp[0])

	resp, err = q.Respond(
		addr,
		bytes.Join(
			[][]byte{
				{1},
				{0, 0, 0, 0}, // challenge
				{0, 1},       // SQP version
				{1},          // Request chunks (server info only)
			},
			nil,
		),
	)
	require.Nil(t, resp)
	require.ErrorIs(t, err, proto.ErrChallengeMismatch)
}
