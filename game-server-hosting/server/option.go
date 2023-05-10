package server

import "time"

type (
	// Option represents a function that modifies a property of the game server.
	Option func(s *Server)
)

// WithQueryWriteBuffer sets the write buffer size for the query handler.
func WithQueryWriteBuffer(sizeBytes int) Option {
	return func(s *Server) {
		s.queryWriteBufferSizeBytes = sizeBytes
	}
}

// WithQueryReadBuffer sets the read buffer size for the query handler.
func WithQueryReadBuffer(sizeBytes int) Option {
	return func(s *Server) {
		s.queryReadBufferSizeBytes = sizeBytes
	}
}

// WithQueryWriteDeadlineDuration sets the write deadline duration for responding to query requests in the query handler.
func WithQueryWriteDeadlineDuration(duration time.Duration) Option {
	return func(s *Server) {
		s.queryWriteDeadlineDuration = duration
	}
}

// WithConfigPath sets the configuration file to use when starting the server. In most circumstances, the default
// value is reasonable to use.
func WithConfigPath(path string) Option {
	return func(s *Server) {
		s.cfgFile = path
	}
}

// WithHomeDirectory sets the home directory for the server. In most circumstances, the default value is reasonable
// to use.
func WithHomeDirectory(dir string) Option {
	return func(s *Server) {
		s.homeDir = dir
	}
}
