package clover

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

type LogListOptions struct {
	Page      int
	Limit     int
	Cursor    string
	RequestID string
	Operation string
	StatusMin *int
	StatusMax *int
	Source    string
	APIKeyID  string
	Query     url.Values
}

func (o LogListOptions) values() url.Values {
	v := PageOptions{Page: o.Page, Limit: o.Limit, Cursor: o.Cursor, Query: o.Query}.values()
	if o.RequestID != "" {
		v.Set("request_id", o.RequestID)
	}
	if o.Operation != "" {
		v.Set("operation", o.Operation)
	}
	if o.StatusMin != nil {
		v.Set("status_min", formatInt(*o.StatusMin))
	}
	if o.StatusMax != nil {
		v.Set("status_max", formatInt(*o.StatusMax))
	}
	if o.Source != "" {
		v.Set("source", o.Source)
	}
	if o.APIKeyID != "" {
		v.Set("api_key_id", o.APIKeyID)
	}
	return v
}

func formatInt(value int) string {
	return strconv.Itoa(value)
}

// MetricsService exposes event-derived email metrics.
type MetricsService struct{ client *Client }

type MetricsOptions struct {
	Start     string
	End       string
	Interval  string
	DomainID  string
	EventType string
	Query     url.Values
}

func (o MetricsOptions) values() url.Values {
	v := cloneValues(o.Query)
	for key, value := range map[string]string{"start": o.Start, "end": o.End, "interval": o.Interval, "domain_id": o.DomainID, "event_type": o.EventType} {
		if value != "" {
			v.Set(key, value)
		}
	}
	return v
}

func (s *MetricsService) Email(ctx context.Context, options MetricsOptions) (EmailMetrics, ResponseMeta, error) {
	var result EmailMetrics
	meta, err := requestTyped(s.client, ctx, http.MethodGet, withValues("/api/v1/metrics/email", options.values()), nil, "", &result)
	return result, meta, err
}

// LogsService reads the sanitized request audit log.
type LogsService struct{ client *Client }

func (s *LogsService) List(ctx context.Context, options LogListOptions) (Page[RequestLog], ResponseMeta, error) {
	var result Page[RequestLog]
	meta, err := requestTyped(s.client, ctx, http.MethodGet, withValues("/api/v1/logs", options.values()), nil, "", &result)
	return result, meta, err
}

type InboundListOptions struct {
	Page        int
	Limit       int
	Cursor      string
	DomainID    string
	ParseStatus string
	Query       url.Values
}

func (o InboundListOptions) values() url.Values {
	v := PageOptions{Page: o.Page, Limit: o.Limit, Cursor: o.Cursor, Query: o.Query}.values()
	if o.DomainID != "" {
		v.Set("domain_id", o.DomainID)
	}
	if o.ParseStatus != "" {
		v.Set("parse_status", o.ParseStatus)
	}
	return v
}

// InboundService reads parsed inbound mail and accepts provider callbacks.
// The callback method is included for deterministic local adapters; providers
// normally call this endpoint directly rather than through the customer SDK.
type InboundService struct{ client *Client }

func (s *InboundService) List(ctx context.Context, options InboundListOptions) (Page[InboundEmail], ResponseMeta, error) {
	var result Page[InboundEmail]
	meta, err := requestTyped(s.client, ctx, http.MethodGet, withValues("/api/v1/received-emails", options.values()), nil, "", &result)
	return result, meta, err
}

func (s *InboundService) Get(ctx context.Context, id string) (InboundEmail, ResponseMeta, error) {
	var result InboundEmail
	meta, err := requestTyped(s.client, ctx, http.MethodGet, "/api/v1/received-emails/"+segment(id), nil, "", &result)
	return result, meta, err
}

