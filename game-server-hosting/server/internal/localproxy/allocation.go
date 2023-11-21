package localproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server/model"
	"github.com/google/uuid"
)

// PatchAllocation triggers the local proxy endpoint to patch this server allocation.
func (c *Client) PatchAllocation(ctx context.Context, allocationID string, args *model.PatchAllocationRequest) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	buf := bytes.NewBuffer(nil)
	if err := json.NewEncoder(buf).Encode(args); err != nil {
		return fmt.Errorf("error encoding args: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPatch,
		fmt.Sprintf("%s/v1/servers/%d/allocations/%s", c.host, c.serverID, allocationID),
		buf,
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
