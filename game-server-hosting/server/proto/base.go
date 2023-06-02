package proto

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"sync"
	"time"
)

type (
	// QueryBase is the base querying struct which can be used by other implementers to handle features such as
	// challenge generation.
	QueryBase struct {
		challenges               sync.Map
		challengesLastPurgedUTC  time.Time
		challengesLastPurgedLock sync.RWMutex

		State *QueryState
	}

	// challengeEntry represents an entry in the query responder challenges map.
	challengeEntry struct {
		value     uint32
		expiryUTC time.Time
	}
)

var (
	ErrChallengeMalformed = errors.New("challenge malformed")
	ErrChallengeMismatch  = errors.New("challenge mismatch")
	ErrNoChallenge        = errors.New("no challenge")
)

// GenerateChallenge generates a challenge value for the calling client address. If stale challenges have not been purged
// for a while, this is done asynchronously.
func (q *QueryBase) GenerateChallenge(clientAddress string) (uint32, error) {
	// Purge entries asynchronously if we haven't done so in a while.
	// Do this at the beginning so that any upcoming failures don't stop us from cleaning up.
	q.challengesLastPurgedLock.RLock()
	if time.Since(q.challengesLastPurgedUTC).Minutes() > 0 {
		q.challengesLastPurgedLock.RUnlock()
		q.challengesLastPurgedLock.Lock()
		q.challengesLastPurgedUTC = time.Now().UTC()
		q.challengesLastPurgedLock.Unlock()
		q.challengesLastPurgedLock.RLock()
		go q.purgeStaleChallenges(time.Now().UTC())
	}
	q.challengesLastPurgedLock.RUnlock()

	randBytes := make([]byte, 4)
	if _, err := rand.Read(randBytes); err != nil {
		return 0, err
	}

	v := binary.BigEndian.Uint32(randBytes)
	q.challenges.Store(clientAddress, challengeEntry{
		value:     v,
		expiryUTC: time.Now().UTC().Add(1 * time.Minute),
	})

	return v, nil
}

// ChallengeMatchesForClient determines whether the challenge value supplied by the client matches what is stored
// by this server.
func (q *QueryBase) ChallengeMatchesForClient(clientAddress string, challenge uint32) error {
	expectedChallenge, ok := q.challenges.LoadAndDelete(clientAddress)
	if !ok {
		return ErrNoChallenge
	}

	expectedChallengeEntry, ok := expectedChallenge.(challengeEntry)
	if !ok {
		return ErrChallengeMalformed
	}

	if challenge != expectedChallengeEntry.value {
		return ErrChallengeMismatch
	}

	return nil
}

// purgeStaleChallenges purges any entries which have an expiry in the past.
func (q *QueryBase) purgeStaleChallenges(epochUTC time.Time) {
	q.challenges.Range(func(k any, v any) bool {
		if entry, ok := v.(challengeEntry); ok {
			if epochUTC.After(entry.expiryUTC) {
				q.challenges.Delete(k)
			}
		}

		return true
	})
}
