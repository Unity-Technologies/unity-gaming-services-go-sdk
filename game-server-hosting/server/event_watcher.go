package server

import (
	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server/internal/localproxy"
)

// listenForEvents listens for events coming from the local event processor. Currently,
// it listens for allocation and deallocation events and propagates them to the user.
func (s *Server) listenForEvents() {
	s.wg.Add(1)

	cfg := s.Config()
	serverID, _ := cfg.ServerID.Int64()

	localProxyClient, err := localproxy.New(
		cfg.LocalProxyURL,
		serverID,
		s.chanError,
	)
	if err != nil {
		s.eventWatcherReady <- err
		return
	}

	// Watch for allocate and deallocate events if the server handles allocations.
	if s.serverType == TypeAllocation {
		localProxyClient.RegisterCallback(localproxy.AllocateEventType, s.watchAllocation)
		localProxyClient.RegisterCallback(localproxy.DeallocateEventType, s.watchDeallocation)
	}

	if err = localProxyClient.Start(); err != nil {
		s.eventWatcherReady <- err
		return
	}

	// Event watcher is now ready.
	s.eventWatcherReady <- nil

	// Tear down the client on exit.
	defer func() {
		_ = localProxyClient.Stop()
		s.wg.Done()
	}()

	// Wait until server has finished.
	<-s.done
}

// watchAllocation is a callback which propagates the allocation ID to the 'allocated'
// channel when signalled.
func (s *Server) watchAllocation(ev localproxy.Event) {
	if ae, ok := ev.(*localproxy.AllocateEvent); ok {
		s.chanAllocated <- ae.AllocationID
	}
}

// watchDeallocation is a callback which propagates the allocation ID to the 'deallocated'
// channel when signalled.
func (s *Server) watchDeallocation(ev localproxy.Event) {
	if de, ok := ev.(*localproxy.DeallocateEvent); ok {
		s.chanDeallocated <- de.AllocationID
	}
}
