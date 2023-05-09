package server

import (
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	gsh "github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server"
)

type (
	// Server represents an instance of a game server which also handles matchmaker backfill requests.
	Server struct {
		*gsh.Server

		// httpClient is an http client that is used to retrieve the token from the payload
		// proxy as well as send backfill ticket approvals to the matchmaker
		httpClient *http.Client

		// Synchronisation
		done chan struct{}
		wg   sync.WaitGroup
	}
)

// New creates a new instance of Server, denoting which type of server to use.
func New(serverType gsh.Type, opts ...gsh.Option) (*Server, error) {
	base, err := gsh.New(serverType, opts...)
	if err != nil {
		return nil, err
	}

	return &Server{
		Server:     base,
		done:       make(chan struct{}, 1),
		httpClient: &http.Client{},
	}, nil
}

// Start starts the server, opening the configured query port which responds with the configured protocol.
// The event loop will also listen for changes to the `server.json` configuration file, publishing any
// changes in the form of allocation or de-allocation messages. The server will also listen for changes to the backfill
// state in Matchmaker and propagate those changes to a consumer.
func (s *Server) Start() error {
	if err := s.Server.Start(); err != nil {
		return err
	}

	if backfillEnabled(s.Config()) {
		go s.keepAliveBackfill()
	}

	return nil
}

// WaitUntilTerminated waits until the server receives a termination signal from the platform.
// The Unity Gaming Services process management daemon will signal the game server to
// stop. A graceful stop signal (SIGTERM) will be sent if the game server fleet has been
// configured to support it.
func (s *Server) WaitUntilTerminated() error {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	return s.Stop()
}

// Stop stops the game, pushing a de-allocation message and closing the query port.
func (s *Server) Stop() error {
	// Stop our server implementation.
	close(s.done)
	s.wg.Wait()

	// Stop base server.
	return s.Server.Stop()
}
