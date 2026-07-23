package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"cloud-client/internal/config"
	"cloud-client/pkg/logger"
)

var (
	ErrUnableToContactCloud = errors.New("Unable to contact Cloud.")
	ErrInvalidToken         = errors.New("Invalid token.")
	ErrConnectionExpired    = errors.New("Connection expired.")
)

type CloudClient interface {
	Connect(ctx context.Context, token string) (*ConnectResponse, error)
	Confirm(ctx context.Context, connectionID string) (*ConfirmResponse, error)
}

type Client struct {
	baseURL    string
	httpClient *http.Client
	userAgent  string
	logger     *logger.Logger
}

func NewClient(cfg *config.Config, log *logger.Logger) *Client {
	return &Client{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		userAgent: cfg.UserAgent,
		logger:    log,
	}
}

func NewClientWithHTTP(baseURL string, httpClient *http.Client, userAgent string, log *logger.Logger) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
		userAgent:  userAgent,
		logger:     log,
	}
}

func (c *Client) Connect(ctx context.Context, token string) (*ConnectResponse, error) {
	reqBody := ConnectRequest{Token: token}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal connect request: %w", err)
	}

	url := c.baseURL + "/client/connect"
	if c.logger != nil {
		c.logger.Debug("HTTP POST %s", url)
		c.logger.Debug("Request Body: %s", string(bodyBytes))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnableToContactCloud, err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Debug("HTTP request error: %v", err)
		}
		return nil, fmt.Errorf("%w: %v", ErrUnableToContactCloud, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if c.logger != nil {
		c.logger.Debug("HTTP Status: %d", resp.StatusCode)
		c.logger.Debug("Response Body: %s", string(respBody))
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, ErrInvalidToken
	}

	if resp.StatusCode == http.StatusGone {
		return nil, ErrConnectionExpired
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp ErrorResponse
		_ = json.Unmarshal(respBody, &errResp)
		if errResp.Message != "" {
			return nil, fmt.Errorf("%w: %s", ErrUnableToContactCloud, errResp.Message)
		}
		if errResp.Error != "" {
			return nil, fmt.Errorf("%w: %s", ErrUnableToContactCloud, errResp.Error)
		}
		return nil, fmt.Errorf("%w: %s returned HTTP status %d", ErrUnableToContactCloud, url, resp.StatusCode)
	}

	var rawResp ConnectApiResponse
	if err := json.Unmarshal(respBody, &rawResp); err != nil {
		return nil, fmt.Errorf("%w: invalid JSON response: %v", ErrUnableToContactCloud, err)
	}

	connectResp := rawResp.ConnectResponse
	if rawResp.Connection != nil {
		connectResp = *rawResp.Connection
	}

	return &connectResp, nil
}

func (c *Client) Confirm(ctx context.Context, connectionID string) (*ConfirmResponse, error) {
	reqBody := ConfirmRequest{ConnectionID: connectionID}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal confirm request: %w", err)
	}

	url := c.baseURL + "/client/confirm"
	if c.logger != nil {
		c.logger.Debug("HTTP POST %s", url)
		c.logger.Debug("Request Body: %s", string(bodyBytes))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnableToContactCloud, err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Debug("HTTP request error: %v", err)
		}
		return nil, fmt.Errorf("%w: %v", ErrUnableToContactCloud, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if c.logger != nil {
		c.logger.Debug("HTTP Status: %d", resp.StatusCode)
		c.logger.Debug("Response Body: %s", string(respBody))
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp ErrorResponse
		_ = json.Unmarshal(respBody, &errResp)
		if errResp.Message != "" {
			return nil, fmt.Errorf("%w: %s", ErrUnableToContactCloud, errResp.Message)
		}
		return nil, fmt.Errorf("%w: %s returned HTTP status %d", ErrUnableToContactCloud, url, resp.StatusCode)
	}

	var confirmResp ConfirmResponse
	if err := json.Unmarshal(respBody, &confirmResp); err != nil {
		return nil, fmt.Errorf("%w: invalid JSON response: %v", ErrUnableToContactCloud, err)
	}

	return &confirmResp, nil
}
