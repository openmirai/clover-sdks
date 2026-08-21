package clover

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// PageOptions is shared by cursor- and page-based list endpoints. Query holds
// additional server-supported filters and is copied before use.
type PageOptions struct {
	Page   int
	Limit  int
	Cursor string
	Query  url.Values
}

func (o PageOptions) values() url.Values {
	v := cloneValues(o.Query)
	if o.Page != 0 {
		v.Set("page", strconv.Itoa(o.Page))
	}
	if o.Limit != 0 {
		v.Set("limit", strconv.Itoa(o.Limit))
	}
	if o.Cursor != "" {
		v.Set("cursor", o.Cursor)
	}
	return v
}

func cloneValues(source url.Values) url.Values {
	result := make(url.Values, len(source))
	for key, values := range source {
		result[key] = append([]string(nil), values...)
	}
	return result
}

func withValues(path string, values url.Values) string {
	// Merge rather than blindly appending a second question mark. Scoped
	// routes are often built first and then receive endpoint-specific filters
	// such as status or from/to selectors.
	if index := strings.IndexByte(path, '?'); index >= 0 {
		existing, err := url.ParseQuery(path[index+1:])
		if err == nil {
			for key, values := range values {
				existing[key] = append([]string(nil), values...)
			}
			path = path[:index]
			values = existing
		}
	}
	if encoded := values.Encode(); encoded != "" {
		return path + "?" + encoded
	}
	return path
}

func segment(value string) string { return url.PathEscape(value) }

// ScopedPageOptions carries the required native lifecycle scope query along
// with standard pagination. Additional query filters can be supplied through
// Query without requiring an SDK release for every additive backend filter.
type ScopedPageOptions struct {
	Page        int
	Limit       int
	Cursor      string
	TenantID    string
	ProjectID   string
	Environment string
	Query       url.Values
}

var (
	// ErrMissingScope indicates that a native full-product operation was
	// attempted without its required project/environment (and, for list
	// endpoints, tenant) boundary.
	ErrMissingScope = errors.New("project_id and environment are required")
	// ErrInvalidScope indicates malformed scope values that can be rejected
	// before a request reaches the transport.
	ErrInvalidScope = errors.New("invalid scope")
)

// Validate checks the scope values required by native full-product routes.
// TenantID is required by lifecycle endpoints; aggregate domain-health,
// reliability, and audit endpoints may intentionally omit it.
func (o ScopedPageOptions) Validate(requireTenant bool) error {
	if strings.TrimSpace(o.ProjectID) == "" || strings.TrimSpace(o.Environment) == "" {
		return fmt.Errorf("%w: project_id and environment must be non-empty", ErrMissingScope)
	}
	switch strings.TrimSpace(o.Environment) {
	case "development", "preview", "staging", "production":
	default:
		return fmt.Errorf("%w: environment must be development, preview, staging, or production", ErrInvalidScope)
	}
	if requireTenant && strings.TrimSpace(o.TenantID) == "" {
		return fmt.Errorf("%w: tenant_id must be non-empty", ErrMissingScope)
	}
	if o.Page < 0 || o.Limit < 0 || o.Limit > 100 {
		return fmt.Errorf("%w: page and limit must be non-negative and limit must be at most 100", ErrInvalidScope)
	}
	if len(o.Cursor) > 512 {
		return fmt.Errorf("%w: cursor is too long", ErrInvalidScope)
	}
	return nil
}

// Validate checks a request body scope. Creation endpoints require a tenant
// because the body establishes ownership rather than inheriting it from a
// query string.
func (s Scope) Validate(requireTenant bool) error {
	if strings.TrimSpace(s.OrganizationID) == "" {
		return fmt.Errorf("%w: organization_id must be non-empty for request-body scopes", ErrMissingScope)
	}
	return (ScopedPageOptions{TenantID: s.TenantID, ProjectID: s.ProjectID, Environment: s.Environment}).Validate(requireTenant)
}

