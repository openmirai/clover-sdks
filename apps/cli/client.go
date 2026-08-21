package main

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
	NextCursor         string
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
	MaxRequestBodyBytes        int
	MaxResponseBodyBytes       int
	Sleep                      func(context.Context, time.Duration) error
}

const DefaultMaxRequestBodyBytes = 4 << 20
const DefaultMaxResponseBodyBytes = 4 << 20

var idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$`)

func NewClient(baseURL, apiKey string) *Client {
	client, err := NewClientWithError(baseURL, apiKey)
	if err != nil {
		panic(err.Error())
	}
	return client
}

// NewClientWithError validates operator configuration without panicking. The
// CLI uses this path so missing environment variables produce an actionable
// error instead of a stack trace; NewClient remains for source compatibility.
func NewClientWithError(baseURL, apiKey string) (*Client, error) {
	baseURL = strings.TrimSpace(baseURL)
	apiKey = strings.TrimSpace(apiKey)
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, errors.New("CLOVER_BASE_URL must be an absolute http(s) URL without userinfo, query, or fragment")
	}
	if apiKey == "" {
		return nil, errors.New("CLOVER_API_KEY is required")
	}
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), APIKey: apiKey, UserAgent: "clover-cli/0.1.0", HTTPClient: http.DefaultClient, MaxRetries: 2, RetryBaseDelay: 100 * time.Millisecond, MaxRequestBodyBytes: DefaultMaxRequestBodyBytes, MaxResponseBodyBytes: DefaultMaxResponseBodyBytes}, nil
}
func (c *Client) Send(ctx context.Context, payload map[string]any, key string) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, "POST", "/api/v1/emails", payload, key)
}
func (c *Client) SendBatch(ctx context.Context, items []map[string]any, key string) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, "POST", "/api/v1/emails/batch", map[string]any{"items": items}, key)
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

// Scope is the ownership selection used by full-product endpoints. Scoped
// queries require project_id and environment (and tenant_id for tenant-owned
// routes); request-body scopes additionally carry organization_id.
type Scope struct {
	OrganizationID string
	ProjectID      string
	Environment    string
	TenantID       string
}

func (s Scope) validate(requireTenant bool) error {
	if strings.TrimSpace(s.ProjectID) == "" {
		return errors.New("project-id is required")
	}
	if strings.TrimSpace(s.Environment) == "" {
		return errors.New("environment is required")
	}
	switch strings.TrimSpace(s.Environment) {
	case "development", "preview", "staging", "production":
	default:
		return errors.New("environment must be development, preview, staging, or production")
	}
	if requireTenant && strings.TrimSpace(s.TenantID) == "" {
		return errors.New("tenant-id is required")
	}
	return nil
}

func (s Scope) values(query url.Values, requireTenant bool) (url.Values, error) {
	if err := s.validate(requireTenant); err != nil {
		return nil, err
	}
	if query == nil {
		query = url.Values{}
	} else {
		query = cloneValues(query)
	}
	query.Set("project_id", strings.TrimSpace(s.ProjectID))
	query.Set("environment", strings.TrimSpace(s.Environment))
	if strings.TrimSpace(s.TenantID) != "" {
		query.Set("tenant_id", strings.TrimSpace(s.TenantID))
	}
	return query, nil
}

func cloneValues(input url.Values) url.Values {
	output := make(url.Values, len(input))
	for key, values := range input {
		output[key] = append([]string(nil), values...)
	}
	return output
}

func (c *Client) scopedRequest(ctx context.Context, method, path string, payload any, key string, scope Scope, requireTenant bool) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequestWithKeyRequirement(ctx, method, path, payload, key, scope, requireTenant, true)
}

func (c *Client) scopedOptionalRequest(ctx context.Context, method, path string, payload any, scope Scope, requireTenant bool) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequestWithKeyRequirement(ctx, method, path, payload, "", scope, requireTenant, false)
}

func (c *Client) scopedRequestWithKeyRequirement(ctx context.Context, method, path string, payload any, key string, scope Scope, requireTenant, requireKey bool) (json.RawMessage, ResponseMeta, error) {
	basePath := path
	existing := url.Values{}
	if index := strings.IndexByte(path, '?'); index >= 0 {
		basePath = path[:index]
		var err error
		existing, err = url.ParseQuery(path[index+1:])
		if err != nil {
			return nil, ResponseMeta{}, fmt.Errorf("invalid request query: %w", err)
		}
	}
	query, err := scope.values(existing, requireTenant)
	if err != nil {
		return nil, ResponseMeta{}, err
	}
	if encoded := query.Encode(); encoded != "" {
		basePath += "?" + encoded
	}
	return c.requestWithHeadersRequirement(ctx, method, basePath, payload, key, http.Header{"X-Environment": []string{strings.TrimSpace(scope.Environment)}}, requireKey)
}

func (c *Client) requestOptional(ctx context.Context, method, path string, payload any) (json.RawMessage, ResponseMeta, error) {
	return c.requestWithHeadersRequirement(ctx, method, path, payload, "", nil, false)
}

func (c *Client) requestOptionalWithKey(ctx context.Context, method, path string, payload any, key string) (json.RawMessage, ResponseMeta, error) {
	return c.requestWithHeadersRequirement(ctx, method, path, payload, key, nil, false)
}

// ListDomains lists organization-owned sending domains.
func (c *Client) ListDomains(ctx context.Context, query url.Values) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, http.MethodGet, withQuery("/api/v1/domains", query), nil, "")
}

func (c *Client) GetDomain(ctx context.Context, id string) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, http.MethodGet, "/api/v1/domains/"+url.PathEscape(id), nil, "")
}

func (c *Client) CreateDomain(ctx context.Context, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.requestOptionalWithKey(ctx, http.MethodPost, "/api/v1/domains", payload, key)
}

func (c *Client) ConfigureDomain(ctx context.Context, id string, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.requestOptionalWithKey(ctx, http.MethodPatch, "/api/v1/domains/"+url.PathEscape(id), payload, key)
}

func (c *Client) VerifyDomain(ctx context.Context, id string, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.requestOptionalWithKey(ctx, http.MethodPost, "/api/v1/domains/"+url.PathEscape(id)+"/verify", payload, key)
}

func (c *Client) DeleteDomain(ctx context.Context, id, key string) (json.RawMessage, ResponseMeta, error) {
	return c.requestOptionalWithKey(ctx, http.MethodDelete, "/api/v1/domains/"+url.PathEscape(id), nil, key)
}

func (c *Client) ProvisionDomainDNS(ctx context.Context, id string, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, http.MethodPost, "/api/v1/domains/"+url.PathEscape(id)+"/dns-records/provision", payload, key)
}

func (c *Client) ListDomainDNSRecords(ctx context.Context, id string) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, http.MethodGet, "/api/v1/domains/"+url.PathEscape(id)+"/dns-records", nil, "")
}

func (c *Client) ListTemplates(ctx context.Context, query url.Values, scope Scope) (json.RawMessage, ResponseMeta, error) {
	query, err := scope.values(query, true)
	if err != nil {
		return nil, ResponseMeta{}, err
	}
	return c.requestWithHeaders(ctx, http.MethodGet, withQuery("/api/v1/templates", query), nil, "", http.Header{"X-Environment": []string{strings.TrimSpace(scope.Environment)}})
}

func (c *Client) CreateTemplate(ctx context.Context, payload json.RawMessage, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodPost, "/api/v1/templates", payload, key, scope, true)
}

func (c *Client) CreateTemplateScoped(ctx context.Context, payload json.RawMessage, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.CreateTemplate(ctx, payload, key, scope)
}

func (c *Client) GetTemplate(ctx context.Context, id string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodGet, "/api/v1/templates/"+url.PathEscape(id), nil, "", scope, true)
}

func (c *Client) TransitionTemplate(ctx context.Context, id string, payload json.RawMessage, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodPatch, "/api/v1/templates/"+url.PathEscape(id), payload, key, scope, true)
}

func (c *Client) ListTemplateVersions(ctx context.Context, templateID string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodGet, "/api/v1/templates/"+url.PathEscape(templateID)+"/versions", nil, "", scope, true)
}

func (c *Client) CreateTemplateVersion(ctx context.Context, templateID string, payload json.RawMessage, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodPost, "/api/v1/templates/"+url.PathEscape(templateID)+"/versions", payload, key, scope, true)
}

func (c *Client) CreateTemplateVersionScoped(ctx context.Context, templateID string, payload json.RawMessage, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.CreateTemplateVersion(ctx, templateID, payload, key, scope)
}

func (c *Client) PublishTemplateVersion(ctx context.Context, templateID, versionID, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodPost, "/api/v1/templates/"+url.PathEscape(templateID)+"/versions/"+url.PathEscape(versionID)+"/publish", map[string]any{}, key, scope, true)
}

func (c *Client) GetTemplateVersion(ctx context.Context, templateID, versionRef string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodGet, "/api/v1/templates/"+url.PathEscape(templateID)+"/versions/"+url.PathEscape(versionRef), nil, "", scope, true)
}

func (c *Client) CompareTemplateVersions(ctx context.Context, templateID, from, to string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	query, err := scope.values(url.Values{"from": []string{from}, "to": []string{to}}, true)
	if err != nil {
		return nil, ResponseMeta{}, err
	}
	return c.requestWithHeaders(ctx, http.MethodGet, withQuery("/api/v1/templates/"+url.PathEscape(templateID)+"/versions/compare", query), nil, "", http.Header{"X-Environment": []string{strings.TrimSpace(scope.Environment)}})
}

func (c *Client) RollbackTemplateVersion(ctx context.Context, templateID, versionID, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodPost, "/api/v1/templates/"+url.PathEscape(templateID)+"/versions/"+url.PathEscape(versionID)+"/rollback", map[string]any{}, key, scope, true)
}

func (c *Client) ListContacts(ctx context.Context, query url.Values, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodGet, withQuery("/api/v1/contacts", query), nil, "", scope, true)
}

func (c *Client) CreateContact(ctx context.Context, payload json.RawMessage, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodPost, "/api/v1/contacts", payload, key, scope, true)
}

func (c *Client) GetContact(ctx context.Context, id string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodGet, "/api/v1/contacts/"+url.PathEscape(id), nil, "", scope, true)
}

func (c *Client) TransitionContact(ctx context.Context, id string, payload json.RawMessage, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodPatch, "/api/v1/contacts/"+url.PathEscape(id), payload, key, scope, true)
}

func (c *Client) ListSegments(ctx context.Context, query url.Values, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodGet, withQuery("/api/v1/segments", query), nil, "", scope, true)
}

func (c *Client) CreateSegment(ctx context.Context, payload json.RawMessage, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodPost, "/api/v1/segments", payload, key, scope, true)
}

func (c *Client) GetSegment(ctx context.Context, id string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodGet, "/api/v1/segments/"+url.PathEscape(id), nil, "", scope, true)
}

func (c *Client) ArchiveSegment(ctx context.Context, id, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodPatch, "/api/v1/segments/"+url.PathEscape(id), map[string]any{}, key, scope, true)
}

func (c *Client) EvaluateSegment(ctx context.Context, id, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	path := "/api/v1/segments/" + url.PathEscape(id) + "/evaluate"
	return c.scopedRequestWithKeyRequirement(ctx, http.MethodPost, path, map[string]any{}, key, scope, true, false)
}

func (c *Client) ListBroadcasts(ctx context.Context, query url.Values, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodGet, withQuery("/api/v1/broadcasts", query), nil, "", scope, true)
}

func (c *Client) CreateBroadcast(ctx context.Context, payload json.RawMessage, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodPost, "/api/v1/broadcasts", payload, key, scope, true)
}

func (c *Client) GetBroadcast(ctx context.Context, id string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodGet, "/api/v1/broadcasts/"+url.PathEscape(id), nil, "", scope, true)
}

func (c *Client) ScheduleBroadcast(ctx context.Context, id string, payload json.RawMessage, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodPost, "/api/v1/broadcasts/"+url.PathEscape(id)+"/schedule", payload, key, scope, true)
}

func (c *Client) CancelBroadcast(ctx context.Context, id, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodPost, "/api/v1/broadcasts/"+url.PathEscape(id)+"/cancel", map[string]any{}, key, scope, true)
}

func (c *Client) ListAutomations(ctx context.Context, query url.Values, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodGet, withQuery("/api/v1/automations", query), nil, "", scope, true)
}

func (c *Client) CreateAutomation(ctx context.Context, payload json.RawMessage, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodPost, "/api/v1/automations", payload, key, scope, true)
}

func (c *Client) UpdateAutomation(ctx context.Context, id string, payload json.RawMessage, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodPatch, "/api/v1/automations/"+url.PathEscape(id), payload, key, scope, true)
}

func (c *Client) GetAutomation(ctx context.Context, id string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodGet, "/api/v1/automations/"+url.PathEscape(id), nil, "", scope, true)
}

func (c *Client) TransitionAutomation(ctx context.Context, id, action, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodPost, "/api/v1/automations/"+url.PathEscape(id)+"/"+url.PathEscape(action), map[string]any{}, key, scope, true)
}

func (c *Client) StartAutomationRun(ctx context.Context, id string, payload json.RawMessage, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodPost, "/api/v1/automations/"+url.PathEscape(id)+"/runs", payload, key, scope, true)
}

func (c *Client) GetAutomationRun(ctx context.Context, automationID, runID string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodGet, "/api/v1/automations/"+url.PathEscape(automationID)+"/runs/"+url.PathEscape(runID), nil, "", scope, true)
}

func (c *Client) IngestAutomationEvent(ctx context.Context, payload json.RawMessage, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodPost, "/api/v1/automation-events", payload, key, scope, true)
}

func (c *Client) ListDomainHealth(ctx context.Context, query url.Values, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodGet, withQuery("/api/v1/domain-health", query), nil, "", scope, false)
}

func (c *Client) VerifyDomainHealth(ctx context.Context, payload json.RawMessage, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodPost, "/api/v1/domain-health", payload, key, scope, false)
}

func (c *Client) GetDomainHealth(ctx context.Context, id string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodGet, "/api/v1/domain-health/"+url.PathEscape(id), nil, "", scope, false)
}

func (c *Client) ListAuditEvents(ctx context.Context, query url.Values, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodGet, withQuery("/api/v1/audit-events", query), nil, "", scope, false)
}

func (c *Client) AppendAuditEvent(ctx context.Context, payload json.RawMessage, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodPost, "/api/v1/audit-events", payload, key, scope, false)
}

func (c *Client) GetAuditEvent(ctx context.Context, id string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodGet, "/api/v1/audit-events/"+url.PathEscape(id), nil, "", scope, false)
}

func (c *Client) ListAuditHolds(ctx context.Context, query url.Values, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodGet, withQuery("/api/v1/audit-events/holds", query), nil, "", scope, false)
}

func (c *Client) CreateAuditHold(ctx context.Context, payload json.RawMessage, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodPost, "/api/v1/audit-events/holds", payload, key, scope, false)
}

func (c *Client) ReleaseAuditHold(ctx context.Context, id, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodDelete, "/api/v1/audit-events/holds/"+url.PathEscape(id), map[string]any{}, key, scope, false)
}

func (c *Client) ListDeliveryPolicies(ctx context.Context, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodGet, "/api/v1/delivery-policies", nil, "", scope, false)
}

func (c *Client) CreateDeliveryPolicy(ctx context.Context, payload json.RawMessage, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodPost, "/api/v1/delivery-policies", payload, key, scope, false)
}

func (c *Client) UpdateDeliveryPolicy(ctx context.Context, id string, payload json.RawMessage, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodPatch, "/api/v1/delivery-policies/"+url.PathEscape(id), payload, key, scope, false)
}

func (c *Client) ListDeliveryRoutes(ctx context.Context, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodGet, "/api/v1/delivery-routes", nil, "", scope, false)
}

func (c *Client) CreateDeliveryRoute(ctx context.Context, payload json.RawMessage, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodPost, "/api/v1/delivery-routes", payload, key, scope, false)
}

func (c *Client) ListSMTPCredentials(ctx context.Context, query url.Values, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodGet, withQuery("/api/v1/smtp-credentials", query), nil, "", scope, false)
}

func (c *Client) CreateSMTPCredential(ctx context.Context, payload json.RawMessage, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodPost, "/api/v1/smtp-credentials", payload, key, scope, false)
}

func (c *Client) GetSMTPCredential(ctx context.Context, id string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodGet, "/api/v1/smtp-credentials/"+url.PathEscape(id), nil, "", scope, false)
}

func (c *Client) RevokeSMTPCredential(ctx context.Context, id, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodDelete, "/api/v1/smtp-credentials/"+url.PathEscape(id), nil, key, scope, false)
}

// Provider-neutral routing control-plane endpoints. These routes are
// organization-owned and intentionally do not carry the full-product scope
// query because the authenticated principal selects the organization.
func (c *Client) GetRoutingPolicy(ctx context.Context) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, http.MethodGet, "/api/v1/routing/policy", nil, "")
}

func (c *Client) PutRoutingPolicy(ctx context.Context, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, http.MethodPut, "/api/v1/routing/policy", payload, key)
}

func (c *Client) ListRoutingCapabilities(ctx context.Context) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, http.MethodGet, "/api/v1/routing/capabilities", nil, "")
}

func (c *Client) ListRoutingPools(ctx context.Context) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, http.MethodGet, "/api/v1/routing/pools", nil, "")
}

func (c *Client) CreateRoutingPool(ctx context.Context, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, http.MethodPost, "/api/v1/routing/pools", payload, key)
}

func (c *Client) GetRoutingPool(ctx context.Context, id string) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, http.MethodGet, "/api/v1/routing/pools/"+url.PathEscape(id), nil, "")
}

func (c *Client) ApplyRoutingPoolCommand(ctx context.Context, id string, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, http.MethodPost, "/api/v1/routing/pools/"+url.PathEscape(id)+"/command", payload, key)
}

func (c *Client) ApplyRoutingIPCommand(ctx context.Context, poolID, ipID string, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, http.MethodPost, "/api/v1/routing/pools/"+url.PathEscape(poolID)+"/ips/"+url.PathEscape(ipID)+"/command", payload, key)
}

func (c *Client) ListRoutingAudit(ctx context.Context, query url.Values) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, http.MethodGet, withQuery("/api/v1/routing/audit", query), nil, "")
}

func (c *Client) GetEmailTrace(ctx context.Context, id string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodGet, "/api/v1/emails/"+url.PathEscape(id)+"/trace", nil, "", scope, false)
}

func (c *Client) ReplayEmail(ctx context.Context, id string, payload json.RawMessage, key string, scope Scope) (json.RawMessage, ResponseMeta, error) {
	return c.scopedRequest(ctx, http.MethodPost, "/api/v1/emails/"+url.PathEscape(id)+"/replay", payload, key, scope, false)
}

func (c *Client) ListAPIKeys(ctx context.Context) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, http.MethodGet, "/api/v1/api-keys", nil, "")
}

func (c *Client) CreateAPIKey(ctx context.Context, payload json.RawMessage) (json.RawMessage, ResponseMeta, error) {
	return c.requestOptional(ctx, http.MethodPost, "/api/v1/api-keys", payload)
}

func (c *Client) UpdateAPIKey(ctx context.Context, id string, payload json.RawMessage) (json.RawMessage, ResponseMeta, error) {
	return c.requestOptional(ctx, http.MethodPatch, "/api/v1/api-keys/"+url.PathEscape(id), payload)
}

func (c *Client) RevokeAPIKey(ctx context.Context, id string) (json.RawMessage, ResponseMeta, error) {
	return c.requestOptional(ctx, http.MethodDelete, "/api/v1/api-keys/"+url.PathEscape(id), nil)
}

func (c *Client) ListReceivedEmails(ctx context.Context, query url.Values) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, http.MethodGet, withQuery("/api/v1/received-emails", query), nil, "")
}

func (c *Client) GetReceivedEmail(ctx context.Context, id string) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, http.MethodGet, "/api/v1/received-emails/"+url.PathEscape(id), nil, "")
}

func (c *Client) GetReceivedEmailAttachment(ctx context.Context, emailID, attachmentID string) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, http.MethodGet, "/api/v1/received-emails/"+url.PathEscape(emailID)+"/attachments/"+url.PathEscape(attachmentID), nil, "")
}

func (c *Client) AcceptInbound(ctx context.Context, provider string, payload json.RawMessage) (json.RawMessage, ResponseMeta, error) {
	return c.AcceptInboundWithHeaders(ctx, provider, payload, nil)
}

func (c *Client) AcceptProviderEvent(ctx context.Context, provider string, payload json.RawMessage) (json.RawMessage, ResponseMeta, error) {
	return c.AcceptProviderEventWithHeaders(ctx, provider, payload, nil)
}

// AcceptInboundWithHeaders sends a provider callback with the optional
// provider verification headers. The public endpoint validates these headers
// when its provider verifier is configured; they are kept separate from the
// bearer API key used by the CLI client.
func (c *Client) AcceptInboundWithHeaders(ctx context.Context, provider string, payload json.RawMessage, headers http.Header) (json.RawMessage, ResponseMeta, error) {
	return c.requestWithHeadersRequirement(ctx, http.MethodPost, "/api/v1/inbound/"+url.PathEscape(provider), payload, "", headers, false)
}

// AcceptProviderEventWithHeaders is the provider-event counterpart to
// AcceptInboundWithHeaders.
func (c *Client) AcceptProviderEventWithHeaders(ctx context.Context, provider string, payload json.RawMessage, headers http.Header) (json.RawMessage, ResponseMeta, error) {
	return c.requestWithHeadersRequirement(ctx, http.MethodPost, "/api/v1/provider-events/"+url.PathEscape(provider), payload, "", headers, false)
}

func (c *Client) ListSuppressions(ctx context.Context, query url.Values) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, http.MethodGet, withQuery("/api/v1/suppressions", query), nil, "")
}

func (c *Client) CreateSuppression(ctx context.Context, payload json.RawMessage) (json.RawMessage, ResponseMeta, error) {
	return c.requestOptional(ctx, http.MethodPost, "/api/v1/suppressions", payload)
}

func (c *Client) DeleteSuppression(ctx context.Context, id string) (json.RawMessage, ResponseMeta, error) {
	return c.requestOptional(ctx, http.MethodDelete, "/api/v1/suppressions/"+url.PathEscape(id), nil)
}

func (c *Client) GetPreferenceCenter(ctx context.Context, token string) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, http.MethodGet, "/api/v1/preferences/"+url.PathEscape(token), nil, "")
}

func (c *Client) UpdatePreferenceCenter(ctx context.Context, token string, payload json.RawMessage) (json.RawMessage, ResponseMeta, error) {
	return c.requestOptional(ctx, http.MethodPost, "/api/v1/preferences/"+url.PathEscape(token), payload)
}

func (c *Client) ListWebhooks(ctx context.Context, query url.Values) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, http.MethodGet, withQuery("/api/v1/webhooks", query), nil, "")
}

func (c *Client) CreateWebhook(ctx context.Context, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.requestOptionalWithKey(ctx, http.MethodPost, "/api/v1/webhooks", payload, key)
}

func (c *Client) GetWebhook(ctx context.Context, id string) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, http.MethodGet, "/api/v1/webhooks/"+url.PathEscape(id), nil, "")
}

func (c *Client) UpdateWebhook(ctx context.Context, id string, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.requestOptionalWithKey(ctx, http.MethodPatch, "/api/v1/webhooks/"+url.PathEscape(id), payload, key)
}

func (c *Client) DeleteWebhook(ctx context.Context, id, key string) (json.RawMessage, ResponseMeta, error) {
	return c.requestOptionalWithKey(ctx, http.MethodDelete, "/api/v1/webhooks/"+url.PathEscape(id), nil, key)
}

func (c *Client) RotateWebhookSecret(ctx context.Context, id string, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.requestOptionalWithKey(ctx, http.MethodPost, "/api/v1/webhooks/"+url.PathEscape(id)+"/rotate-secret", payload, key)
}

func (c *Client) ListWebhookDeliveries(ctx context.Context, query url.Values) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, http.MethodGet, withQuery("/api/v1/webhook-deliveries", query), nil, "")
}

func (c *Client) GetWebhookDelivery(ctx context.Context, id string) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, http.MethodGet, "/api/v1/webhook-deliveries/"+url.PathEscape(id), nil, "")
}

func (c *Client) ReplayWebhookDelivery(ctx context.Context, id, key string) (json.RawMessage, ResponseMeta, error) {
	return c.requestOptionalWithKey(ctx, http.MethodPost, "/api/v1/webhook-deliveries/"+url.PathEscape(id)+"/replay", map[string]any{}, key)
}

func (c *Client) ListLogs(ctx context.Context, query url.Values) (json.RawMessage, ResponseMeta, error) {
	return c.request(ctx, http.MethodGet, withQuery("/api/v1/logs", query), nil, "")
}

// FollowLogs polls the documented request-log endpoint. It emits each unseen
// item and advances X-Next-Cursor when the API provides one. The function
// returns when ctx is cancelled, a request fails, or onEvent returns an error.
// It intentionally does not assume an SSE endpoint exists.
func (c *Client) FollowLogs(ctx context.Context, query url.Values, interval time.Duration, onEvent func(json.RawMessage) error) error {
	if onEvent == nil {
		return errors.New("log follow callback is required")
	}
	if interval < 100*time.Millisecond {
		return errors.New("log follow interval must be at least 100ms")
	}
	query = cloneValues(query)
	seen := make(map[string]struct{})
	seenOrder := make([]string, 0, 1024)
	for {
		page, meta, err := c.ListLogs(ctx, query)
		if err != nil {
			return err
		}
		items, err := pageItems(page)
		if err != nil {
			return err
		}
		for _, item := range items {
			identity := itemIdentity(item)
			if identity != "" {
				if _, exists := seen[identity]; exists {
					continue
				}
				seen[identity] = struct{}{}
				seenOrder = append(seenOrder, identity)
				if len(seenOrder) > 10000 {
					delete(seen, seenOrder[0])
					seenOrder = seenOrder[1:]
				}
			}
			if err := onEvent(item); err != nil {
				return err
			}
		}
		if meta.NextCursor != "" {
			query.Set("cursor", meta.NextCursor)
		}
		timer := time.NewTimer(interval)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		}
	}
}

func pageItems(page json.RawMessage) ([]json.RawMessage, error) {
	var value struct {
		Items []json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(page, &value); err != nil {
		return nil, fmt.Errorf("Clover logs response is not valid JSON: %w", err)
	}
	if value.Items == nil {
		return []json.RawMessage{}, nil
	}
	return value.Items, nil
}

func itemIdentity(item json.RawMessage) string {
	var value struct {
		ID        string `json:"id"`
		RequestID string `json:"request_id"`
	}
	if json.Unmarshal(item, &value) != nil {
		return ""
	}
	if value.ID != "" {
		return "id:" + value.ID
	}
	if value.RequestID != "" {
		return "request_id:" + value.RequestID
	}
	return ""
}

func withQuery(path string, query url.Values) string {
	if encoded := query.Encode(); encoded != "" {
		return path + "?" + encoded
	}
	return path
}

// StreamEvents is retained for source compatibility with older callers, but
// Clover does not expose an SSE endpoint. Use the documented request-log
// polling contract instead; a non-empty legacy path is rejected so callers do
// not accidentally rely on a nonexistent stream route.
func (c *Client) StreamEvents(ctx context.Context, path string, onEvent func(json.RawMessage) error) error {
	if strings.TrimSpace(path) != "" && strings.TrimSpace(path) != "/api/v1/logs" {
		return errors.New("Clover SSE streaming is unsupported; use FollowLogs over /api/v1/logs")
	}
	return c.FollowLogs(ctx, nil, 2*time.Second, onEvent)
}

func (c *Client) request(ctx context.Context, method, path string, payload any, key string) (json.RawMessage, ResponseMeta, error) {
	return c.requestWithHeadersRequirement(ctx, method, path, payload, key, nil, true)
}

func (c *Client) requestWithHeaders(ctx context.Context, method, path string, payload any, key string, headers http.Header) (json.RawMessage, ResponseMeta, error) {
	return c.requestWithHeadersRequirement(ctx, method, path, payload, key, headers, true)
}

func (c *Client) requestWithHeadersRequirement(ctx context.Context, method, path string, payload any, key string, headers http.Header, requireKey bool) (json.RawMessage, ResponseMeta, error) {
	key = strings.TrimSpace(key)
	if method != "GET" && requireKey && !idempotencyKeyPattern.MatchString(key) {
		return nil, ResponseMeta{}, errors.New("idempotency key must match ^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$")
	}
	body, err := json.Marshal(payload)
	if payload == nil {
		body = nil
	}
	if err != nil {
		return nil, ResponseMeta{}, err
	}
	if len(body) > c.maxRequestBodyBytes() {
		return nil, ResponseMeta{}, errors.New("Clover request body exceeds the configured limit")
	}
	// Only routes whose contract requires idempotency are retried after a
	// transient response. An optional caller-supplied key is forwarded for
	// observability but must not upgrade an otherwise non-idempotent mutation
	// into a retryable request.
	safe := method == "GET" || (key != "" && requireKey)
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
		for name, values := range headers {
			for _, value := range values {
				req.Header.Add(name, value)
			}
		}
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
			return nil, ResponseMeta{}, errors.New(c.safeErrorMessage(err.Error()))
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
		problem = c.safeProblem(problem)
		return nil, meta, &APIError{Status: response.StatusCode, Problem: problem, Metadata: meta, Message: c.safeErrorMessage(message)}
	}
}

func unwrapEnvelope(body []byte) ([]byte, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return []byte("{}"), nil
	}
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

func (c *Client) maxRequestBodyBytes() int {
	if c.MaxRequestBodyBytes > 0 {
		return c.MaxRequestBodyBytes
	}
	return DefaultMaxRequestBodyBytes
}

func (c *Client) safeErrorMessage(message string) string {
	if c.APIKey == "" {
		if sensitiveText(message) {
			return "[REDACTED]"
		}
		return message
	}
	message = strings.ReplaceAll(message, c.APIKey, "[REDACTED]")
	if sensitiveText(message) {
		return "[REDACTED]"
	}
	return message
}

func (c *Client) safeProblem(problem Problem) Problem {
	if problem == nil {
		return nil
	}
	value := redactJSONValue("", map[string]any(problem), c.APIKey)
	if sanitized, ok := value.(map[string]any); ok {
		return Problem(sanitized)
	}
	return nil
}

func redactJSONValue(key string, value any, secret string) any {
	keyLower := strings.ToLower(strings.TrimSpace(key))
	if keyLower != "" && (strings.Contains(keyLower, "secret") || strings.Contains(keyLower, "token") || strings.Contains(keyLower, "password") || strings.Contains(keyLower, "authorization") || strings.Contains(keyLower, "api_key") || strings.Contains(keyLower, "raw") || strings.Contains(keyLower, "body") || strings.Contains(keyLower, "payload") || strings.Contains(keyLower, "header")) {
		return "[REDACTED]"
	}
	switch typed := value.(type) {
	case string:
		if secret != "" {
			typed = strings.ReplaceAll(typed, secret, "[REDACTED]")
		}
		if sensitiveText(typed) {
			return "[REDACTED]"
		}
		return typed
	case map[string]any:
		output := make(map[string]any, len(typed))
		for name, nested := range typed {
			output[name] = redactJSONValue(name, nested, secret)
		}
		return output
	case []any:
		output := make([]any, len(typed))
		for index, nested := range typed {
			output[index] = redactJSONValue("", nested, secret)
		}
		return output
	default:
		return value
	}
}

func sensitiveText(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{"secret", "token", "password", "authorization", "api_key", "private key", "bearer "} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
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
	meta := ResponseMeta{RequestID: headers.Get("X-Request-ID"), NextCursor: headers.Get("X-Next-Cursor"), Replayed: strings.EqualFold(headers.Get("Idempotency-Replayed"), "true")}
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
