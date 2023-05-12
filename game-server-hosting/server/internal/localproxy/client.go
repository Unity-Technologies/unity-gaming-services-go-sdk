package localproxy

import (
	"fmt"
	"strings"
	"time"

	"github.com/centrifugal/centrifuge-go"
)

// Client represents a client to the local proxy.
type Client struct {
	centrifugeClient *centrifuge.Client
	sub              *centrifuge.Subscription
	serverID         int64
	callbacks        map[EventType]func(Event)
	done             chan struct{}
	chanSubscribed   chan struct{}
	chanError        chan<- error
}

// New constructs a new instance of the local proxy client.
func New(host string, serverID int64, chanError chan<- error) (*Client, error) {
	hostWithoutProtocol := strings.ReplaceAll(host, "http://", "")
	return &Client{
		centrifugeClient: centrifuge.NewJsonClient(
			fmt.Sprintf("ws://%s/v1/connection/websocket", hostWithoutProtocol),
			centrifuge.Config{},
		),
		serverID:       serverID,
		callbacks:      map[EventType]func(Event){},
		done:           make(chan struct{}),
		chanSubscribed: make(chan struct{}),
		chanError:      chanError,
	}, nil
}

// Start subscribes to the centrifuge broker and connects to it. Start() blocks until the client has subscribed successfully.
func (c *Client) Start() error {
	if subscribeErr := c.subscribe(); subscribeErr != nil {
		return subscribeErr
	}

	if connectErr := c.centrifugeClient.Connect(); connectErr != nil {
		return connectErr
	}

	// Wait for the client to be subscribed before continuing.
	<-c.chanSubscribed
	return nil
}

// Stop stops the client.
func (c *Client) Stop() error {
	c.centrifugeClient.Close()
	close(c.done)
	return nil
}

// RegisterCallback registers a callback function for the specified EventType.
func (c *Client) RegisterCallback(ev EventType, cb func(Event)) {
	c.callbacks[ev] = cb
}

// OnPublish implements centrifuge.PublicationHandler and is triggered when a message is published to this subscriber.
func (c *Client) OnPublish(e centrifuge.PublicationEvent) {
	event, err := unmarshalEvent(e.Data)
	if err != nil {
		select {
		case c.chanError <- err:
		default:
		}
		return
	}

	// Trigger the callback if one has been registered.
	if cb, ok := c.callbacks[event.Type()]; ok {
		cb(event)
	}
}

// OnError implements centrifuge.SubscriptionErrorHandler and is triggered when an error is encountered with this subscriber.
func (c *Client) OnError(_ centrifuge.SubscriptionErrorEvent) {
	// Retry connecting to the SDK daemon. In some cases the server may be
	// attempting to connect before the SDK daemon has registered the existence
	// of the server.
	select {
	case <-c.done:
		return
	default:
		time.Sleep(1 * time.Second)

		if err := c.subscribe(); err != nil {
			select {
			case c.chanError <- err:
			default:
			}
		}
	}
}

// OnSubscribed implements centrifuge.SubscribedHandler and is triggered when the client has successfully subscribed
// to the broker.
func (c *Client) OnSubscribed(_ centrifuge.SubscribedEvent) {
	c.chanSubscribed <- struct{}{}
}

// subscribe creates a new subscription to the centrifuge broker and sets up relevant callbacks.
func (c *Client) subscribe() error {
	var subErr error
	if c.sub, subErr = c.centrifugeClient.NewSubscription(fmt.Sprintf("server#%d", c.serverID)); subErr != nil {
		return subErr
	}

	c.sub.OnSubscribed(c.OnSubscribed)
	c.sub.OnPublication(c.OnPublish)
	c.sub.OnError(c.OnError)

	return c.sub.Subscribe()
}