func scopedPathChecked(path string, options ScopedPageOptions, requireTenant bool) (string, error) {
	if err := options.Validate(requireTenant); err != nil {
		return "", err
	}
	return withValues(path, options.values()), nil
}

func scopeOption(options []ScopedPageOptions, requireTenant bool) (ScopedPageOptions, error) {
	if len(options) == 0 {
		return ScopedPageOptions{}, ErrMissingScope
	}
	if len(options) != 1 {
		return ScopedPageOptions{}, fmt.Errorf("%w: exactly one scope option is required", ErrInvalidScope)
	}
	if err := options[0].Validate(requireTenant); err != nil {
		return ScopedPageOptions{}, err
	}
	return options[0], nil
}

func optionalIdempotencyKey(keys []string) (string, error) {
	if len(keys) > 1 {
		return "", fmt.Errorf("%w: at most one idempotency key is accepted", ErrInvalidIdempotencyKey)
	}
	if len(keys) == 0 || strings.TrimSpace(keys[0]) == "" {
		return "", nil
	}
	if err := ValidateIdempotencyKey(keys[0]); err != nil {
		return "", err
	}
	return keys[0], nil
}

func (o ScopedPageOptions) values() url.Values {
	v := PageOptions{Page: o.Page, Limit: o.Limit, Cursor: o.Cursor, Query: o.Query}.values()
	if o.TenantID != "" {
		v.Set("tenant_id", o.TenantID)
	}
	if o.ProjectID != "" {
		v.Set("project_id", o.ProjectID)
	}
	if o.Environment != "" {
		v.Set("environment", o.Environment)
	}
	return v
}

type DomainListOptions struct {
	Page             int
	Limit            int
	Status           string
	Provider         string
	SendingEnabled   *bool
	ReceivingEnabled *bool
	Query            url.Values
}

func (o DomainListOptions) values() url.Values {
	v := PageOptions{Page: o.Page, Limit: o.Limit, Query: o.Query}.values()
	if o.Status != "" {
		v.Set("status", o.Status)
	}
	if o.Provider != "" {
		v.Set("provider", o.Provider)
	}
	if o.SendingEnabled != nil {
		v.Set("sendingEnabled", strconv.FormatBool(*o.SendingEnabled))
	}
	if o.ReceivingEnabled != nil {
		v.Set("receivingEnabled", strconv.FormatBool(*o.ReceivingEnabled))
	}
	return v
}

type EmailListOptions struct {
	Page      int
	Limit     int
	Cursor    string
	Status    string
	DomainID  string
	APIKeyID  string
	RequestID string
	Query     url.Values
}

func (o EmailListOptions) values() url.Values {
	v := PageOptions{Page: o.Page, Limit: o.Limit, Cursor: o.Cursor, Query: o.Query}.values()
	if o.Status != "" {
		v.Set("status", o.Status)
	}
	if o.DomainID != "" {
		v.Set("domain_id", o.DomainID)
	}
	if o.APIKeyID != "" {
		v.Set("api_key_id", o.APIKeyID)
	}
	if o.RequestID != "" {
		v.Set("request_id", o.RequestID)
	}
	return v
}

type WebhookListOptions struct {
	Page    int
	Limit   int
	Cursor  string
	Enabled *bool
	Query   url.Values
}

func (o WebhookListOptions) values() url.Values {
	v := PageOptions{Page: o.Page, Limit: o.Limit, Cursor: o.Cursor, Query: o.Query}.values()
	if o.Enabled != nil {
		v.Set("enabled", strconv.FormatBool(*o.Enabled))
	}
	return v
}

type APIKeyListOptions struct {
	Page   int
	Limit  int
	Cursor string
	Query  url.Values
}

func (o APIKeyListOptions) values() url.Values {
	return (PageOptions{Page: o.Page, Limit: o.Limit, Cursor: o.Cursor, Query: o.Query}).values()
}

