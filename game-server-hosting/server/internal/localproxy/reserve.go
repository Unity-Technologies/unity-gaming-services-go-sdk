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

// ReserveSelf triggers the local proxy endpoint to reserve this server instance. Only applicable for reservation-based fleets.
func (c *Client) ReserveSelf(ctx context.Context, args *model.ReserveRequest) (*model.ReserveResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	buf := bytes.NewBuffer(nil)
	if err := json.NewEncoder(buf).Encode(args); err != nil {
		return nil, fmt.Errorf("error encoding args: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/v1/servers/%d/reservations", c.host, c.serverID),
		buf,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
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
		return nil, fmt.Errorf("error making request: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, NewUnexpectedResponseWithError(requestID.String(), resp.StatusCode, err)
		}
		return nil, NewUnexpectedResponseWithBody(requestID.String(), resp.StatusCode, body)
	}

	var body *model.ReserveResponse
	if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return body, nil
}
