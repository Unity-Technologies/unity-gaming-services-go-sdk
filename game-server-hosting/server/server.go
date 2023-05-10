package server

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server/proto"
)

type (
	// Type represents the type of server, that being 'allocations' or 'reservations'.
	Type int8

	// Server represents an instance of a game server, handling changes to configuration and responding to query requests.
	Server struct {
		// cfgFile is the file path this game uses to read its configuration from
		cfgFile string

		// internalEventProcessorReady is a channel that, when written to,
		// indicates that the internal event processor is ready.
		internalEventProcessorReady chan struct{}

		// queryBind is a UDP endpoint which responds to game queries
		queryBind *udpBinding

		// queryProto is an implementation of an interface which responds on a particular
		// query format, for example sqp, tf2e, etc.
		queryProto proto.QueryResponder

		// serverType holds the type of server this instance is.
		serverType Type

		// state represents current game states which are applicable to an incoming query,
		// for example current players, map name
		state     proto.QueryState
		stateLock sync.Mutex

		// Event Channels
		chanAllocated            chan string
		chanConfigurationChanged chan Config
		chanDeallocated          chan string
		chanError                chan error

		// Configuration-related items
		currentConfigMtx sync.RWMutex
		currentConfig    Config

		// Query-related configuration
		queryWriteBufferSizeBytes  int
		queryWriteDeadlineDuration time.Duration
		queryReadBufferSizeBytes   int

		// Synchronisation
		done chan struct{}
		wg   sync.WaitGroup

		// Environment
		homeDir string
	}
)

const (
	// TypeAllocation represents a server which is using the 'allocations' model of server usage.
	TypeAllocation = Type(0)

	// TypeReservation represents a server which is using the 'reservations' model of server usage.
	TypeReservation = Type(1)

	// DefaultWriteBufferSizeBytes represents the default size of the write buffer for the query handler.
	DefaultWriteBufferSizeBytes = 1024

	// DefaultWriteDeadlineDuration represents the default write deadline duration for responding in the query handler.
	DefaultWriteDeadlineDuration = 1 * time.Second

	// DefaultReadBufferSizeBytes represents the default size of the read buffer for the query handler.
	DefaultReadBufferSizeBytes = 1024
)

var (
	// ErrReservationsNotYetSupported represents that a reservation-based server is not yet supported by the SDK.
	ErrReservationsNotYetSupported = errors.New("reservations are not yet supported")
)

// New creates a new instance of Server, denoting which type of server to use.
func New(serverType Type, opts ...Option) (*Server, error) {
	// Reservations are not supported just yet, but provided to make the API stable.
	if serverType == TypeReservation {
		return nil, ErrReservationsNotYetSupported
	}

	dir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("error getting user home directory: %w", err)
	}

	s := &Server{
		serverType:                  serverType,
		cfgFile:                     filepath.Join(dir, "server.json"),
		homeDir:                     dir,
		chanAllocated:               make(chan string, 1),
		chanDeallocated:             make(chan string, 1),
		chanError:                   make(chan error, 1),
		chanConfigurationChanged:    make(chan Config, 1),
		internalEventProcessorReady: make(chan struct{}, 1),
		done:                        make(chan struct{}, 1),
		queryWriteBufferSizeBytes:   DefaultWriteBufferSizeBytes,
		queryWriteDeadlineDuration:  DefaultWriteDeadlineDuration,
		queryReadBufferSizeBytes:    DefaultReadBufferSizeBytes,
	}

	// Apply any specified options.
	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

// Start starts the server, opening the configured query port which responds with the configured protocol.
// The event loop will also listen for changes to the `server.json` configuration file, publishing any
// changes in the form of allocation or de-allocation messages.
// As the server can start in an allocated state, make sure that another goroutine is consuming messages from at least
// the `OnAllocated()` and `OnDeallocated()` channels before calling this method.
func (s *Server) Start() error {
	c, err := newConfigFromFile(s.cfgFile, s.homeDir)
	if err != nil {
		return err
	}

	s.setConfig(c)

	// Create the directory the logs will be present in.
	if err = os.MkdirAll(c.ServerLogDir, 0744); err != nil {
		return fmt.Errorf("error creating log directory: %w", err)
	}

	// Set some defaults for the query endpoint. These can get overwritten by the user, but best to set some defaults
	// to keep friction to a minimum.
	s.SetServerName(fmt.Sprintf("go-sdk-server - %s", c.ServerID))
	s.SetGameMap("go-sdk-map")

	if err = s.switchQueryProtocol(*c); err != nil {
		return err
	}

	port, _ := c.Port.Int64()
	s.state.Port = uint16(port)

	go s.processInternalEvents()

	// Wait until the internal event processor is ready.
	<-s.internalEventProcessorReady

	// Handle the app starting with an allocation
	if c.AllocatedUUID != "" {
		// Configuration has changed - propagate to consumer. This is optional, so make sure we don't deadlock if
		// nobody is listening.
		select {
		case s.chanConfigurationChanged <- *c:
		default:
		}

		s.chanAllocated <- c.AllocatedUUID
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
	if s.queryBind != nil {
		s.queryBind.Close()
	}

	// Publish a de-allocation message.
	s.chanDeallocated <- ""
	close(s.done)
	s.wg.Wait()

	return nil
}

// OnAllocate returns a read-only channel that receives messages when the server is allocated.
func (s *Server) OnAllocate() <-chan string {
	return s.chanAllocated
}

// OnDeallocate returns a read-only channel that receives messages when the server is de-allocated.
func (s *Server) OnDeallocate() <-chan string {
	return s.chanDeallocated
}

// OnError returns a read-only channel that receives messages when the server encounters an error.
func (s *Server) OnError() <-chan error {
	return s.chanError
}

// OnConfigurationChanged returns a read-only channel that receives messages when the server detects a change in the
// configuration file.
func (s *Server) OnConfigurationChanged() <-chan Config {
	return s.chanConfigurationChanged
}

// PlayerJoined indicates a new player has joined the server.
func (s *Server) PlayerJoined() int32 {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()
	s.state.CurrentPlayers += 1
	return s.state.CurrentPlayers
}

// PlayerLeft indicates a player has left the server.
func (s *Server) PlayerLeft() int32 {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()
	if s.state.CurrentPlayers > 0 {
		s.state.CurrentPlayers -= 1
	}
	return s.state.CurrentPlayers
}

// SetMaxPlayers sets the maximum players this server will host. It does not enforce this number,
// it only serves for query / metrics.
func (s *Server) SetMaxPlayers(max int32) {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()
	s.state.MaxPlayers = max
}

// SetServerName sets the server name for query / metrics purposes.
func (s *Server) SetServerName(name string) {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()
	s.state.ServerName = name
}

// SetGameType sets the server game type for query / metrics purposes.
func (s *Server) SetGameType(gameType string) {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()
	s.state.GameType = gameType
}

// SetGameMap sets the server game map for query / metrics purposes.
func (s *Server) SetGameMap(gameMap string) {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()
	s.state.Map = gameMap
}

// Config returns a copy of the configuration the server is currently using.
func (s *Server) Config() Config {
	s.currentConfigMtx.Lock()
	defer s.currentConfigMtx.Unlock()
	return s.currentConfig
}

// setConfig sets the configuration the server is currently using.
func (s *Server) setConfig(c *Config) {
	s.currentConfigMtx.Lock()
	s.currentConfig = *c
	s.currentConfigMtx.Unlock()

	// Configuration has changed - propagate to consumer. This is optional, so make sure we don't deadlock if
	// nobody is listening.
	select {
	case s.chanConfigurationChanged <- *c:
	default:
	}
}