// EmailsService provides typed email lifecycle methods. The flat methods on
// Client are retained for source compatibility with the first SDK release.
type EmailsService struct{ client *Client }

func (s *EmailsService) Send(ctx context.Context, request SendEmailRequest, idempotencyKey string) (EmailAccepted, ResponseMeta, error) {
	var result EmailAccepted
	meta, err := requestTyped(s.client, ctx, http.MethodPost, "/api/v1/emails", request, idempotencyKey, &result)
	return result, meta, err
}

func (s *EmailsService) SendBatch(ctx context.Context, items []SendEmailRequest, idempotencyKey string) (EmailBatchAccepted, ResponseMeta, error) {
	var result EmailBatchAccepted
	meta, err := requestTyped(s.client, ctx, http.MethodPost, "/api/v1/emails/batch", struct {
		Items []SendEmailRequest `json:"items"`
	}{Items: items}, idempotencyKey, &result)
	return result, meta, err
}

func (s *EmailsService) Schedule(ctx context.Context, id string, request ScheduleEmailRequest, idempotencyKey string) (EmailAccepted, ResponseMeta, error) {
	var result EmailAccepted
	meta, err := requestTyped(s.client, ctx, http.MethodPost, "/api/v1/emails/"+segment(id)+"/schedule", request, idempotencyKey, &result)
	return result, meta, err
}

func (s *EmailsService) Cancel(ctx context.Context, id, idempotencyKey string) (EmailSummary, ResponseMeta, error) {
	var result EmailSummary
	meta, err := requestTyped(s.client, ctx, http.MethodPost, "/api/v1/emails/"+segment(id)+"/cancel", nil, idempotencyKey, &result)
	return result, meta, err
}

func (s *EmailsService) Get(ctx context.Context, id string) (EmailDetail, ResponseMeta, error) {
	var result EmailDetail
	meta, err := requestTyped(s.client, ctx, http.MethodGet, "/api/v1/emails/"+segment(id), nil, "", &result)
	return result, meta, err
}

func (s *EmailsService) List(ctx context.Context, options EmailListOptions) (Page[EmailSummary], ResponseMeta, error) {
	var result Page[EmailSummary]
	meta, err := requestTyped(s.client, ctx, http.MethodGet, withValues("/api/v1/emails", options.values()), nil, "", &result)
	return result, meta, err
}

func (s *EmailsService) Replay(ctx context.Context, id string, request ReplayEmailRequest, idempotencyKey string, options ...ScopedPageOptions) (ReplayPlan, ResponseMeta, error) {
	var result ReplayPlan
	scope, err := scopeOption(options, false)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	path, err := scopedPathChecked("/api/v1/emails/"+segment(id)+"/replay", scope, false)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, request, idempotencyKey, &result)
	return result, meta, err
}

// ReplayRaw preserves a loose JSON escape hatch for additive backend fields;
// Replay is the typed contract surface used by new integrations.
func (s *EmailsService) ReplayRaw(ctx context.Context, id string, request JSON, idempotencyKey string, options ...ScopedPageOptions) (JSON, ResponseMeta, error) {
	var result JSON
	scope, err := scopeOption(options, false)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	path, err := scopedPathChecked("/api/v1/emails/"+segment(id)+"/replay", scope, false)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, request, idempotencyKey, &result)
	return result, meta, err
}

