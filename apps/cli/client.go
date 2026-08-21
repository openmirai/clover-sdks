package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type Problem map[string]any
type ResponseMeta struct {
	RequestID          string
	RetryAfter         time.Duration
	RetryAfterSet      bool
	RateLimitRemaining int
	Replayed           bool
}
type APIError struct {
	Status   int
	Problem  Problem
	Metadata ResponseMeta
	Message  string
}

func (e *APIError) Error() string { return e.Message }

type Client struct {
	BaseURL, APIKey, UserAgent string
	HTTPClient                 *http.Client
	MaxRetries                 int
	RetryBaseDelay             time.Duration
	MaxResponseBodyBytes       int
	Sleep                      func(context.Context, time.Duration) error
}

const DefaultMaxResponseBodyBytes = 4 << 20

var idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$`)

func NewClient(baseURL, apiKey string) *Client {
	baseURL = strings.TrimSpace(baseURL)
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		panic("baseURL must be an absolute http(s) URL without userinfo/query/fragment")
	}
	if strings.TrimSpace(apiKey) == "" {
		panic("apiKey is required")
	}
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), APIKey: apiKey, UserAgent: "clover-cli/0.1.0", HTTPClient: http.DefaultClient, MaxRetries: 2, RetryBaseDelay: 100 * time.Millisecond, MaxResponseBodyBytes: DefaultMaxResponseBodyBytes}
}
func (c *Client) Send(ctx context.Context, payload map[string]any, key string) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, "POST", "/api/v1/emails", payload, key)
}
func (c *Client) SendBatch(ctx context.Context, items []map[string]any, key string) (json.RawMessage, ResponseMeta, error) {
	sanitized := make([]map[string]any, 0, len(items))
	for _, item := range items {
		copyItem := map[string]any{}
		for name, value := range item {
			if name != "scheduled_at" {
				copyItem[name] = value
			}
		}
		sanitized = append(sanitized, copyItem)
	}
	return c.request(ctx, "POST", "/api/v1/emails/batch", map[string]any{"items": sanitized}, key)
}
func (c *Client) Schedule(ctx context.Context, id, scheduledAt, key string) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, "POST", "/api/v1/emails/"+url.PathEscape(id)+"/schedule", map[string]any{"scheduled_at": scheduledAt}, key)
}
func (c *Client) Get(ctx context.Context, id string) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, "GET", "/api/v1/emails/"+url.PathEscape(id), nil, "")
}
func (c *Client) List(ctx context.Context, query url.Values) (json.RawMessage, ResponseMeta, error) {
	path := "/api/v1/emails"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	return c.request(ctx, "GET", path, nil, "")
}
func (c *Client) Cancel(ctx context.Context, id, key string) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, "POST", "/api/v1/emails/"+url.PathEscape(id)+"/cancel", nil, key)
}

func (c *Client) StreamEvents(ctx context.Context, path string, onEvent func(json.RawMessage) error) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("User-Agent", c.UserAgent)
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, overflow, _ := readBounded(response.Body, c.maxResponseBodyBytes())
		if overflow {
			return &APIError{Status: response.StatusCode, Metadata: metadata(response.Header), Message: "Clover stream response body exceeds the configured limit"}
		}
		var problem Problem
		_ = json.Unmarshal(body, &problem)
		return &APIError{Status: response.StatusCode, Problem: problem, Metadata: metadata(response.Header), Message: fmt.Sprintf("Clover stream failed (%d)", response.StatusCode)}
	}
	limited := &io.LimitedReader{R: response.Body, N: int64(c.maxResponseBodyBytes()) + 1}
	scanner := bufio.NewScanner(limited)
	scanner.Buffer(make([]byte, 64*1024), c.maxResponseBodyBytes()+1)
	var data strings.Builder
	var events []json.RawMessage
	flush := func() error {
		if data.Len() == 0 {
			return nil
		}
		events = append(events, json.RawMessage(data.String()))
		data.Reset()
		return nil
	}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if err := flush(); err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if err := scanner.Err(); err != nil {
		if limited.N <= 0 {
			return &APIError{Status: response.StatusCode, Metadata: metadata(response.Header), Message: "Clover stream response body exceeds the configured limit"}
		}
		return err
	}
	if limited.N <= 0 {
		return &APIError{Status: response.StatusCode, Metadata: metadata(response.Header), Message: "Clover stream response body exceeds the configured limit"}
	}
	if err := flush(); err != nil {
		return err
	}
	for _, event := range events {
		if err := onEvent(event); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) request(ctx context.Context, method, path string, payload map[string]any, key string) (json.RawMessage, ResponseMeta, error) {
	if method != "GET" && !idempotencyKeyPattern.MatchString(key) {
		return nil, ResponseMeta{}, errors.New("idempotency key must match ^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$")
	}
	body, err := json.Marshal(payload)
	if payload == nil {
		body = nil
	}
	if err != nil {
		return nil, ResponseMeta{}, err
	}
	safe := method == "GET" || key != ""
	maxRetries := c.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}
	if maxRetries > 3 {
		maxRetries = 3
	}
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bytes.NewReader(body))
		if err != nil {
			return nil, ResponseMeta{}, err
		}
		req.Header.Set("Accept", "application/json, application/problem+json")
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("User-Agent", c.UserAgent)
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if key != "" {
			req.Header.Set("Idempotency-Key", key)
		}
		client := c.HTTPClient
		if client == nil {
			client = http.DefaultClient
		}
		response, err := client.Do(req)
		if err != nil {
			return nil, ResponseMeta{}, err
		}
		bytesBody, overflow, readErr := readBounded(response.Body, c.maxResponseBodyBytes())
		response.Body.Close()
		if readErr != nil {
			return nil, ResponseMeta{}, readErr
		}
		meta := metadata(response.Header)
		if overflow {
			return nil, meta, &APIError{Status: response.StatusCode, Metadata: meta, Message: "Clover response body exceeds the configured limit"}
		}
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			unwrapped, err := unwrapEnvelope(bytesBody)
			if err != nil {
				return nil, meta, err
			}
			return unwrapped, meta, nil
		}
		if safe && retryable(response.StatusCode) && attempt < maxRetries {
			delay := meta.RetryAfter
			if !meta.RetryAfterSet {
				delay = c.RetryBaseDelay * time.Duration(1<<attempt)
			}
			sleeper := c.Sleep
			if sleeper == nil {
				sleeper = func(ctx context.Context, delay time.Duration) error {
					timer := time.NewTimer(delay)
					defer timer.Stop()
					select {
					case <-timer.C:
						return nil
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			}
			if err := sleeper(ctx, delay); err != nil {
				return nil, meta, err
			}
			continue
		}
		var envelope struct {
			Success bool           `json:"success"`
			Error   map[string]any `json:"error"`
		}
		message := fmt.Sprintf("Clover request failed (%d)", response.StatusCode)
		var problem Problem
		if json.Unmarshal(bytesBody, &envelope) == nil && envelope.Error != nil {
			if msg, ok := envelope.Error["message"].(string); ok {
				message = msg
			}
			problem = envelope.Error
		} else {
			_ = json.Unmarshal(bytesBody, &problem)
			if title, ok := problem["title"].(string); ok {
				message = title
			}
		}
		return nil, meta, &APIError{Status: response.StatusCode, Problem: problem, Metadata: meta, Message: message}
	}
}

func unwrapEnvelope(body []byte) ([]byte, error) {
	var envelope map[string]any
	if err := json.Unmarshal(body, &envelope); err != nil {
		return body, nil
	}
	success, ok := envelope["success"].(bool)
	if !ok {
		return body, nil
	}
	if !success {
		return body, nil
	}
	data, ok := envelope["data"]
	if !ok || data == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(data)
}

func (c *Client) maxResponseBodyBytes() int {
	if c.MaxResponseBodyBytes > 0 {
		return c.MaxResponseBodyBytes
	}
	return DefaultMaxResponseBodyBytes
}

func readBounded(body io.Reader, limit int) ([]byte, bool, error) {
	data, err := io.ReadAll(io.LimitReader(body, int64(limit)+1))
	return data, len(data) > limit, err
}
func retryable(status int) bool {
	switch status {
	case 408, 425, 429, 500, 502, 503, 504:
		return true
	}
	return false
}
func metadata(headers http.Header) ResponseMeta {
	meta := ResponseMeta{RequestID: headers.Get("X-Request-ID"), Replayed: strings.EqualFold(headers.Get("Idempotency-Replayed"), "true")}
	if values := headers.Values("Retry-After"); len(values) > 0 {
		if value, err := time.ParseDuration(values[0] + "s"); err == nil && value >= 0 {
			meta.RetryAfter = value
			meta.RetryAfterSet = true
		}
	}
	if value, err := fmt.Sscan(headers.Get("X-RateLimit-Remaining"), &meta.RateLimitRemaining); err != nil || value == 0 {
		meta.RateLimitRemaining = 0
	}
	return meta
}
