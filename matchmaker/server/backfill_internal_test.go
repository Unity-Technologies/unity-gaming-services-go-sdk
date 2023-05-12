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
	"github.com/stretchr/testify/require"
)

func localProxyServer() (*httptest.Server, string) {
	token := "eyJhbGciOiJSUzI1NiIsImtpZCI6IjAwOWFkOGYzYWJhN2U4NjRkNTg5NTVmNzYwMWY1YTgzNDg2OWJjNTMiLCJ0eXAiOiJKV1QifQ." +
		"eyJlbnZpcm9ubWVudF9pZCI6ImJiNjc5ZWMxLTM3ZmItNDZjNi1iMmZjLWNkNDk4NzJlMmMxYSIsImV4cCI6MTY3NDg1NDEzNiwiaWF0Ijox" +
		"NjQzMzE4MTM2LCJwcm9qZWN0X2d1aWQiOiJlODBlMmZmMS0zZmFhLTRhOTQtOWUyZC1hMDIxMDdhZTJhODMifQ.FejrCFVs351JQmt_QYUGy" +
		"pG6ECy8c2N2WDFu2a7Ww85MvUWXpdB6KRnRdryKIGTNqNrRhP1wHLQZDYtCGZGc36mBoJ3Kz_1yONp3MDmC92cHWP-9duoB5otrkD66TigtI" +
		"cXruKdD65vBehFHod2gYvAwhnGa0GWJV4TLR927KiFC_O4mkxIAyTYued3rsFRgCXwlePY2kglOcpCaa8r_86hta4QYbZRmdfTu9ZNeW6K92" +
		"t8cMoUF_01Re7Gq4gZ-UwEi9IQ9E1ltITyfkY6ksmoURGEZKNuicRrzSTAzUpv460YGCJOZSbbA7ua8DR4qcTgZKDpWUN1LEJoYkuovJcAgj" +
		"_5svOgdAcPAnmwtkpQQsJx1SSwy9ODFgGozis8k3jxbj_nyd-7zve5KG7l6nNbpnQvG8DIJTIGAl-pQQ_lVvhBlcdeaUeiu4zx5DbijEgqiE" +
		"XGeTEWZegCMDET_4kyEN-Bs8Bzu4wH_w7MPMQANWuQnB5P-Y4t_wKSLLgOUF5yEZnDm5cVOojnIbYCaGOC5IVj8o4ki2vuff92mAdKWOWIYV" +
		"-9pg24XDlgss6csGw_8vVO-5p9fUHI4d0nRsIB_YeblNrVEcJeiVtVFA_yzx_v9K8AJyt_xZUhsJ3N85E9ftIP5NuHIL0sNxwl7m6dzHQ9Xw" +
		"iQJ_pZU4QFzIJI"

	// Example JWT token with invalid signature
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{
  			"token": "%s",
  			"error": ""
		}
`, token)
	})), token
}

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

	proxy, token := localProxyServer()
	defer proxy.Close()

	g, err := New(gsh.TypeAllocation)
	require.NoError(t, err)

	result, err := g.getJwtToken(&gsh.Config{
		LocalProxyURL: proxy.URL,
	})
	require.NoError(t, err)
	require.Equal(t, token, result)
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

	proxy, _ := localProxyServer()
	defer proxy.Close()

	queryEndpoint, err := getRandomPortAssignment()
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "server.json")
	require.NoError(t, os.WriteFile(path, []byte(fmt.Sprintf(`{
		"localProxyUrl": "%s",
		"queryPort": "%s"
	}`, proxy.URL, strings.Split(queryEndpoint, ":")[1])), 0o600))

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
