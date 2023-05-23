package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gsh "github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server"
	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/internal/localproxytest"
	"github.com/stretchr/testify/require"
)

func Test_approveBackfillTicket(t *testing.T) {
	t.Parallel()

	mmBackfillServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{
  			"ID": "77c31f84-b890-48e8-be08-5db9a551bba3",
  			"Connection": "127.0.0.1:9555",
  			"Attributes": {
    			"att1": 100
  			}
		}`)
	}))
	defer mmBackfillServer.Close()

	g, err := New(gsh.TypeAllocation)
	require.NoError(t, err)

	ticket, err := g.approveBackfillTicket(&gsh.Config{
		AllocatedUUID: "77c31f84-b890-48e8-be08-5db9a551bba3",
		Extra: map[string]string{
			"matchmakerUrl": mmBackfillServer.URL,
		},
	}, "token")
	require.NoError(t, err)
	require.NotNil(t, ticket)
	require.Equal(t, "77c31f84-b890-48e8-be08-5db9a551bba3", ticket.ID)
	require.Equal(t, "127.0.0.1:9555", ticket.Connection)
	require.Equal(t, 1, len(ticket.Attributes))
	require.Equal(t, 100.0, ticket.Attributes["att1"])
}

func Test_approveBackfillTicket_NotAllocated(t *testing.T) {
	t.Parallel()

	g, err := New(gsh.TypeAllocation)
	require.NoError(t, err)

	ticket, err := g.approveBackfillTicket(&gsh.Config{}, "token")
	require.Nil(t, ticket)
	require.ErrorIs(t, err, ErrNotAllocated)
}

func Test_approveBackfillTicket_TooManyRequests(t *testing.T) {
	t.Parallel()

	mmBackfillServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer mmBackfillServer.Close()

	g, err := New(gsh.TypeAllocation)
	require.NoError(t, err)

	ticket, err := g.approveBackfillTicket(&gsh.Config{
		AllocatedUUID: "77c31f84-b890-48e8-be08-5db9a551bba3",
		Extra: map[string]string{
			"matchmakerUrl": mmBackfillServer.URL,
		},
	}, "token")
	require.Nil(t, ticket)
	require.ErrorIs(t, err, errRetry)
}

func Test_approveBackfillTicket_NonOKRequest(t *testing.T) {
	t.Parallel()

	mmBackfillServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mmBackfillServer.Close()

	g, err := New(gsh.TypeAllocation)
	require.NoError(t, err)

	ticket, err := g.approveBackfillTicket(&gsh.Config{
		AllocatedUUID: "77c31f84-b890-48e8-be08-5db9a551bba3",
		Extra: map[string]string{
			"matchmakerUrl": mmBackfillServer.URL,
		},
	}, "token")
	require.Nil(t, ticket)
	require.ErrorIs(t, err, ErrBackfillApprove)
}

func Test_getJwtToken(t *testing.T) {
	t.Parallel()

	svr, err := localproxytest.NewLocalProxy()
	require.NoError(t, err)
	defer svr.Close()

	g, err := New(gsh.TypeAllocation)
	require.NoError(t, err)

	result, err := g.getJwtToken(&gsh.Config{
		LocalProxyURL: svr.Host,
	})
	require.NoError(t, err)
	require.Equal(t, svr.JWT, result)
}

func Test_getJwtToken_error(t *testing.T) {
	t.Parallel()

	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer proxy.Close()

	g, err := New(gsh.TypeAllocation)
	require.NoError(t, err)

	result, err := g.getJwtToken(&gsh.Config{
		LocalProxyURL: proxy.URL,
	})
	require.Empty(t, result)
	require.ErrorIs(t, err, errTokenFetch)
}

func Test_wrapWithConfigAndJWT(t *testing.T) {
	t.Parallel()

	svr, err := localproxytest.NewLocalProxy()
	require.NoError(t, err)
	defer svr.Close()

	queryEndpoint, err := getRandomPortAssignment()
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "server.json")
	require.NoError(t, os.WriteFile(path, []byte(fmt.Sprintf(`{
		"localProxyUrl": "%s",
		"queryPort": "%s",
		"serverLogDir": "%s",
		"serverID": "1"
	}`, svr.Host, strings.Split(queryEndpoint, ":")[1], dir)), 0o600))

	s, err := New(gsh.TypeAllocation, gsh.WithConfigPath(path))
	require.NoError(t, err)

	require.NoError(t, s.Start())

	ticket, err := s.wrapWithConfigAndJWT(func(c *gsh.Config, token string) (*BackfillTicket, error) {
		require.Equal(t, s.Config(), *c)
		return &BackfillTicket{
			ID: "abc",
		}, nil
	})
	require.NoError(t, err)
	require.Equal(t, &BackfillTicket{
		ID: "abc",
	}, ticket)

	require.NoError(t, s.Stop())
}
