package localproxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// UnreserveSelf triggers the local proxy endpoint to unreserve this server instance. Only applicable for reservation-based fleets.
func (c *Client) UnreserveSelf(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodDelete,
		fmt.Sprintf("%s/v1/servers/%d/reservations", c.host, c.serverID),
		bytes.NewBufferString(`{}`),
	)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	// Add a request ID - if we cannot generate a UUID for any reason, just populate an empty one.
	requestID, err := uuid.NewUUID()
	if err != nil {
		requestID = uuid.UUID{}
	}

	req.Header.Add("X-Request-ID", requestID.String())

	var resp *http.Response
	httpClient := &http.Client{}
	if resp, err = httpClient.Do(req); err != nil {
		return fmt.Errorf("error making request: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusNoContent {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return NewUnexpectedResponseWithError(requestID.String(), resp.StatusCode, readErr)
		}
		return NewUnexpectedResponseWithBody(requestID.String(), resp.StatusCode, body)
	}

	return nil
}
