package proto

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_GenerateAndMatchChallenge(t *testing.T) {
	t.Parallel()

	q := &QueryBase{}

	// Add a stale challenge to make sure the responder cleans it up.
	const staleKey = "stale-challenge"
	q.challenges.Store(staleKey, challengeEntry{
		expiryUTC: time.Now().Add(-1 * time.Hour),
	})

	// Generate a challenge, make sure a mismatch returns as such
	const clientAddr = "client-addr:1234"
	c, err := q.GenerateChallenge(clientAddr)
	require.NoError(t, err)
	require.ErrorIs(t, q.ChallengeMatchesForClient(clientAddr, c+1), ErrChallengeMismatch)

	// Generate a challenge, make sure it has been stored properly
	c, err = q.GenerateChallenge(clientAddr)
	require.NoError(t, err)
	require.NoError(t, q.ChallengeMatchesForClient(clientAddr, c))

	// Ensure stale key has been purged
	require.Eventually(t, func() bool {
		_, ok := q.challenges.Load(staleKey)
		return !ok
	}, 5*time.Second, 100*time.Millisecond)
}

func Test_purgeStaleChallenges(t *testing.T) {
	t.Parallel()

	q := &QueryBase{}
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
