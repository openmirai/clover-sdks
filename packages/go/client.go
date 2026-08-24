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
	// RawBody retains bounded GET response bytes for forward-compatible
	// inspection when a server adds fields that this SDK version does not yet
	// model. Mutation bodies are intentionally not retained because they may
	// contain one-time API-key or webhook secrets.
	RawBody []byte
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
const DefaultMaxRequestBodyBytes = 4 << 20

var idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$`)

// ErrInvalidIdempotencyKey is returned before transport when a mutation does
// not carry the canonical Clover idempotency key format.
var ErrInvalidIdempotencyKey = errors.New("idempotency key must match ^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$")

// ErrRequestBodyTooLarge is returned before transport when a caller attempts
// to send a payload larger than the configured bounded request body.
var ErrRequestBodyTooLarge = errors.New("Clover request body exceeds the configured limit")

func ValidateIdempotencyKey(key string) error {
	if !idempotencyKeyPattern.MatchString(key) {
		return ErrInvalidIdempotencyKey
	}
	return nil
}

type Client struct {
	BaseURL              string
	APIKey               string
	UserAgent            string
	HTTPClient           Doer
	MaxRetries           int
	RetryBaseDelay       time.Duration
	MaxRequestBodyBytes  int
	MaxResponseBodyBytes int
	Sleep                func(context.Context, time.Duration) error

	// Typed resource services. The original flat email methods remain
	// available for compatibility; new integrations should prefer these
	// namespaced services.
	Emails          *EmailsService
	Domains         *DomainsService
	DomainHealth    *DomainHealthService
	Routing         *RoutingService
	ProviderRouting *RoutingService
	Templates       *TemplatesService
	Webhooks        *WebhooksService
	APIKeys         *APIKeysService
	Logs            *LogsService
	Metrics         *MetricsService
	Inbound         *InboundService
	ProviderEvents  *ProviderEventsService
	Suppressions    *SuppressionsService
	Preferences     *PreferencesService
	Contacts        *ContactsService
	Segments        *SegmentsService
	Broadcasts      *BroadcastsService
	Automations     *AutomationsService
	Audit           *AuditService
	Platform        *PlatformService
	Accounts        *PlatformAccountsService
	Environments    *PlatformEnvironmentsService
	Messages        *PlatformMessagesService
	SMTP            *SMTPService
	Usage           *UsageContractService
	// Platform* aliases keep the account/environment services discoverable for
	// callers that prefer flat fields while Platform remains the canonical
	// namespace.
	PlatformAccounts         *PlatformAccountsService
	PlatformEnvironments     *PlatformEnvironmentsService
	PlatformMessages         *PlatformMessagesService
	PlatformTemplates        *PlatformTemplatesService
	PlatformWebhooks         *PlatformWebhooksService
	PlatformTimeline         *MessageTimelineService
	PlatformLogs             *MessageTimelineService
	PlatformInbound          *PlatformInboundService
	PlatformPreferences      *PlatformPreferencesService
	PlatformSuppressions     *PlatformSuppressionsService
	PlatformSMTP             *SMTPService
	PlatformContacts         *PlatformContactsService
	PlatformSegments         *PlatformSegmentsService
	PlatformAutomations      *PlatformAutomationsService
	PlatformRouting          *PlatformRoutingService
	PlatformDomains          *PlatformDomainsService
	PlatformDomainHealth     *PlatformDomainHealthService
	PlatformUsage            *UsageContractService
	PlatformAudiences        *PlatformAudiencesService
	PlatformBroadcasts       *PlatformBroadcastsService
	PlatformProviderBindings *PlatformProviderBindingsService
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
	return newClient(baseURL, apiKey)
}

// NewPublicClient creates a client for tokenized public preference-center and
// one-click unsubscribe routes. It deliberately omits Authorization; use
// NewClient for any authenticated resource.
func NewPublicClient(baseURL string) *Client {
	baseURL = strings.TrimSpace(baseURL)
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		panic("baseURL must be an absolute http(s) URL without userinfo/query/fragment")
	}
	return newClient(baseURL, "")
}

func newClient(baseURL, apiKey string) *Client {
	client := &Client{BaseURL: strings.TrimRight(baseURL, "/"), APIKey: apiKey, UserAgent: "clover-sdk-go/0.1.0", HTTPClient: http.DefaultClient, MaxRetries: 2, RetryBaseDelay: 100 * time.Millisecond, MaxRequestBodyBytes: DefaultMaxRequestBodyBytes, MaxResponseBodyBytes: DefaultMaxResponseBodyBytes}
	client.Emails = &EmailsService{client: client}
	client.Domains = &DomainsService{client: client}
	client.DomainHealth = &DomainHealthService{client: client}
	client.Routing = &RoutingService{client: client}
	client.ProviderRouting = client.Routing
	client.Templates = &TemplatesService{client: client}
	client.Webhooks = &WebhooksService{client: client}
	client.APIKeys = &APIKeysService{client: client}
	client.Logs = &LogsService{client: client}
	client.Metrics = &MetricsService{client: client}
	client.Inbound = &InboundService{client: client}
	client.ProviderEvents = &ProviderEventsService{client: client}
	client.Suppressions = &SuppressionsService{client: client}
	client.Preferences = &PreferencesService{client: client}
	client.Contacts = &ContactsService{client: client}
	client.Segments = &SegmentsService{client: client}
	client.Broadcasts = &BroadcastsService{client: client}
	client.Automations = &AutomationsService{client: client}
	client.Audit = &AuditService{client: client}
	client.Platform = newPlatformService(client)
	client.Accounts = client.Platform.Accounts
	client.Environments = client.Platform.Environments
	client.Messages = client.Platform.Messages
	client.SMTP = client.Platform.SMTP
	client.Usage = client.Platform.Usage
	client.PlatformAccounts = client.Platform.Accounts
	client.PlatformEnvironments = client.Platform.Environments
	client.PlatformMessages = client.Platform.Messages
	client.PlatformTemplates = client.Platform.Templates
	client.PlatformWebhooks = client.Platform.Webhooks
	client.PlatformTimeline = client.Platform.Timeline
	client.PlatformLogs = client.Platform.Logs
	client.PlatformInbound = client.Platform.Inbound
	client.PlatformPreferences = client.Platform.Preferences
	client.PlatformSuppressions = client.Platform.Suppressions
	client.PlatformSMTP = client.Platform.SMTP
	client.PlatformContacts = client.Platform.Contacts
	client.PlatformSegments = client.Platform.Segments
	client.PlatformAutomations = client.Platform.Automations
	client.PlatformRouting = client.Platform.Routing
	client.PlatformDomains = client.Platform.Domains
	client.PlatformDomainHealth = client.Platform.DomainHealth
	client.PlatformUsage = client.Platform.Usage
	client.PlatformAudiences = client.Platform.Audiences
	client.PlatformBroadcasts = client.Platform.Broadcasts
	client.PlatformProviderBindings = client.Platform.ProviderBindings
	return client
}

func (c *Client) Send(ctx context.Context, request JSON, idempotencyKey string) (JSON, ResponseMeta, error) {
	return c.request(ctx, http.MethodPost, "/api/v1/emails", request, idempotencyKey)
}

func (c *Client) SendBatch(ctx context.Context, items []JSON, idempotencyKey string) (JSON, ResponseMeta, error) {
	// Preserve the caller's payload exactly.  In particular, scheduled_at is
	// part of the backend batch contract and must be validated by the server;
	// silently dropping it here made an otherwise invalid request appear valid.
	return c.request(ctx, http.MethodPost, "/api/v1/emails/batch", JSON{"items": items}, idempotencyKey)
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
	var result JSON
	meta, err := requestTyped(c, ctx, method, path, payload, idempotencyKey, &result)
	return result, meta, err
}

// requestTyped performs one API request and decodes the CommonResponse data
// member into result. The function deliberately lives outside Client because
// Go does not permit methods with their own type parameters. Resource services
// use it to retain a typed API without introducing a third-party transport.
func requestTyped[T any](c *Client, ctx context.Context, method, path string, payload any, idempotencyKey string, result *T) (ResponseMeta, error) {
	return requestTypedWithPolicy(c, ctx, method, path, payload, idempotencyKey, result, method == http.MethodGet)
}

// requestTypedWithHeaders is the bounded transport seam used by provider
// callbacks and other integrations that must attach verified request
// metadata. Header values are copied into each retry attempt.
func requestTypedWithHeaders[T any](c *Client, ctx context.Context, method, path string, payload any, idempotencyKey string, result *T, allowMissingIdempotency bool, headers http.Header) (ResponseMeta, error) {
	return requestTypedWithPolicyAndHeaders(c, ctx, method, path, payload, idempotencyKey, result, allowMissingIdempotency, headers)
}

// requestTypedWithoutIdempotency is used only for side-effect-free POST
// endpoints such as segment evaluation. The backend does not put an
// idempotency middleware in front of those operations.
func requestTypedWithoutIdempotency[T any](c *Client, ctx context.Context, method, path string, payload any, result *T) (ResponseMeta, error) {
	return requestTypedWithPolicy(c, ctx, method, path, payload, "", result, true)
}

func requestTypedWithPolicy[T any](c *Client, ctx context.Context, method, path string, payload any, idempotencyKey string, result *T, allowMissingIdempotency bool) (ResponseMeta, error) {
	return requestTypedWithPolicyAndHeaders(c, ctx, method, path, payload, idempotencyKey, result, allowMissingIdempotency, nil)
}

func requestTypedWithPolicyAndHeaders[T any](c *Client, ctx context.Context, method, path string, payload any, idempotencyKey string, result *T, allowMissingIdempotency bool, headers http.Header) (ResponseMeta, error) {
	if !allowMissingIdempotency {
		if err := ValidateIdempotencyKey(idempotencyKey); err != nil {
			return ResponseMeta{}, err
		}
	}
	body, err := json.Marshal(payload)
	if payload == nil {
		body = nil
	}
	if err != nil {
		return ResponseMeta{}, err
	}
	maxRequestBodyBytes := c.MaxRequestBodyBytes
	if maxRequestBodyBytes <= 0 {
		maxRequestBodyBytes = DefaultMaxRequestBodyBytes
	}
	if len(body) > maxRequestBodyBytes {
		return ResponseMeta{}, ErrRequestBodyTooLarge
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
			return ResponseMeta{}, err
		}
		req.Header.Set("Accept", "application/json")
		if c.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+c.APIKey)
		}
		req.Header.Set("User-Agent", c.UserAgent)
		for key, values := range headers {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
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
			return ResponseMeta{}, err
		}
		maxResponseBodyBytes := c.MaxResponseBodyBytes
		if maxResponseBodyBytes <= 0 {
			maxResponseBodyBytes = DefaultMaxResponseBodyBytes
		}
		data, readErr := io.ReadAll(io.LimitReader(response.Body, int64(maxResponseBodyBytes)+1))
		response.Body.Close()
		if readErr != nil {
			return ResponseMeta{}, readErr
		}
		meta := responseMeta(response.Header)
		// RawBody exists to let callers inspect additive fields on safe read
		// responses.  Do not retain successful mutation bodies: API-key and
		// webhook creation responses can contain one-time secrets.
		if method == http.MethodGet {
			meta.RawBody = append([]byte(nil), data...)
		}
		if len(data) > maxResponseBodyBytes {
			return meta, &Error{Status: response.StatusCode, Meta: meta, Message: "Clover response body exceeds the configured limit"}
		}
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			if len(data) == 0 || result == nil {
				return meta, nil
			}
			var envelope struct {
				Success        *bool           `json:"success"`
				Data           json.RawMessage `json:"data"`
				RequestID      string          `json:"requestId"`
				RequestIDSnake string          `json:"request_id"`
			}
			if err := json.Unmarshal(data, &envelope); err != nil {
				return meta, err
			}
			if envelope.Success != nil {
				if !*envelope.Success {
					return meta, decodeAPIError(response.StatusCode, data, meta)
				}
				if meta.RequestID == "" {
					meta.RequestID = envelope.RequestID
					if meta.RequestID == "" {
						meta.RequestID = envelope.RequestIDSnake
					}
				}
				if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
					return meta, nil
				}
				if err := json.Unmarshal(envelope.Data, result); err != nil {
					return meta, err
				}
				return meta, nil
			}
			if err := json.Unmarshal(data, result); err != nil {
				return meta, err
			}
			return meta, nil
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
				return meta, err
			}
			continue
		}
		return meta, decodeAPIError(response.StatusCode, data, meta)
	}
}

func decodeAPIError(status int, data []byte, meta ResponseMeta) *Error {
	var envelope struct {
		Success        bool         `json:"success"`
		Error          *ErrorDetail `json:"error"`
		RequestID      string       `json:"requestId"`
		RequestIDSnake string       `json:"request_id"`
	}
	if json.Unmarshal(data, &envelope) == nil && envelope.Error != nil && envelope.Error.Message != "" {
		if meta.RequestID == "" {
			meta.RequestID = envelope.RequestID
			if meta.RequestID == "" {
				meta.RequestID = envelope.RequestIDSnake
			}
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
