package sqp

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"sync"
	"time"

	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server/proto"
)

type (
	// QueryResponder represents a responder capable of responding to SQP-formatted queries.
	QueryResponder struct {
		challenges               sync.Map
		enc                      *encoder
		state                    *proto.QueryState
		challengesLastPurgedUTC  time.Time
		challengesLastPurgedLock sync.RWMutex
	}

	// challengeWireFormat describes the format of an SQP challenge response.
	challengeWireFormat struct {
		Header    byte
		Challenge uint32
	}

	// queryWireFormat describes the format of an SQP query response.
	queryWireFormat struct {
		Header           byte
		Challenge        uint32
		SQPVersion       uint16
		CurrentPacketNum byte
		LastPacketNum    byte
		PayloadLength    uint16
		ServerInfoLength uint32
		ServerInfo       sqpServerInfo
	}

	// challengeEntry represents an entry in the query responder challenges map.
	challengeEntry struct {
		value     uint32
		expiryUTC time.Time
	}
)

// NewQueryResponder returns creates a new responder capable of responding
// to SQP-formatted queries.
func NewQueryResponder(state *proto.QueryState) (*QueryResponder, error) {
	q := &QueryResponder{
		enc:                     &encoder{},
		state:                   state,
		challengesLastPurgedUTC: time.Now().UTC(),
	}

	return q, nil
}

// Respond writes a query response to the requester in the SQP wire protocol.
func (q *QueryResponder) Respond(clientAddress string, buf []byte) ([]byte, error) {
	switch {
	case isChallenge(buf):
		return q.handleChallenge(clientAddress)

	case isQuery(buf):
		return q.handleQuery(clientAddress, buf)
	}

	return nil, ErrUnsupportedQuery
}

// isChallenge determines if the input buffer corresponds to a challenge packet.
func isChallenge(buf []byte) bool {
	return bytes.Equal(buf[0:5], []byte{0, 0, 0, 0, 0})
}

// isQuery determines if the input buffer corresponds to a query packet.
func isQuery(buf []byte) bool {
	return buf[0] == 1
}

// handleChallenge handles an incoming challenge packet.
func (q *QueryResponder) handleChallenge(clientAddress string) ([]byte, error) {
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
		return nil, err
	}

	v := binary.BigEndian.Uint32(randBytes)
	q.challenges.Store(clientAddress, challengeEntry{
		value:     v,
		expiryUTC: time.Now().UTC().Add(1 * time.Minute),
	})

	resp := bytes.NewBuffer(nil)
	err := proto.WireWrite(
		resp,
		q.enc,
		challengeWireFormat{
			Header:    0,
			Challenge: v,
		},
	)
	if err != nil {
		return nil, err
	}

	return resp.Bytes(), nil
}

// handleQuery handles an incoming query packet.
func (q *QueryResponder) handleQuery(clientAddress string, buf []byte) ([]byte, error) {
	expectedChallenge, ok := q.challenges.LoadAndDelete(clientAddress)
	if !ok {
		return nil, ErrNoChallenge
	}

	if len(buf) < 8 {
		return nil, ErrInvalidPacketLength
	}

	// Challenge doesn't match, return with no response
	expectedChallengeEntry, ok := expectedChallenge.(challengeEntry)
	if !ok {
		return nil, ErrChallengeMalformed
	}

	if binary.BigEndian.Uint32(buf[1:5]) != expectedChallengeEntry.value {
		return nil, ErrChallengeMismatch
	}

	if binary.BigEndian.Uint16(buf[5:7]) != 1 {
		return nil, NewUnsupportedSQPVersionError(int8(buf[6]))
	}

	requestedChunks := buf[7]
	wantsServerInfo := requestedChunks&0x1 == 1
	f := queryWireFormat{
		Header:     1,
		Challenge:  expectedChallengeEntry.value,
		SQPVersion: 1,
	}

	resp := bytes.NewBuffer(nil)

	if wantsServerInfo {
		f.ServerInfo = queryStateToServerInfo(q.state)
		f.ServerInfoLength = f.ServerInfo.Size()
		f.PayloadLength += uint16(f.ServerInfoLength) + 4
	}

	if err := proto.WireWrite(resp, q.enc, f); err != nil {
		return nil, err
	}

	return resp.Bytes(), nil
}

// purgeStaleChallenges purges any entries which have an expiry in the past.
func (q *QueryResponder) purgeStaleChallenges(epochUTC time.Time) {
	q.challenges.Range(func(k any, v any) bool {
		if entry, ok := v.(challengeEntry); ok {
			if epochUTC.After(entry.expiryUTC) {
				q.challenges.Delete(k)
			}
		}

		return true
	})
}
