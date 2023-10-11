package server

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server/model"
	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server/proto"
	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server/proto/sqp"
	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/internal/localproxytest"
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

	svr, err := localproxytest.NewLocalProxy()
	require.NoError(t, err)
	defer svr.Close()

	path := filepath.Join(dir, "server.json")
	require.NoError(t, os.WriteFile(path, []byte(fmt.Sprintf(`{
		"queryPort": "%s",
		"serverID": "1234",
		"serverLogDir": "%s",
		"localProxyUrl": "%s"
	}`, strings.Split(queryEndpoint, ":")[1], filepath.Join(dir, "logs"), svr.Host)), 0o600))

	s, err := New(
		TypeAllocation,
		WithConfigPath(path),
	)
	require.NoError(t, err)
	require.NotNil(t, s)

	require.NoError(t, s.Start())

	// Make sure logging directory has been created
	require.DirExists(t, filepath.Join(dir, "logs"))

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

	e := errors.New("bang") //nolint: goerr113
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
	require.Equal(t, int32(0), s.state.CurrentPlayers)

	// Make sure we do not underflow.
	i = s.PlayerLeft()
	require.Equal(t, int32(0), i)
	require.Equal(t, int32(0), s.state.CurrentPlayers)
}

func Test_SetCurrentPlayers(t *testing.T) {
	t.Parallel()

	s, err := New(TypeAllocation)
	require.NoError(t, err)

	s.SetCurrentPlayers(10)
	require.Equal(t, int32(10), s.state.CurrentPlayers)

	s.SetCurrentPlayers(-1)
	require.Equal(t, int32(0), s.state.CurrentPlayers)
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

func Test_Reserve_Unreserve(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	queryEndpoint, err := getRandomPortAssignment()
	require.NoError(t, err)

	svr, err := localproxytest.NewLocalProxy()
	require.NoError(t, err)
	defer svr.Close()

	path := filepath.Join(dir, "server.json")
	require.NoError(t, os.WriteFile(path, []byte(fmt.Sprintf(`{
		"queryPort": "%s",
		"serverID": "1",
		"serverLogDir": "%s",
		"localProxyUrl": "%s",
		"queryType": "sqp"
	}`, strings.Split(queryEndpoint, ":")[1], filepath.Join(dir, "logs"), svr.Host)), 0o600))

	s, err := New(
		TypeReservation,
		WithConfigPath(path),
	)
	require.NoError(t, err)
	require.NotNil(t, s)

	require.NoError(t, s.Start())

	resp, err := s.Reserve(context.Background(), &model.ReserveRequest{})
	require.Nil(t, err)
	require.Equal(t, svr.ReserveResponse, resp)
	require.NoError(t, s.Unreserve(context.Background()))

	require.NoError(t, s.Stop())
}

func Test_Hold_Release(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	queryEndpoint, err := getRandomPortAssignment()
	require.NoError(t, err)

	svr, err := localproxytest.NewLocalProxy()
	require.NoError(t, err)
	defer svr.Close()

	path := filepath.Join(dir, "server.json")
	require.NoError(t, os.WriteFile(path, []byte(fmt.Sprintf(`{
		"queryPort": "%s",
		"serverID": "1",
		"serverLogDir": "%s",
		"localProxyUrl": "%s",
		"queryType": "sqp"
	}`, strings.Split(queryEndpoint, ":")[1], filepath.Join(dir, "logs"), svr.Host)), 0o600))

	s, err := New(
		TypeReservation,
		WithConfigPath(path),
	)
	require.NoError(t, err)
	require.NotNil(t, s)

	require.NoError(t, s.Start())

	resp, err := s.Hold(context.Background(), &model.HoldRequest{})
	require.NoError(t, err)
	require.Equal(t, svr.HoldStatus, resp)

	resp, err = s.HoldStatus(context.Background())
	require.NoError(t, err)
	require.Equal(t, svr.HoldStatus, resp)

	require.NoError(t, s.Release(context.Background()))
	require.NoError(t, s.Stop())
}

func Test_Reserve_Unreserve_ErrorPaths(t *testing.T) {
	t.Parallel()

	allocationServer, err := New(TypeAllocation)
	require.NoError(t, err)
	reservationServer, err := New(TypeReservation)
	require.NoError(t, err)

	_, err = allocationServer.Reserve(context.Background(), &model.ReserveRequest{})
	require.ErrorIs(t, err, ErrOperationNotApplicable)
	err = allocationServer.Unreserve(context.Background())
	require.ErrorIs(t, err, ErrOperationNotApplicable)

	_, err = reservationServer.Reserve(nil, &model.ReserveRequest{}) //nolint: staticcheck
	require.ErrorIs(t, err, ErrNilContext)
	_, err = reservationServer.Reserve(context.Background(), nil)
	require.ErrorIs(t, err, ErrNilArgs)
	err = reservationServer.Unreserve(nil) //nolint: staticcheck
	require.ErrorIs(t, err, ErrNilContext)
}

func Test_Hold_Release_ErrorPaths(t *testing.T) {
	reservationServer, err := New(TypeReservation)
	require.NoError(t, err)

	_, err = reservationServer.Hold(nil, &model.HoldRequest{}) //nolint: staticcheck
	require.ErrorIs(t, err, ErrNilContext)

	_, err = reservationServer.HoldStatus(nil) //nolint: staticcheck
	require.ErrorIs(t, err, ErrNilContext)

	err = reservationServer.Release(nil) //nolint: staticcheck
	require.ErrorIs(t, err, ErrNilContext)
}

func Test_SetMetric(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	queryEndpoint, err := getRandomPortAssignment()
	require.NoError(t, err)

	svr, err := localproxytest.NewLocalProxy()
	require.NoError(t, err)
	defer svr.Close()

	path := filepath.Join(dir, "server.json")
	require.NoError(t, os.WriteFile(path, []byte(fmt.Sprintf(`{
		"queryPort": "%s",
		"serverID": "1234",
		"serverLogDir": "%s",
		"localProxyUrl": "%s",
		"queryType": "sqp"
	}`, strings.Split(queryEndpoint, ":")[1], filepath.Join(dir, "logs"), svr.Host)), 0o600))

	s, err := New(
		TypeAllocation,
		WithConfigPath(path),
	)
	require.NoError(t, err)
	require.NotNil(t, s)

	require.NoError(t, s.Start())

	// Add metric to the first index.
	require.NoError(t, s.SetMetric(0, 1.234))
	s.stateLock.Lock()
	require.Equal(t, []float32{1.234}, s.state.Metrics)
	s.stateLock.Unlock()

	// Add metric to the last index - make sure all indices in between are set to the default values.
	require.NoError(t, s.SetMetric(sqp.MaxMetrics-1, 5.678))
	s.stateLock.Lock()
	require.Equal(t, []float32{1.234, 0, 0, 0, 0, 0, 0, 0, 0, 5.678}, s.state.Metrics)
	s.stateLock.Unlock()

	// Add a metric somewhere in the middle.
	require.NoError(t, s.SetMetric(4, 9.012))
	s.stateLock.Lock()
	require.Equal(t, []float32{1.234, 0, 0, 0, 9.012, 0, 0, 0, 0, 5.678}, s.state.Metrics)
	s.stateLock.Unlock()

	// Attempt to add metric out of bounds - an error should be returned and no change observed to the underlying buffer.
	require.ErrorIs(t, s.SetMetric(sqp.MaxMetrics, 0.123), ErrMetricOutOfBounds)
	s.stateLock.Lock()
	require.Equal(t, []float32{1.234, 0, 0, 0, 9.012, 0, 0, 0, 0, 5.678}, s.state.Metrics)
	s.stateLock.Unlock()

	// Attempt to set metrics on A2S - this should fail as this is currently unsupported.
	s.currentConfigMtx.Lock()
	s.currentConfig.QueryType = QueryProtocolA2S
	s.currentConfigMtx.Unlock()
	require.ErrorIs(t, s.SetMetric(0, 1.234), ErrMetricsUnsupported)

	require.NoError(t, s.Stop())
}
