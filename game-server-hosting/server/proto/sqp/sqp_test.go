package sqp

import (
	"bytes"
	"testing"
	"time"

	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server/proto"
	"github.com/stretchr/testify/require"
)

func Test_Respond(t *testing.T) {
	t.Parallel()
	q, err := NewQueryResponder(&proto.QueryState{
		CurrentPlayers: 1,
		MaxPlayers:     2,
	})
	require.NoError(t, err)
	require.NotNil(t, q)

	// Add a scale challenge to make sure the responder cleans it up.
	const staleKey = "stale-challenge"
	q.challenges.Store(staleKey, challengeEntry{
		expiryUTC: time.Now().Add(-1 * time.Hour),
	})

	addr := "client-addr:65534"

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
				resp[1:5], // challenge
				{0, 1},    // SQP version
				{1},       // Request chunks (server info only)
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
				{0x0, 0xe, 0x0, 0x0, 0x0, 0xa, 0x0, 0x1, 0x0, 0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
			},
			nil,
		),
		resp,
	)

	// Ensure stale key has been purged
	require.Eventually(t, func() bool {
		_, ok := q.challenges.Load(staleKey)
		return !ok
	}, 5*time.Second, 100*time.Millisecond)
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
	require.ErrorIs(t, err, ErrNoChallenge)
}

func Test_Respond_mismatchedChallenge(t *testing.T) {
	t.Parallel()
	q, err := NewQueryResponder(&proto.QueryState{
		CurrentPlayers: 1,
		MaxPlayers:     2,
	})
	require.NoError(t, err)
	require.NotNil(t, q)

	addr := "client-addr:65534"

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
	require.ErrorIs(t, err, ErrChallengeMismatch)
}

func Test_purgeStaleChallenges(t *testing.T) {
	t.Parallel()
	q, err := NewQueryResponder(&proto.QueryState{
		CurrentPlayers: 1,
		MaxPlayers:     2,
	})
	require.NoError(t, err)

	now := time.Now().UTC()
	q.challenges.Store("127.0.0.1", challengeEntry{expiryUTC: now.Add(-1 * time.Minute)})
	q.challenges.Store("127.0.0.2", challengeEntry{expiryUTC: now.Add(1 * time.Minute)})
	q.challenges.Store("127.0.0.3", challengeEntry{expiryUTC: now.Add(-1 * time.Hour)})

	q.purgeStaleChallenges(now)

	keys := make([]string, 0)
	q.challenges.Range(func(key any, _ any) bool {
		keys = append(keys, key.(string))
		return true
	})

	require.Equal(t, []string{"127.0.0.2"}, keys)
}
