package server

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server/proto"
	"github.com/stretchr/testify/require"
)

// getRandomPortAssignment returns a free port and network available for testing.
func getRandomPortAssignment() (string, error) {
	addr, err := net.ResolveTCPAddr("tcp4", "localhost:0")
	if err != nil {
		return "", err
	}

	l, err := net.ListenTCP("tcp4", addr)
	if err != nil {
		return "", err
	}

	if err = l.Close(); err != nil {
		return "", err
	}

	return l.Addr().String(), nil
}

func Test_StartStopQuery(t *testing.T) {
	t.Parallel()

	queryEndpoint, err := getRandomPortAssignment()
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "server.json")
	require.NoError(t, os.WriteFile(path, []byte(fmt.Sprintf(`{
		"queryPort": "%s"
	}`, strings.Split(queryEndpoint, ":")[1])), 0600))

	s, err := New(TypeAllocation)
	require.NoError(t, err)
	require.NotNil(t, s)
	s.cfgFile = path

	require.NoError(t, s.Start())

	// Check query port is open on SQP (check that we receive an SQP challenge response)
	conn, err := net.Dial("udp4", queryEndpoint)
	require.NoError(t, err)
	require.NotNil(t, conn)

	_, err = conn.Write([]byte{0, 0, 0, 0, 0})
	require.NoError(t, err)

	buf := make([]byte, 9)
	_, err = conn.Read(buf)
	require.NoError(t, err)
	require.Equal(t, byte(0), buf[0])

	challenge := binary.BigEndian.Uint32(buf[1:])
	require.Greater(t, challenge, uint32(0))

	require.NoError(t, s.Stop())
	require.Len(t, s.OnError(), 0)
}

func Test_OnAllocate(t *testing.T) {
	t.Parallel()

	c := Config{
		AllocatedUUID: "a-uuid",
	}
	s, err := New(TypeAllocation)
	require.NoError(t, err)

	go func() {
		s.chanAllocated <- c
	}()

	require.Equal(t, c, <-s.OnAllocate())
}

func Test_OnDeallocate(t *testing.T) {
	t.Parallel()

	c := Config{
		AllocatedUUID: "a-uuid",
	}
	s, err := New(TypeAllocation)
	require.NoError(t, err)

	go func() {
		s.chanDeallocated <- c
	}()

	require.Equal(t, c, <-s.OnDeallocate())
}

func Test_OnConfigurationChanged(t *testing.T) {
	t.Parallel()

	c := Config{
		AllocatedUUID: "a-uuid",
	}
	s, err := New(TypeAllocation)
	require.NoError(t, err)

	go func() {
		s.chanConfigurationChanged <- c
	}()

	require.Equal(t, c, <-s.OnConfigurationChanged())
}

func Test_OnError(t *testing.T) {
	t.Parallel()

	e := errors.New("bang")
	s, err := New(TypeAllocation)
	require.NoError(t, err)

	go func() {
		s.chanError <- e
	}()

	require.Equal(t, e, <-s.OnError())
}

func Test_PlayerJoinedAndLeft(t *testing.T) {
	t.Parallel()

	s, err := New(TypeAllocation)
	require.NoError(t, err)

	i := s.PlayerJoined()
	require.Equal(t, int32(1), i)
	require.Equal(t, int32(1), s.state.CurrentPlayers)

	i = s.PlayerLeft()
	require.Equal(t, int32(0), i)
	require.Equal(t, int32(0), i)

	// Make sure we do not underflow.
	i = s.PlayerLeft()
	require.Equal(t, int32(0), i)
	require.Equal(t, int32(0), i)
}

func Test_DataSettings(t *testing.T) {
	t.Parallel()

	s, err := New(TypeAllocation)
	require.NoError(t, err)

	s.SetMaxPlayers(10)
	s.SetServerName("foo")
	s.SetGameType("type")
	s.SetGameMap("map")

	require.Equal(t, proto.QueryState{
		MaxPlayers: 10,
		ServerName: "foo",
		GameType:   "type",
		Map:        "map",
	}, s.state)
}
