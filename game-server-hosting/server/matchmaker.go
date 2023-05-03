package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type (
	// tokenResponse is the representation of a token and an error from the payload proxy service.
	tokenResponse struct {
		Token string `json:"token"`
		Error string `json:"error"`
	}
)

var (
	errTokenFetch      = errors.New("failed to retrieve JWT token")
	errBackfillApprove = errors.New("failed to approve backfill ticket")
)

func (s *Server) keepAliveBackfill(c *Config) {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			resp, err := s.approveBackfillTicket(c)
			if err != nil {
				s.pushError(fmt.Errorf("error approving backfill ticket: %w", err))
			} else {
				_ = resp.Body.Close()
			}
		case <-s.done:
			ticker.Stop()

			return
		}
	}
}

// approveBackfillTicket is called in a loop to update and keep the backfill ticket alive.
func (s *Server) approveBackfillTicket(c *Config) (*http.Response, error) {
	token, err := s.getJwtToken(c)
	if err != nil {
		return nil, err
	}

	resp, err := s.updateBackfillAllocation(c, token)
	if err != nil {
		s.pushError(fmt.Errorf("error updating backfill allocation: %w", err))
	}
	if resp == nil || resp.StatusCode != http.StatusOK {
		err = errBackfillApprove
	}

	return resp, err
}

// getJwtToken calls the local proxy token endpoint to retrieve the token used for matchmaker backfill approval.
func (s *Server) getJwtToken(c *Config) (string, error) {
	localProxyTokenURL := fmt.Sprintf("%s/token", c.LocalProxyURL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, localProxyTokenURL, http.NoBody)
	if err != nil {
		return "", err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errTokenFetch
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tr tokenResponse
	err = json.Unmarshal(bodyBytes, &tr)

	if err != nil {
		return "", err
	}

	if len(tr.Error) != 0 {
		return "", errTokenFetch
	}

	return tr.Token, nil
}

// updateBackfillAllocation calls the matchmaker backfill approval endpoint to update and keep the backfill ticket
// alive.
func (s *Server) updateBackfillAllocation(c *Config, token string) (*http.Response, error) {
	backfillApprovalURL := fmt.Sprintf("%s/v2/backfill/%s/approvals",
		c.MatchmakerURL,
		c.AllocatedUUID,
	)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, backfillApprovalURL, http.NoBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	return s.httpClient.Do(req)
}
