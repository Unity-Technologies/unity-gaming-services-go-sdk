package localproxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// ReleaseSelf triggers the local proxy endpoint to release the hold on this server instance.
func (c *Client) ReleaseSelf(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodDelete,
		fmt.Sprintf("%s/v1/servers/%d/hold", c.host, c.serverID),
		http.NoBody,
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
	if resp, err = c.httpClient.Do(req); err != nil {
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
