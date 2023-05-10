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
	"time"

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

	dir := t.TempDir()
	queryEndpoint, err := getRandomPortAssignment()
	require.NoError(t, err)

	path := filepath.Join(dir, "server.json")
	require.NoError(t, os.WriteFile(path, []byte(fmt.Sprintf(`{
		"queryPort": "%s",
		"serverID": "1234",
		"serverLogDir": "1234/logs"
	}`, strings.Split(queryEndpoint, ":")[1])), 0600))

	s, err := New(
		TypeAllocation,
		WithConfigPath(path),
		WithHomeDirectory(dir),
	)
	require.NoError(t, err)
	require.NotNil(t, s)

	require.NoError(t, s.Start())

	// Make sure logging directory has been created
	require.DirExists(t, filepath.Join(dir, "1234", "logs"))

	// Make sure query parameters have been set
	s.stateLock.Lock()
	require.Equal(t, "go-sdk-server - 1234", s.state.ServerName)
	require.Equal(t, "go-sdk-map", s.state.Map)
	s.stateLock.Unlock()

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

	s, err := New(TypeAllocation)
	require.NoError(t, err)

	go func() {
		s.chanAllocated <- "a-uuid"
	}()

	require.Equal(t, "a-uuid", <-s.OnAllocate())
}

func Test_OnDeallocate(t *testing.T) {
	t.Parallel()

	s, err := New(TypeAllocation)
	require.NoError(t, err)

	go func() {
		s.chanDeallocated <- ""
	}()

	require.Equal(t, "", <-s.OnDeallocate())
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

func Test_New_noReservations(t *testing.T) {
	t.Parallel()

	s, err := New(TypeReservation)
	require.Nil(t, s)
	require.ErrorIs(t, err, ErrReservationsNotYetSupported)
}

func Test_New_appliesOptions(t *testing.T) {
	t.Parallel()

	s, err := New(
		TypeAllocation,
		WithQueryWriteDeadlineDuration(2*time.Second),
		WithQueryWriteBuffer(1),
		WithQueryReadBuffer(2),
	)
	require.NoError(t, err)
	require.Equal(t, 2*time.Second, s.queryWriteDeadlineDuration)
	require.Equal(t, 1, s.queryWriteBufferSizeBytes)
	require.Equal(t, 2, s.queryReadBufferSizeBytes)
}

func Test_New_Allocations(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	require.NoError(t, err)

	s, err := New(TypeAllocation)
	require.NoError(t, err)
	require.Equal(t, TypeAllocation, s.serverType)
	require.Equal(t, filepath.Join(home, "server.json"), s.cfgFile)
	require.NotNil(t, s.chanAllocated)
	require.NotNil(t, s.chanDeallocated)
	require.NotNil(t, s.chanError)
	require.NotNil(t, s.chanConfigurationChanged)
	require.NotNil(t, s.done)
	require.Equal(t, DefaultWriteBufferSizeBytes, s.queryWriteBufferSizeBytes)
	require.Equal(t, DefaultReadBufferSizeBytes, s.queryReadBufferSizeBytes)
	require.Equal(t, DefaultWriteDeadlineDuration, s.queryWriteDeadlineDuration)
}
