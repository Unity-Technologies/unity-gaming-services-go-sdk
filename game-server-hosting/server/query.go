package server

import (
	"errors"
	"fmt"

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
	// Documentation: https://docs.unity.com/game-server-hosting/en/manual/concepts/a2s
	QueryProtocolA2S = QueryProtocol("a2s")

	// QueryProtocolSQP represents the 'sqp' query protocol.
	// Documentation: https://docs.unity.com/game-server-hosting/en/manual/concepts/sqp
	QueryProtocolSQP = QueryProtocol("sqp")
)

var (
	// ErrUnsupportedQueryType is an error that specifies the provided query type is not supported by this library.Ï"
	ErrUnsupportedQueryType = errors.New("supplied query type is not supported")
)

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
	if s.queryBind, err = newUDPBinding(fmt.Sprintf(":%s", c.QueryPort)); err != nil {
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

			s.pushError(fmt.Errorf("query: error reading from socket: %w", err))
			continue
		}

		resp, err := s.queryProto.Respond(to.String(), buf)
		if err != nil {
			s.pushError(fmt.Errorf("query: error responding: %w", err))
			continue
		}

		if _, err = s.queryBind.Write(resp, to); err != nil {
			if s.queryBind.IsDone() {
				return
			}

			s.pushError(fmt.Errorf("query: error writing to socket: %w", err))
			continue
		}
	}
}
