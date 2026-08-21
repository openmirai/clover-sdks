// Package clover provides a small, dependency-free Clover API client.
package clover

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type JSON map[string]any

type Problem struct {
	Type        string                     `json:"type"`
	Title       string                     `json:"title"`
	Status      int                        `json:"status"`
	Code        string                     `json:"code"`
	Detail      *string                    `json:"detail,omitempty"`
	RequestID   string                     `json:"request_id"`
	FieldErrors map[string][]string        `json:"field_errors,omitempty"`
	Extra       map[string]json.RawMessage `json:"-"`
}

func (p *Problem) UnmarshalJSON(data []byte) error {
	type alias Problem
	var decoded struct {
		*alias
		Extra map[string]json.RawMessage `json:"-"`
	}
	decoded.alias = (*alias)(p)
	if err := json.Unmarshal(data, &decoded.alias); err != nil {
		return err
	}
	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		return err
	}
	for _, key := range []string{"type", "title", "status", "code", "detail", "request_id", "field_errors"} {
		delete(all, key)
	}
	p.Extra = all
	return nil
}

type ResponseMeta struct {
	RequestID          string
	RetryAfter         time.Duration
	RetryAfterSet      bool
	RateLimitRemaining int
	Replayed           bool
}

type ErrorDetail struct {
	Code    int               `json:"code"`
	Type    string            `json:"type"`
	Message string            `json:"message"`
	Details string            `json:"details,omitempty"`
	Fields  map[string]string `json:"fields,omitempty"`
}

type Error struct {
	Status  int
	Problem *Problem
	Detail  *ErrorDetail
	Meta    ResponseMeta
	Message string
}

func (e *Error) Error() string { return e.Message }

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

const DefaultMaxResponseBodyBytes = 4 << 20

var idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$`)

type Client struct {
	BaseURL              string
	APIKey               string
	UserAgent            string
	HTTPClient           Doer
	MaxRetries           int
	RetryBaseDelay       time.Duration
	MaxResponseBodyBytes int
	Sleep                func(context.Context, time.Duration) error
}

func NewClient(baseURL, apiKey string) *Client {
	baseURL = strings.TrimSpace(baseURL)
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		panic("baseURL must be an absolute http(s) URL without userinfo/query/fragment")
	}
	if strings.TrimSpace(apiKey) == "" {
		panic("apiKey is required")
	}
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), APIKey: apiKey, UserAgent: "clover-sdk-go/0.1.0", HTTPClient: http.DefaultClient, MaxRetries: 2, RetryBaseDelay: 100 * time.Millisecond, MaxResponseBodyBytes: DefaultMaxResponseBodyBytes}
}

func (c *Client) Send(ctx context.Context, request JSON, idempotencyKey string) (JSON, ResponseMeta, error) {
	return c.request(ctx, http.MethodPost, "/api/v1/emails", request, idempotencyKey)
}

func (c *Client) SendBatch(ctx context.Context, items []JSON, idempotencyKey string) (JSON, ResponseMeta, error) {
	sanitized := make([]JSON, 0, len(items))
	for _, item := range items {
		copyItem := JSON{}
		for key, value := range item {
			if key != "scheduled_at" {
				copyItem[key] = value
			}
		}
		sanitized = append(sanitized, copyItem)
	}
	return c.request(ctx, http.MethodPost, "/api/v1/emails/batch", JSON{"items": sanitized}, idempotencyKey)
}

func (c *Client) Schedule(ctx context.Context, id, scheduledAt, idempotencyKey string) (JSON, ResponseMeta, error) {
	return c.request(ctx, http.MethodPost, "/api/v1/emails/"+url.PathEscape(id)+"/schedule", JSON{"scheduled_at": scheduledAt}, idempotencyKey)
}

func (c *Client) Cancel(ctx context.Context, id, idempotencyKey string) (JSON, ResponseMeta, error) {
	return c.request(ctx, http.MethodPost, "/api/v1/emails/"+url.PathEscape(id)+"/cancel", nil, idempotencyKey)
}

func (c *Client) Get(ctx context.Context, id string) (JSON, ResponseMeta, error) {
	return c.request(ctx, http.MethodGet, "/api/v1/emails/"+url.PathEscape(id), nil, "")
}

func (c *Client) List(ctx context.Context, query url.Values) (JSON, ResponseMeta, error) {
	path := "/api/v1/emails"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	return c.request(ctx, http.MethodGet, path, nil, "")
}

func (c *Client) request(ctx context.Context, method, path string, payload JSON, idempotencyKey string) (JSON, ResponseMeta, error) {
	if method != http.MethodGet && !idempotencyKeyPattern.MatchString(idempotencyKey) {
		return nil, ResponseMeta{}, errors.New("idempotency key must match ^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$")
	}
	body, err := json.Marshal(payload)
	if payload == nil {
		body = nil
	}
	if err != nil {
		return nil, ResponseMeta{}, err
	}
	safe := method == http.MethodGet || idempotencyKey != ""
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
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("User-Agent", c.UserAgent)
		req.Header.Set("X-Request-ID", "req_"+strconv.FormatInt(time.Now().UnixNano(), 36)+strconv.Itoa(attempt))
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if idempotencyKey != "" {
			req.Header.Set("Idempotency-Key", idempotencyKey)
		}
		client := c.HTTPClient
		if client == nil {
			client = http.DefaultClient
		}
		response, err := client.Do(req)
		if err != nil {
			return nil, ResponseMeta{}, err
		}
		maxResponseBodyBytes := c.MaxResponseBodyBytes
		if maxResponseBodyBytes <= 0 {
			maxResponseBodyBytes = DefaultMaxResponseBodyBytes
		}
		data, readErr := io.ReadAll(io.LimitReader(response.Body, int64(maxResponseBodyBytes)+1))
		response.Body.Close()
		if readErr != nil {
			return nil, ResponseMeta{}, readErr
		}
		meta := responseMeta(response.Header)
		if len(data) > maxResponseBodyBytes {
			return nil, meta, &Error{Status: response.StatusCode, Meta: meta, Message: "Clover response body exceeds the configured limit"}
		}
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			var result JSON
			if len(data) > 0 {
				if err := json.Unmarshal(data, &result); err != nil {
					return nil, meta, err
				}
			}
			if success, ok := result["success"].(bool); ok {
				if !success {
					return nil, meta, decodeAPIError(response.StatusCode, data, meta)
				}
				if requestID, ok := result["requestId"].(string); ok && meta.RequestID == "" {
					meta.RequestID = requestID
				}
				if payload, ok := result["data"].(map[string]any); ok {
					return JSON(payload), meta, nil
				}
				return JSON{}, meta, nil
			}
			return result, meta, nil
		}
		if safe && isRetryable(response.StatusCode) && attempt < maxRetries {
			delay := meta.RetryAfter
			if !meta.RetryAfterSet {
				delay = c.RetryBaseDelay * time.Duration(1<<attempt)
			}
			sleep := c.Sleep
			if sleep == nil {
				sleep = func(ctx context.Context, d time.Duration) error {
					timer := time.NewTimer(d)
					defer timer.Stop()
					select {
					case <-timer.C:
						return nil
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			}
			if err := sleep(ctx, delay); err != nil {
				return nil, meta, err
			}
			continue
		}
		return nil, meta, decodeAPIError(response.StatusCode, data, meta)
	}
}

func decodeAPIError(status int, data []byte, meta ResponseMeta) *Error {
	var envelope struct {
		Success   bool         `json:"success"`
		Error     *ErrorDetail `json:"error"`
		RequestID string       `json:"requestId"`
	}
	if json.Unmarshal(data, &envelope) == nil && envelope.Error != nil && envelope.Error.Message != "" {
		if meta.RequestID == "" {
			meta.RequestID = envelope.RequestID
		}
		return &Error{
			Status:  status,
			Detail:  envelope.Error,
			Problem: &Problem{Title: envelope.Error.Message, Status: status, Code: strconv.Itoa(envelope.Error.Code), Type: envelope.Error.Type, RequestID: meta.RequestID},
			Meta:    meta,
			Message: envelope.Error.Message,
		}
	}
	var problem Problem
	var problemPtr *Problem
	if json.Unmarshal(data, &problem) == nil && problem.Title != "" {
		problemPtr = &problem
	}
	message := fmt.Sprintf("Clover request failed (%d)", status)
	if problemPtr != nil {
		message = problemPtr.Title
	}
	return &Error{Status: status, Problem: problemPtr, Meta: meta, Message: message}
}

func responseMeta(headers http.Header) ResponseMeta {
	meta := ResponseMeta{RequestID: headers.Get("X-Request-ID"), Replayed: strings.EqualFold(headers.Get("Idempotency-Replayed"), "true")}
	if value, err := strconv.Atoi(headers.Get("X-RateLimit-Remaining")); err == nil {
		meta.RateLimitRemaining = value
	}
	if values := headers.Values("Retry-After"); len(values) > 0 {
		if value, err := strconv.Atoi(values[0]); err == nil && value >= 0 {
			meta.RetryAfter = time.Duration(value) * time.Second
			meta.RetryAfterSet = true
		}
	}
	return meta
}

func isRetryable(status int) bool {
	switch status {
	case 408, 425, 429, 500, 502, 503, 504:
		return true
	}
	return false
}
