// Package iamclient implements middleware.Authorizer over HTTP, calling
// lms-iam's one decision endpoint (ADR-012) before a destructive /books
// write goes through. Same internal-HS256-token shape every other
// service-to-service client in this project already uses (see
// lms-circulation-api's catalogclient/membershipclient).
package iamclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Client struct {
	baseURL   string
	jwtSecret string
	http      *http.Client
}

func NewClient(baseURL, jwtSecret string) *Client {
	return &Client{baseURL: baseURL, jwtSecret: jwtSecret, http: &http.Client{Timeout: 5 * time.Second}}
}

type authorizeResponse struct {
	Allowed bool `json:"allowed"`
}

// Authorize calls GET /api/v1/iam/authorize?administratorId=&permission=
// (ADR-012). lms-iam itself already defaults to allowed=true for an
// administrator with no role assignment row — this client has no default
// of its own, it only relays whatever lms-iam decides.
func (c *Client) Authorize(ctx context.Context, administratorID, permission string) (bool, error) {
	token, err := c.mintServiceToken()
	if err != nil {
		return false, fmt.Errorf("minting internal service token: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/v1/iam/authorize?administratorId=%s&permission=%s",
		c.baseURL, url.QueryEscape(administratorID), url.QueryEscape(permission))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.http.Do(req)
	if err != nil {
		return false, fmt.Errorf("calling iam-service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("iam-service returned %d", resp.StatusCode)
	}

	var body authorizeResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return false, fmt.Errorf("decoding iam-service response: %w", err)
	}
	return body.Allowed, nil
}

// mintServiceToken signs a short-lived internal token with
// INTERNAL_JWT_SECRET, sub=catalog-service — the identity lms-iam's own
// RequireCaller allow-list already permits on /authorize (ADR-013).
func (c *Client) mintServiceToken() (string, error) {
	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"sub": "catalog-service",
		"iat": now.Unix(),
		"exp": now.Add(1 * time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(c.jwtSecret))
}
