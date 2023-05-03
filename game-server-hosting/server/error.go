package server

// pushError pushes an error to a channel consumer. Listening for errors is optional, so this makes sure we don't deadlock
// if nobody is listening.
func (s *Server) pushError(err error) {
	select {
	case s.chanError <- err:
	default:
	}
}
