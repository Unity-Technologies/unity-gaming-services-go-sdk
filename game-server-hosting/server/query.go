package server

import (
	"errors"
	"fmt"
	"net"

	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server/proto/a2s"
	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server/proto/sqp"
)

type (
	// QueryProtocol represents the query type the server uses.
	// Documentation: https://docs.unity.com/game-server-hosting/en/manual/concepts/query-protocols
	QueryProtocol string
)

const (
	// QueryProtocolA2S represents the 'a2s' query protocol.
	// Although A2S is supported, SQP is the recommended query implementation.
	// Documentation: https://docs.unity.com/game-server-hosting/en/manual/concepts/a2s
	QueryProtocolA2S = QueryProtocol("a2s")

	// QueryProtocolSQP represents the 'sqp' query protocol.
	// SQP is the recommended query protocol.
	// Documentation: https://docs.unity.com/game-server-hosting/en/manual/concepts/sqp
	QueryProtocolSQP = QueryProtocol("sqp")

	// QueryProtocolRecommended represents the recommended query protocol.
	QueryProtocolRecommended
)

// ErrUnsupportedQueryType is an error that specifies the provided query type is not supported by this library.
var ErrUnsupportedQueryType = errors.New("supplied query type is not supported")

// switchQueryProtocol switches to a query protocol specified in the configuration.
// The query binding endpoints are restarted to serve on this endpoint.
func (s *Server) switchQueryProtocol(c Config) error {
	var err error
	switch c.QueryType {
	case QueryProtocolA2S:
		s.queryProto, err = a2s.NewQueryResponder(&s.state)
	case QueryProtocolSQP:
		s.queryProto, err = sqp.NewQueryResponder(&s.state)
	default:
		return ErrUnsupportedQueryType
	}

	if err != nil {
		return err
	}

	return s.restartQueryEndpoint(c)
}

// restartQueryEndpoint restarts the query endpoint to support a potential change of query protocol in the
// configuration.
func (s *Server) restartQueryEndpoint(c Config) error {
	if s.queryBind != nil {
		s.queryBind.Close()
		s.queryBind = nil
	}

	var err error
	s.queryBind, err = newUDPBinding(
		fmt.Sprintf(":%s", c.QueryPort),
		s.queryReadBufferSizeBytes,
		s.queryWriteBufferSizeBytes,
		s.queryReadDeadlineDuration,
		s.queryWriteDeadlineDuration,
	)

	if err != nil {
		return err
	}

	go s.handleQuery()
	return nil
}

// handleQuery handles responding to query commands on an incoming UDP port.
func (s *Server) handleQuery() {
	size := 16

	s.wg.Add(1)
	defer s.wg.Done()

	for {
		buf := make([]byte, size)
		_, to, err := s.queryBind.Read(buf)
		if err != nil {
			if s.queryBind.IsDone() {
				return
			}

			// Ignore timeouts, as reading from the buffer is configured to timeout after a small period of time.
			var netErr net.Error
			if errors.As(err, &netErr) {
				if netErr.Timeout() {
					continue
				}
			}

			s.PushError(fmt.Errorf("query: error reading from socket: %w", err))
			continue
		}

		resp, err := s.queryProto.Respond(to.String(), buf)
		if err != nil {
			s.PushError(fmt.Errorf("query: error responding: %w", err))
			continue
		}

		if _, err = s.queryBind.Write(resp, to); err != nil {
			if s.queryBind.IsDone() {
				return
			}

			s.PushError(fmt.Errorf("query: error writing to socket: %w", err))
			continue
		}
	}
}
