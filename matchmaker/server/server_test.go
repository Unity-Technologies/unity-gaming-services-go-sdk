package server

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gsh "github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server"
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

	dir := t.TempDir()
	path := filepath.Join(dir, "server.json")
	require.NoError(t, os.WriteFile(path, []byte(fmt.Sprintf(`{
		"queryPort": "%s",
		"serverLogDir": "1234/logs"
	}`, strings.Split(queryEndpoint, ":")[1])), 0o600))

	s, err := New(
		gsh.TypeAllocation,
		gsh.WithConfigPath(path),
		gsh.WithHomeDirectory(dir),
	)
	require.NoError(t, err)
	require.NotNil(t, s)

	require.NoError(t, s.Start())

	// Make sure logging directory has been created
	require.DirExists(t, filepath.Join(dir, "1234", "logs"))

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
