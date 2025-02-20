package a2s

import (
	"bytes"
	"encoding/binary"
	"runtime"
	"sync/atomic"

	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server/proto"
)

type (
	// QueryResponder implements proto.QueryResponder for the A2S protocol.
	QueryResponder struct {
		*proto.QueryBase
		enc *encoder
	}

	// challengeWireFormat describes the format of a S2C_CHALLENGE query response.
	challengeWireFormat struct {
		Header    []byte
		Challenge uint32
	}

	// infoWireFormat describes the format of a A2S_INFO query response.
	infoWireFormat struct {
		Header      []byte
		Protocol    byte
		ServerName  string
		GameMap     string
		GameFolder  string
		GameName    string
		SteamAppID  int16
		PlayerCount uint8
		MaxPlayers  uint8
		NumBots     uint8
		ServerType  byte
		Environment byte
		Visibility  byte
		VACEnabled  byte
	}

	// infoRequest represents the request format for an A2S_INFO query.
	infoRequest struct {
		Payload   string
		Challenge uint32
	}
)

var (
	a2sInfoRequest       = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x54}
	a2sInfoResponse      = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x49}
	s2cChallengeResponse = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x41}
)

// NewQueryResponder returns creates a new responder capable of responding
// to a2s-formatted queries.
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
	if bytes.Equal(buf[0:5], a2sInfoRequest) {
		return q.handleInfoRequest(clientAddress, buf)
	}

	return nil, NewUnsupportedQueryError(buf[0:5])
}

func (q *QueryResponder) handleInfoRequest(clientAddress string, buf []byte) ([]byte, error) {
	info, err := parseInfoRequest(buf)
	if err != nil {
		return nil, err
	}

	var f any

	// If no challenge has been supplied, respond with one. Expect it on the next request.
	if info.Challenge == 0 {
		challenge, handleErr := q.GenerateChallenge(clientAddress)
		if handleErr != nil {
			return nil, handleErr
		}

		f = challengeWireFormat{
			Header:    s2cChallengeResponse,
			Challenge: challenge,
		}
	} else {
		if err = q.ChallengeMatchesForClient(clientAddress, info.Challenge); err != nil {
			return nil, err
		}

		w := infoWireFormat{
			Header:      a2sInfoResponse,
			Protocol:    1,
			ServerName:  "n/a",
			GameMap:     "n/a",
			GameFolder:  "n/a",
			GameName:    "n/a",
			ServerType:  'd', // d = dedicated server
			Environment: environmentFromRuntime(runtime.GOOS),
		}

		if q.State != nil {
			w.ServerName = q.State.ServerName
			w.GameMap = q.State.Map
			w.PlayerCount = byte(atomic.LoadInt32(&q.State.CurrentPlayers))
			w.MaxPlayers = byte(q.State.MaxPlayers)
			w.GameName = q.State.GameType
		}

		f = w
	}

	resp := bytes.NewBuffer(nil)

	if err := proto.WireWrite(resp, q.enc, f); err != nil {
		return nil, err
	}

	return resp.Bytes(), nil
}

// parseInfoRequest parses the incoming request as a A2S_INFO request.
func parseInfoRequest(buf []byte) (*infoRequest, error) {
	if !bytes.Equal(buf[0:5], a2sInfoRequest) {
		return nil, errNotAnInfoRequest
	}

	info := &infoRequest{}
	rBuf := bytes.NewBuffer(buf[5:])

	// Read through buffer until we reach a null-terminator
	n := 5
	for n < len(buf) {
		if buf[n] == 0 {
			break
		}
		n++
	}

	// Read the payload if it exists
	l := n - 5
	if l > 0 {
		d := make([]byte, l)
		if err := binary.Read(rBuf, binary.LittleEndian, &d); err != nil {
			return nil, err
		}

		info.Payload = string(d)
	}

	// Skip past null terminator
	_ = rBuf.Next(1)
	n++

	// No space for a challenge - return as-is.
	if n >= len(buf) {
		return info, nil
	}

	if err := binary.Read(rBuf, binary.LittleEndian, &info.Challenge); err != nil {
		return nil, err
	}

	return info, nil
}

// environmentFromRuntime returns the environment byte required for the a2s response backend based upon the operating
// system the server is running on.
func environmentFromRuntime(rt string) byte {
	switch rt {
	case "darwin":
		return byte('m')
	case "windows":
		return byte('w')
	default:
		return byte('l')
	}
}
