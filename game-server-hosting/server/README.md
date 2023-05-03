# Unity Game Server Hosting Server Library

A Golang game server can be integrated with the platform quickly by using this library. A fully-formed usage example can be found [here](https://github.com/Unity-Technologies/multiplay-examples/tree/main/simple-game-server-go).

## Short Demonstration

A basic server can be set up by using the `server.New()` function with the `Start()`, `Stop()` and message channels:

```go
package main

import "github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server"

func main() {
	s, err := server.New(server.TypeAllocation)
	if err != nil {
		// ...
	}

	if err = s.Start(); err != nil {
		// ...
	}

	// Handle server events (i.e. allocation, deallocation, errors)
	done := make(chan struct{})
	go handleEvents(s, done)

	if err = s.WaitUntilTerminated(); err != nil {
		close(done)
		// ...
	}
}

func handleEvents(s *server.Server, done chan struct{}) {
	for {
		select {
		case <-s.OnAllocate():
			// handle allocation

		case <-s.OnDeallocate():
			// handle deallocation

		case <-s.OnError():
			// handle error

		case <-s.OnConfigurationChanged():
			// handle configuration change

		case <-done:
			return
		}
	}
}
```
