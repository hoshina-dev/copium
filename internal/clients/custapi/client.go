// Package custapi is the HTTP client for the custapi service. It is the
// adapter that satisfies services.UserResolver via structural typing.
//
// All upstream errors map to apperrors sentinels so callers branch with
// errors.Is rather than string comparison.
package custapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hoshina-dev/copium/internal/apperrors"
)

type User struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Name  string    `json:"name"`
}

type Client struct {
	baseURL string
	hc      *http.Client
}

type Option func(*Client)

func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.hc.Timeout = d }
}

func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.hc = h }
}

func New(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		hc:      &http.Client{Timeout: 5 * time.Second},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

func (c *Client) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	url := c.baseURL + "/api/v1/users/id/" + id.String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, apperrors.Internal("custapi: build request", err)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, apperrors.Upstream("custapi: GET "+url, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var u User
		body, _ := io.ReadAll(resp.Body)
		if err := json.Unmarshal(body, &u); err != nil {
			return nil, apperrors.Upstream(fmt.Sprintf("custapi: decode user: body=%q", body), err)
		}
		return &u, nil
	case http.StatusNotFound:
		return nil, apperrors.NotFound(fmt.Sprintf("custapi user %s", id), nil)
	default:
		body, _ := io.ReadAll(resp.Body)
		return nil, apperrors.Upstream(
			fmt.Sprintf("custapi: GET %s -> %d: %s", url, resp.StatusCode, body), nil)
	}
}

func (c *Client) ResolveEmail(ctx context.Context, id uuid.UUID) (string, error) {
	u, err := c.GetUserByID(ctx, id)
	if err != nil {
		return "", err
	}
	return u.Email, nil
}
