package server

import (
	"fmt"

	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server/internal/localproxy"
)

// listenForEvents listens for events coming from the local event processor. Currently,
// it listens for allocation and deallocation events and propagates them to the user.
func (s *Server) listenForEvents() {
	s.wg.Add(1)

	cfg := s.Config()
	serverID, err := cfg.ServerID.Int64()
	if err != nil {
		s.eventWatcherReady <- fmt.Errorf("error parsing server ID: %w", err)
		return
	}

	s.localProxyClient, err = localproxy.New(
		cfg.LocalProxyURL,
		serverID,
		s.chanError,
	)
	if err != nil {
		s.eventWatcherReady <- fmt.Errorf("error creating local proxy client: %w", err)
		return
	}

	// Watch for allocate and deallocate events if the server handles allocations.
	if s.serverType == TypeAllocation {
		s.localProxyClient.RegisterCallback(localproxy.AllocateEventType, s.watchAllocation)
		s.localProxyClient.RegisterCallback(localproxy.DeallocateEventType, s.watchDeallocation)
	}

	if err = s.localProxyClient.Start(); err != nil {
		s.eventWatcherReady <- err
		return
	}

	// Event watcher is now ready.
	s.eventWatcherReady <- nil

	// Tear down the client on exit.
	defer func() {
		defer s.wg.Done()
		_ = s.localProxyClient.Stop()
	}()

	// Wait until server has finished.
	<-s.done
}

// watchAllocation is a callback which propagates the allocation ID to the 'allocated'
// channel when signalled.
func (s *Server) watchAllocation(ev localproxy.Event) {
	if ae, ok := ev.(*localproxy.AllocateEvent); ok {
		s.allocatedUUIDMtx.Lock()
		s.allocatedUUID = ae.AllocationID
		s.allocatedUUIDMtx.Unlock()

		s.chanAllocated <- ae.AllocationID
	}
}

// watchDeallocation is a callback which propagates the allocation ID to the 'deallocated'
// channel when signalled.
func (s *Server) watchDeallocation(ev localproxy.Event) {
	if de, ok := ev.(*localproxy.DeallocateEvent); ok {
		s.allocatedUUIDMtx.Lock()
		s.allocatedUUID = de.AllocationID
		s.allocatedUUIDMtx.Unlock()

		s.chanDeallocated <- de.AllocationID
	}
}