func (s *EmailsService) Trace(ctx context.Context, id string, options ScopedPageOptions) (EmailTrace, ResponseMeta, error) {
	var result EmailTrace
	path, err := scopedPathChecked("/api/v1/emails/"+segment(id)+"/trace", options, false)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

func (s *EmailsService) UploadAttachment(ctx context.Context, request AttachmentUploadRequest, idempotencyKey string) (AttachmentUpload, ResponseMeta, error) {
	var result AttachmentUpload
	meta, err := requestTyped(s.client, ctx, http.MethodPost, "/api/v1/attachments/uploads", request, idempotencyKey, &result)
	return result, meta, err
}

// DomainsService manages sender domains and their DNS verification state.
type DomainsService struct{ client *Client }

func (s *DomainsService) List(ctx context.Context, options DomainListOptions) (Page[Domain], ResponseMeta, error) {
	var result Page[Domain]
	meta, err := requestTyped(s.client, ctx, http.MethodGet, withValues("/api/v1/domains", options.values()), nil, "", &result)
	return result, meta, err
}

func (s *DomainsService) Create(ctx context.Context, request CreateDomainRequest, idempotencyKeys ...string) (Domain, ResponseMeta, error) {
	var result Domain
	idempotencyKey, err := optionalIdempotencyKey(idempotencyKeys)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithPolicy(s.client, ctx, http.MethodPost, "/api/v1/domains", request, idempotencyKey, &result, true)
	return result, meta, err
}

func (s *DomainsService) Get(ctx context.Context, id string) (Domain, ResponseMeta, error) {
	var result Domain
	meta, err := requestTyped(s.client, ctx, http.MethodGet, "/api/v1/domains/"+segment(id), nil, "", &result)
	return result, meta, err
}

func (s *DomainsService) Update(ctx context.Context, id string, request UpdateDomainRequest, idempotencyKeys ...string) (Domain, ResponseMeta, error) {
	var result Domain
	idempotencyKey, err := optionalIdempotencyKey(idempotencyKeys)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithPolicy(s.client, ctx, http.MethodPatch, "/api/v1/domains/"+segment(id), request, idempotencyKey, &result, true)
	return result, meta, err
}

func (s *DomainsService) Delete(ctx context.Context, id string, idempotencyKeys ...string) (ResponseMeta, error) {
	idempotencyKey, err := optionalIdempotencyKey(idempotencyKeys)
	if err != nil {
		return ResponseMeta{}, err
	}
	return requestTypedWithPolicy(s.client, ctx, http.MethodDelete, "/api/v1/domains/"+segment(id), nil, idempotencyKey, (*JSON)(nil), true)
}

func (s *DomainsService) Verify(ctx context.Context, id string, request VerifyDomainRequest, idempotencyKeys ...string) (DomainVerification, ResponseMeta, error) {
	var result DomainVerification
	idempotencyKey, err := optionalIdempotencyKey(idempotencyKeys)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithPolicy(s.client, ctx, http.MethodPost, "/api/v1/domains/"+segment(id)+"/verify", request, idempotencyKey, &result, true)
	return result, meta, err
}

func (s *DomainsService) DNSRecords(ctx context.Context, id string) (Page[DNSRecord], ResponseMeta, error) {
	var result struct {
		Items []DNSRecord `json:"items"`
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, "/api/v1/domains/"+segment(id)+"/dns-records", nil, "", &result)
	return Page[DNSRecord]{Items: result.Items}, meta, err
}

// ProvisionDNS requests provider-managed DNS provisioning. The backend
// returns HTTP 202 and the exact records that were accepted for verification.
func (s *DomainsService) ProvisionDNS(ctx context.Context, id string, request ProvisionDNSRequest, idempotencyKey string) (DNSProvisionAccepted, ResponseMeta, error) {
	var result DNSProvisionAccepted
	path := "/api/v1/domains/" + segment(id) + "/dns-records/provision"
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, request, idempotencyKey, &result)
	return result, meta, err
}

// DomainHealthListOptions is the scoped filter for domain-health reports.
// Domain health is an organization/project aggregate, so tenant_id is
// optional even though project_id and environment remain mandatory.
type DomainHealthListOptions struct {
	Page        int
	Limit       int
	Cursor      string
	TenantID    string
	ProjectID   string
	Environment string
	Query       url.Values
}

func (o DomainHealthListOptions) scope() ScopedPageOptions {
	return ScopedPageOptions{Page: o.Page, Limit: o.Limit, Cursor: o.Cursor, TenantID: o.TenantID, ProjectID: o.ProjectID, Environment: o.Environment, Query: o.Query}
}

