package sqp

import (
	"bytes"
	"encoding/binary"

	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server/proto"
)

type (
	// QueryResponder represents a responder capable of responding to SQP-formatted queries.
	QueryResponder struct {
		*proto.QueryBase
		enc *encoder
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
)

// NewQueryResponder returns creates a new responder capable of responding
// to SQP-formatted queries.
func NewQueryResponder(state *proto.QueryState) (*QueryResponder, error) {
	q := &QueryResponder{
		QueryBase: &proto.QueryBase{
			State: state,
		},
		enc: &encoder{},
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

	return nil, errUnsupportedQuery
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
	v, err := q.GenerateChallenge(clientAddress)
	if err != nil {
		return nil, err
	}

	resp := bytes.NewBuffer(nil)
	err = proto.WireWrite(
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
	if len(buf) < 8 {
		return nil, errInvalidPacketLength
	}

	challenge := binary.BigEndian.Uint32(buf[1:5])
	if err := q.ChallengeMatchesForClient(clientAddress, challenge); err != nil {
		return nil, err
	}

	if binary.BigEndian.Uint16(buf[5:7]) != 1 {
		return nil, NewUnsupportedSQPVersionError(int8(buf[6]))
	}

	requestedChunks := buf[7]
	wantsServerInfo := requestedChunks&0x1 == 1
	f := queryWireFormat{
		Header:     1,
		Challenge:  challenge,
		SQPVersion: 1,
	}

	resp := bytes.NewBuffer(nil)

	if wantsServerInfo {
		f.ServerInfo = queryStateToServerInfo(q.State)
		f.ServerInfoLength = f.ServerInfo.Size()
		f.PayloadLength += uint16(f.ServerInfoLength) + 4
	}

	if err := proto.WireWrite(resp, q.enc, f); err != nil {
		return nil, err
	}

	return resp.Bytes(), nil
}
