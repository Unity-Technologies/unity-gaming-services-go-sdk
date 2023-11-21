package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server/internal/localproxy"
	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server/model"
	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server/proto"
	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server/proto/sqp"
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

		// eventWatcherReady is the channel that, when written to, indicates that the event watcher is ready.
		eventWatcherReady chan error

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
		queryReadDeadlineDuration  time.Duration

		// Local proxy
		localProxyClient *localproxy.Client

		// Synchronisation
		done chan struct{}
		wg   sync.WaitGroup
	}
)

const (
	// TypeAllocation represents a server which is using the 'allocations' model of server usage.
	TypeAllocation = Type(iota)

	// TypeReservation represents a server which is using the 'reservations' model of server usage.
	TypeReservation
)

const (
	// DefaultWriteBufferSizeBytes represents the default size of the write buffer for the query handler.
	DefaultWriteBufferSizeBytes = 1024

	// DefaultWriteDeadlineDuration represents the default write deadline duration for responding in the query handler.
	DefaultWriteDeadlineDuration = 1 * time.Second

	// DefaultReadDeadlineDuration represents the default read deadline duration for consuming a query request.
	DefaultReadDeadlineDuration = 3 * time.Second

	// DefaultReadBufferSizeBytes represents the default size of the read buffer for the query handler.
	DefaultReadBufferSizeBytes = 1024
)

var (
	// ErrOperationNotApplicable represents that the operation being performed is not applicable to the server type.
	ErrOperationNotApplicable = errors.New("the operation requested is not applicable to the server type")

	// ErrNilContext represents that the context supplied is nil.
	ErrNilContext = errors.New("context is nil")

	// ErrNilArgs represents that the arguments supplied are nil.
	ErrNilArgs = errors.New("arguments supplied are nil")

	// ErrMetricsUnsupported represents that the query type this server is using does not support additional metrics.
	ErrMetricsUnsupported = errors.New("metrics are not supported for this query type")

	// ErrMetricOutOfBounds represents that the metric index provided will overflow the metrics buffer.
	ErrMetricOutOfBounds = errors.New("metrics index provided will overflow the metrics buffer")
)

