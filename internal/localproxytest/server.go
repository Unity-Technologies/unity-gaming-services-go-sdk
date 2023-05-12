package localproxytest

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/centrifugal/centrifuge"
)

// MockLocalProxy represents a mock implementation of the Game Server Hosting machine-local proxy.
type MockLocalProxy struct {
	// Server handles arbitrary HTTP requests.
	Server *httptest.Server

	// Node is a centrifuge broker node handled via websockets.
	Node *centrifuge.Node

	// Host is the hostname of the proxy, including protocol.
	Host string

	// JWT is the mock token this instance of the mock uses.
	JWT string
}

// NewLocalProxy sets up a new websocket server with centrifuge which accepts all connections and subscriptions.
// It also handles the JWT token endpoint.
func NewLocalProxy() (*MockLocalProxy, error) {
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

	node, err := centrifuge.New(centrifuge.Config{})
	if err != nil {
		return nil, err
	}

	node.OnConnecting(func(_ context.Context, _ centrifuge.ConnectEvent) (centrifuge.ConnectReply, error) {
		return centrifuge.ConnectReply{
			Credentials: &centrifuge.Credentials{},
		}, nil
	})

	node.OnConnect(func(client *centrifuge.Client) {
		client.OnSubscribe(func(ev centrifuge.SubscribeEvent, cb centrifuge.SubscribeCallback) {
			cb(centrifuge.SubscribeReply{}, nil)
		})
	})

	if err = node.Run(); err != nil {
		return nil, err
	}

	ws := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		// Satisfy the request for a connection to a centrifuge broker.
		case "/v1/connection/websocket":
			centrifuge.NewWebsocketHandler(node, centrifuge.WebsocketConfig{}).ServeHTTP(w, r)

		// Satisfy the request for a JWT.
		case "/token":
			fmt.Fprintf(w, `{
				"token": "%s",
				"error": ""
			}`, token)
		}
	}))

	return &MockLocalProxy{
		Server: ws,
		Node:   node,
		Host:   ws.URL,
		JWT:    token,
	}, nil
}

// Close closes the testing SDK server.
func (s *MockLocalProxy) Close() {
	s.Server.Close()
}