// DomainHealthService reads reports and triggers an asynchronous verification
// for a domain. The service is separate from DomainsService because the
// backend's report lifecycle is a native full-product capability.
type DomainHealthService struct{ client *Client }

func (s *DomainHealthService) List(ctx context.Context, options DomainHealthListOptions) (DomainHealthPage, ResponseMeta, error) {
	var result DomainHealthPage
	path, err := scopedPathChecked("/api/v1/domain-health", options.scope(), false)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

func (s *DomainHealthService) Get(ctx context.Context, domainID string, options ScopedPageOptions) (DomainHealthReport, ResponseMeta, error) {
	var result DomainHealthReport
	path, err := scopedPathChecked("/api/v1/domain-health/"+segment(domainID), options, false)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

func (s *DomainHealthService) Verify(ctx context.Context, request VerifyDomainHealthRequest, idempotencyKey string, options ...ScopedPageOptions) (DomainHealthReport, ResponseMeta, error) {
	var result DomainHealthReport
	scope, err := scopeOption(options, false)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	if request.DomainID == "" {
		return result, ResponseMeta{}, fmt.Errorf("%w: domain_id is required", ErrInvalidScope)
	}
	path, err := scopedPathChecked("/api/v1/domain-health", scope, false)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, request, idempotencyKey, &result)
	return result, meta, err
}

// APIKeysService manages scoped API-key lifecycle. The create response's
// token is returned only once by the backend and is never cached by the SDK.
type APIKeysService struct{ client *Client }

func (s *APIKeysService) List(ctx context.Context, options APIKeyListOptions) (Page[APIKey], ResponseMeta, error) {
	var result Page[APIKey]
	meta, err := requestTyped(s.client, ctx, http.MethodGet, withValues("/api/v1/api-keys", options.values()), nil, "", &result)
	return result, meta, err
}

func (s *APIKeysService) Create(ctx context.Context, request CreateAPIKeyRequest, idempotencyKeys ...string) (CreateAPIKeyResponse, ResponseMeta, error) {
	var result CreateAPIKeyResponse
	idempotencyKey, err := optionalIdempotencyKey(idempotencyKeys)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithPolicy(s.client, ctx, http.MethodPost, "/api/v1/api-keys", request, idempotencyKey, &result, true)
	return result, meta, err
}

func (s *APIKeysService) Update(ctx context.Context, id string, request UpdateAPIKeyRequest, idempotencyKeys ...string) (APIKey, ResponseMeta, error) {
	var result APIKey
	idempotencyKey, err := optionalIdempotencyKey(idempotencyKeys)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithPolicy(s.client, ctx, http.MethodPatch, "/api/v1/api-keys/"+segment(id), request, idempotencyKey, &result, true)
	return result, meta, err
}

func (s *APIKeysService) Revoke(ctx context.Context, id string, idempotencyKeys ...string) (ResponseMeta, error) {
	idempotencyKey, err := optionalIdempotencyKey(idempotencyKeys)
	if err != nil {
		return ResponseMeta{}, err
	}
	return requestTypedWithPolicy(s.client, ctx, http.MethodDelete, "/api/v1/api-keys/"+segment(id), nil, idempotencyKey, (*JSON)(nil), true)
}

// WebhooksService manages endpoint subscriptions and replayable deliveries.
type WebhooksService struct{ client *Client }

func (s *WebhooksService) List(ctx context.Context, options WebhookListOptions) (Page[Webhook], ResponseMeta, error) {
	var result Page[Webhook]
	meta, err := requestTyped(s.client, ctx, http.MethodGet, withValues("/api/v1/webhooks", options.values()), nil, "", &result)
	return result, meta, err
}

func (s *WebhooksService) Create(ctx context.Context, request CreateWebhookRequest, idempotencyKeys ...string) (WebhookCreated, ResponseMeta, error) {
	var result WebhookCreated
	idempotencyKey, err := optionalIdempotencyKey(idempotencyKeys)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithPolicy(s.client, ctx, http.MethodPost, "/api/v1/webhooks", request, idempotencyKey, &result, true)
	return result, meta, err
}

func (s *WebhooksService) Get(ctx context.Context, id string) (Webhook, ResponseMeta, error) {
	var result Webhook
	meta, err := requestTyped(s.client, ctx, http.MethodGet, "/api/v1/webhooks/"+segment(id), nil, "", &result)
	return result, meta, err
}

func (s *WebhooksService) Update(ctx context.Context, id string, request UpdateWebhookRequest, idempotencyKeys ...string) (Webhook, ResponseMeta, error) {
	var result Webhook
	idempotencyKey, err := optionalIdempotencyKey(idempotencyKeys)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithPolicy(s.client, ctx, http.MethodPatch, "/api/v1/webhooks/"+segment(id), request, idempotencyKey, &result, true)
	return result, meta, err
}

func (s *WebhooksService) Delete(ctx context.Context, id string, idempotencyKeys ...string) (ResponseMeta, error) {
	idempotencyKey, err := optionalIdempotencyKey(idempotencyKeys)
	if err != nil {
		return ResponseMeta{}, err
	}
	return requestTypedWithPolicy(s.client, ctx, http.MethodDelete, "/api/v1/webhooks/"+segment(id), nil, idempotencyKey, (*JSON)(nil), true)
}

func (s *WebhooksService) RotateSecret(ctx context.Context, id string, request JSON, idempotencyKeys ...string) (WebhookSecretRotation, ResponseMeta, error) {
	var result WebhookSecretRotation
	idempotencyKey, err := optionalIdempotencyKey(idempotencyKeys)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithPolicy(s.client, ctx, http.MethodPost, "/api/v1/webhooks/"+segment(id)+"/rotate-secret", request, idempotencyKey, &result, true)
	return result, meta, err
}

type DeliveryListOptions struct {
	Page          int
	Limit         int
	Cursor        string
	EndpointID    string
	EmailID       string
	Status        string
	EventType     string
	CreatedAfter  string
	CreatedBefore string
	Query         url.Values
}

func (o DeliveryListOptions) values() url.Values {
	v := PageOptions{Page: o.Page, Limit: o.Limit, Cursor: o.Cursor, Query: o.Query}.values()
	for key, value := range map[string]string{"endpoint_id": o.EndpointID, "email_id": o.EmailID, "status": o.Status, "event_type": o.EventType, "created_after": o.CreatedAfter, "created_before": o.CreatedBefore} {
		if value != "" {
			v.Set(key, value)
		}
	}
	return v
}

func (s *WebhooksService) Deliveries(ctx context.Context, options DeliveryListOptions) (Page[WebhookDelivery], ResponseMeta, error) {
	var result Page[WebhookDelivery]
	meta, err := requestTyped(s.client, ctx, http.MethodGet, withValues("/api/v1/webhook-deliveries", options.values()), nil, "", &result)
	return result, meta, err
}

func (s *WebhooksService) Delivery(ctx context.Context, id string) (WebhookDelivery, ResponseMeta, error) {
	var result WebhookDelivery
	meta, err := requestTyped(s.client, ctx, http.MethodGet, "/api/v1/webhook-deliveries/"+segment(id), nil, "", &result)
	return result, meta, err
}

func (s *WebhooksService) ReplayDelivery(ctx context.Context, id string, idempotencyKeys ...string) (WebhookDelivery, ResponseMeta, error) {
	var result WebhookDelivery
	idempotencyKey, err := optionalIdempotencyKey(idempotencyKeys)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithPolicy(s.client, ctx, http.MethodPost, "/api/v1/webhook-deliveries/"+segment(id)+"/replay", nil, idempotencyKey, &result, true)
	return result, meta, err
}