// New creates a new instance of Server, denoting which type of server to use.
func New(serverType Type, opts ...Option) (*Server, error) {
	dir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("error getting user home directory: %w", err)
	}

	s := &Server{
		serverType:                  serverType,
		cfgFile:                     filepath.Join(dir, "server.json"),
		chanAllocated:               make(chan string, 1),
		chanDeallocated:             make(chan string, 1),
		chanError:                   make(chan error, 1),
		chanConfigurationChanged:    make(chan Config, 1),
		internalEventProcessorReady: make(chan struct{}, 1),
		eventWatcherReady:           make(chan error, 1),
		done:                        make(chan struct{}, 1),
		queryWriteBufferSizeBytes:   DefaultWriteBufferSizeBytes,
		queryWriteDeadlineDuration:  DefaultWriteDeadlineDuration,
		queryReadBufferSizeBytes:    DefaultReadBufferSizeBytes,
		queryReadDeadlineDuration:   DefaultReadDeadlineDuration,
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
	c, err := newConfigFromFile(s.cfgFile)
	if err != nil {
		return err
	}

	s.setConfig(c)

	// Create the directory the logs will be present in.
	if err = os.MkdirAll(c.ServerLogDir, 0o744); err != nil {
		return fmt.Errorf("error creating log directory: %w", err)
	}

	// Set some defaults for the query endpoint. These can get overwritten by the user, but best to set some defaults
	// to keep friction to a minimum.
	s.SetServerName(fmt.Sprintf("go-sdk-server - %s", c.ServerID))
	s.SetGameMap("go-sdk-map")

	// Set up metrics buffer, if supported.
	if c.QueryType == QueryProtocolSQP {
		s.state.Metrics = make([]float32, 0, sqp.MaxMetrics)
	}

	if err = s.switchQueryProtocol(*c); err != nil {
		return err
	}

	port, _ := c.Port.Int64()
	s.state.Port = uint16(port)

	go s.watchForConfigChanges()
	go s.listenForEvents()

	// Wait until the internal event processor is ready.
	<-s.internalEventProcessorReady

	// Wait until the event watcher is ready.
	if err = <-s.eventWatcherReady; err != nil {
		return fmt.Errorf("error configuring event watcher: %w", err)
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

// Reserve reserves this server for use. Only applicable to reservation-based fleets.
func (s *Server) Reserve(ctx context.Context, args *model.ReserveRequest) (*model.ReserveResponse, error) {
	// Operation is only applicable to reservation-based fleets, so return an error otherwise.
	if s.serverType != TypeReservation {
		return nil, ErrOperationNotApplicable
	}

	if ctx == nil {
		return nil, ErrNilContext
	}

	if args == nil {
		return nil, ErrNilArgs
	}

	return s.localProxyClient.ReserveSelf(ctx, args)
}

// Unreserve unreserves this server, making it available for use. Only applicable to reservation-based fleets.
func (s *Server) Unreserve(ctx context.Context) error {
	// Operation is only applicable to reservation-based fleets, so return an error otherwise.
	if s.serverType != TypeReservation {
		return ErrOperationNotApplicable
	}

	if ctx == nil {
		return ErrNilContext
	}

	return s.localProxyClient.UnreserveSelf(ctx)
}

// Hold holds this server, preventing descaling until after a reservation completes, the expiry time elapses,
// or the hold is manually released with the Release() method.
func (s *Server) Hold(ctx context.Context, args *model.HoldRequest) (*model.HoldStatus, error) {
	if ctx == nil {
		return nil, ErrNilContext
	}
	return s.localProxyClient.HoldSelf(ctx, args)
}

// HoldStatus gets the status of the hold for the server, including the time at which the hold expires.
func (s *Server) HoldStatus(ctx context.Context) (*model.HoldStatus, error) {
	if ctx == nil {
		return nil, ErrNilContext
	}
	return s.localProxyClient.HoldStatus(ctx)
}

// Release manually releases any existing holds for this server.
func (s *Server) Release(ctx context.Context) error {
	if ctx == nil {
		return ErrNilContext
	}
	return s.localProxyClient.ReleaseSelf(ctx)
}

// ReadyForPlayers indicates the server is ready for players to join.
func (s *Server) ReadyForPlayers(ctx context.Context) error {
	if ctx == nil {
		return ErrNilContext
	}

	allocationID := s.currentConfig.AllocatedUUID
	patch := &model.PatchAllocationRequest{
		Ready: true,
	}

	return s.localProxyClient.PatchAllocation(ctx, allocationID, patch)
}

// PlayerJoined indicates a new player has joined the server.
func (s *Server) PlayerJoined() int32 {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()
	s.state.CurrentPlayers++
	return s.state.CurrentPlayers
}

// PlayerLeft indicates a player has left the server.
func (s *Server) PlayerLeft() int32 {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()
	if s.state.CurrentPlayers > 0 {
		s.state.CurrentPlayers--
	}
	return s.state.CurrentPlayers
}

// SetCurrentPlayers sets the number of players currently in the game. Can be used as an alternative to PlayerJoined
// and PlayerLeft.
func (s *Server) SetCurrentPlayers(players int32) {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()

	if players < 0 {
		players = 0
	}

	s.state.CurrentPlayers = players
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

// SetMetric sets the metric at index to provided value. Only supported if the query type is QueryProtocolSQP, otherwise,
// ErrMetricsUnsupported is returned. The maximum index is specified by sqp.MaxMetrics, any index supplied above this
// will return ErrMetricOutOfBounds.
func (s *Server) SetMetric(index byte, value float32) error {
	if s.Config().QueryType != QueryProtocolSQP {
		return ErrMetricsUnsupported
	}

	s.stateLock.Lock()
	defer s.stateLock.Unlock()

	if index >= sqp.MaxMetrics {
		return ErrMetricOutOfBounds
	}

	// Expand slice to fit new index if needed.
	if int(index) >= len(s.state.Metrics) {
		s.state.Metrics = s.state.Metrics[:index+1]
	}

	s.state.Metrics[index] = value
	return nil
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
