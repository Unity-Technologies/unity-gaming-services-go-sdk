package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	gsh "github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server"
)

type (
	// backfillWrapFunc is an alias for a function which can be wrapped by `wrapWithConfigAndJWT()`
	backfillWrapFunc func(c *gsh.Config, token string) (*BackfillTicket, error)

	// tokenResponse is the representation of a token and an error from the payload proxy service.
	tokenResponse struct {
		Token string `json:"token"`
		Error string `json:"error"`
	}
)

var (
	errTokenFetch = errors.New("failed to retrieve JWT token")
)

// keepAliveBackfill keeps the backfill ticket alive for the current allocation.
func (s *Server) keepAliveBackfill() {
	s.wg.Add(1)
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			if _, err := s.wrapWithConfigAndJWT(s.approveBackfillTicket); err != nil {
				s.PushError(fmt.Errorf("error approving backfill ticket: %w", err))
			}
		case <-s.done:
			ticker.Stop()
			return
		}
	}
}

// wrapWithConfigAndJWT accepts a function which requires server configuration and a JWT and requests those pieces of data
// before calling the function.
func (s *Server) wrapWithConfigAndJWT(f backfillWrapFunc) (*BackfillTicket, error) {
	c := s.Config()
	token, err := s.getJwtToken(&c)
	if err != nil {
		return nil, err
	}

	return f(&c, token)
}

// approveBackfillTicket calls the matchmaker backfill approval endpoint to update and keep the backfill ticket alive.
// Documentation: https://services.docs.unity.com/matchmaker/v2/index.html#tag/Backfill/operation/approveBackfillTicket
func (s *Server) approveBackfillTicket(c *gsh.Config, token string) (*BackfillTicket, error) {
	// Don't attempt to approve the ticket if the server is not allocated.
	if c.AllocatedUUID == "" {
		return nil, ErrNotAllocated
	}

	backfillApprovalURL := fmt.Sprintf(
		"%s/v2/backfill/%s/approvals",
		matchmakerURL(*c),
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

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request to matchmaker: %w", err)
	}

	if resp == nil || resp.StatusCode != http.StatusOK {
		return nil, ErrBackfillApprove
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var ticket *BackfillTicket
	if err = json.NewDecoder(resp.Body).Decode(&ticket); err != nil {
		return nil, fmt.Errorf("error decoding backfill ticket response: %w", err)
	}

	return ticket, nil
}

// getJwtToken calls the local proxy token endpoint to retrieve the token used for matchmaker backfill approval.
func (s *Server) getJwtToken(c *gsh.Config) (string, error) {
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

// backfillEnabled returns a boolean representation of the `enableBackfill` configuration item.
func backfillEnabled(c gsh.Config) bool {
	b, _ := strconv.ParseBool(c.Extra["enableBackfill"])
	return b
}

// matchmakerURL returns the matchmaker URL, or a default if empty.
func matchmakerURL(c gsh.Config) string {
	u := c.Extra["matchmakerUrl"]
	if u == "" {
		return "https://matchmaker.services.api.unity.com"
	}

	return u
}
