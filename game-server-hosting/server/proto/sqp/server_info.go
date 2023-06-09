package sqp

import (
	"sync/atomic"

	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server/proto"
)

type (
	// sqpServerInfo holds the server info chunk data.
	sqpServerInfo struct {
		CurrentPlayers uint16
		MaxPlayers     uint16
		ServerName     string
		GameType       string
		BuildID        string
		GameMap        string
		Port           uint16
	}
)

// queryStateToServerInfo converts proto.QueryState to sqpServerInfo.
func queryStateToServerInfo(qs *proto.QueryState) *sqpServerInfo {
	if qs == nil {
		return &sqpServerInfo{
			ServerName: "n/a",
			GameType:   "n/a",
			GameMap:    "n/a",
		}
	}

	return &sqpServerInfo{
		CurrentPlayers: uint16(atomic.LoadInt32(&qs.CurrentPlayers)),
		MaxPlayers:     uint16(qs.MaxPlayers),
		ServerName:     qs.ServerName,
		GameType:       qs.GameType,
		GameMap:        qs.Map,
		Port:           qs.Port,
	}
}

// Size returns the number of bytes sqpServerInfo will use on the wire.
func (si sqpServerInfo) Size() uint32 {
	return uint32(
		2 + // CurrentPlayers
			2 + // MaxPlayers
			len([]byte(si.ServerName)) + 1 +
			len([]byte(si.GameType)) + 1 +
			len([]byte(si.BuildID)) + 1 +
			len([]byte(si.GameMap)) + 1 +
			2, // Port
	)
}