func (s *InboundService) AttachmentURL(ctx context.Context, emailID, attachmentID string) (InboundAttachmentURL, ResponseMeta, error) {
	var result InboundAttachmentURL
	path := "/api/v1/received-emails/" + segment(emailID) + "/attachments/" + segment(attachmentID)
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

func (s *InboundService) Accept(ctx context.Context, provider string, payload JSON, idempotencyKeys ...string) (JSON, ResponseMeta, error) {
	var result JSON
	idempotencyKey, err := optionalIdempotencyKey(idempotencyKeys)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithPolicy(s.client, ctx, http.MethodPost, "/api/v1/inbound/"+segment(provider), payload, idempotencyKey, &result, true)
	return result, meta, err
}

// ProviderEventsService is an alias-friendly callback surface for adapters
// that use the generic provider-events route instead of inbound mail.
type ProviderEventsService struct{ client *Client }

func (s *ProviderEventsService) Accept(ctx context.Context, provider string, payload JSON, idempotencyKeys ...string) (JSON, ResponseMeta, error) {
	var result JSON
	idempotencyKey, err := optionalIdempotencyKey(idempotencyKeys)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithPolicy(s.client, ctx, http.MethodPost, "/api/v1/provider-events/"+segment(provider), payload, idempotencyKey, &result, true)
	return result, meta, err
}

type SuppressionListOptions struct {
	Page          int
	Limit         int
	Cursor        string
	Active        *bool
	Reason        string
	AddressSHA256 string
	Query         url.Values
}

func (o SuppressionListOptions) values() url.Values {
	v := PageOptions{Page: o.Page, Limit: o.Limit, Cursor: o.Cursor, Query: o.Query}.values()
	if o.Active != nil {
		v.Set("active", boolString(*o.Active))
	}
	if o.Reason != "" {
		v.Set("reason", o.Reason)
	}
	if o.AddressSHA256 != "" {
		v.Set("address_sha256", o.AddressSHA256)
	}
	return v
}

// SuppressionsService manages recipient suppressions. Delete is the backend's
// reactivation operation; Reactivate is provided as an intention-revealing
// alias and intentionally uses the same canonical DELETE path.
type SuppressionsService struct{ client *Client }

func (s *SuppressionsService) List(ctx context.Context, options SuppressionListOptions) (Page[Suppression], ResponseMeta, error) {
	var result Page[Suppression]
	meta, err := requestTyped(s.client, ctx, http.MethodGet, withValues("/api/v1/suppressions", options.values()), nil, "", &result)
	return result, meta, err
}

func (s *SuppressionsService) Create(ctx context.Context, request CreateSuppressionRequest, idempotencyKeys ...string) (Suppression, ResponseMeta, error) {
	var result Suppression
	idempotencyKey, err := optionalIdempotencyKey(idempotencyKeys)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithPolicy(s.client, ctx, http.MethodPost, "/api/v1/suppressions", request, idempotencyKey, &result, true)
	return result, meta, err
}

func (s *SuppressionsService) Delete(ctx context.Context, id string, idempotencyKeys ...string) (ResponseMeta, error) {
	idempotencyKey, err := optionalIdempotencyKey(idempotencyKeys)
	if err != nil {
		return ResponseMeta{}, err
	}
	return requestTypedWithPolicy(s.client, ctx, http.MethodDelete, "/api/v1/suppressions/"+segment(id), nil, idempotencyKey, (*JSON)(nil), true)
}

func (s *SuppressionsService) Reactivate(ctx context.Context, id string, idempotencyKeys ...string) (ResponseMeta, error) {
	return s.Delete(ctx, id, idempotencyKeys...)
}

// PreferencesService covers dashboard preference topics plus public tokenized
// preference-center and one-click unsubscribe URLs.
type PreferencesService struct{ client *Client }

func (s *PreferencesService) Topics(ctx context.Context) (Page[PreferenceTopic], ResponseMeta, error) {
	var result Page[PreferenceTopic]
	meta, err := requestTyped(s.client, ctx, http.MethodGet, "/api/v1/preference-topics", nil, "", &result)
	return result, meta, err
}

func (s *PreferencesService) CreateTopic(ctx context.Context, request CreatePreferenceTopicRequest, idempotencyKeys ...string) (PreferenceTopic, ResponseMeta, error) {
	var result PreferenceTopic
	idempotencyKey, err := optionalIdempotencyKey(idempotencyKeys)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithPolicy(s.client, ctx, http.MethodPost, "/api/v1/preference-topics", request, idempotencyKey, &result, true)
	return result, meta, err
}

func (s *PreferencesService) Get(ctx context.Context, token string) (PreferenceCenter, ResponseMeta, error) {
	var result PreferenceCenter
	meta, err := requestTyped(s.client, ctx, http.MethodGet, "/api/v1/preferences/"+segment(token), nil, "", &result)
	return result, meta, err
}

func (s *PreferencesService) Update(ctx context.Context, token string, request UpdatePreferenceRequest, idempotencyKeys ...string) (PreferenceCenter, ResponseMeta, error) {
	var result PreferenceCenter
	idempotencyKey, err := optionalIdempotencyKey(idempotencyKeys)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithPolicy(s.client, ctx, http.MethodPost, "/api/v1/preferences/"+segment(token), request, idempotencyKey, &result, true)
	return result, meta, err
}

func (s *PreferencesService) Unsubscribe(ctx context.Context, token string, idempotencyKeys ...string) (ResponseMeta, error) {
	idempotencyKey, err := optionalIdempotencyKey(idempotencyKeys)
	if err != nil {
		return ResponseMeta{}, err
	}
	return requestTypedWithPolicy(s.client, ctx, http.MethodPost, "/api/v1/unsubscribe/"+segment(token), nil, idempotencyKey, (*JSON)(nil), true)
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
