package clover

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type typedTransport struct {
	response *http.Response
	request  *http.Request
	body     []byte
}

func (t *typedTransport) Do(request *http.Request) (*http.Response, error) {
	if err := request.Context().Err(); err != nil {
		return nil, err
	}
	t.request = request
	if request.Body != nil {
		t.body, _ = io.ReadAll(request.Body)
		request.Body = io.NopCloser(bytes.NewReader(t.body))
	}
	return t.response, nil
}

func typedResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func TestDomainsTypedServiceUsesCanonicalPathQueryAndEnvelope(t *testing.T) {
	transport := &typedTransport{response: typedResponse(http.StatusOK, `{"success":true,"requestId":"req_domains","data":{"items":[{"id":"dom_1","name":"mail.example.com","status":"verified"}],"pagination":{"page":2,"limit":50,"total":1,"hasNext":false,"hasPrevious":true},"future_field":"kept"}}`)}
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = transport
	client.MaxRetries = 0
	result, meta, err := client.Domains.List(context.Background(), DomainListOptions{Page: 2, Limit: 50, Status: "verified", Provider: "cloudflare", SendingEnabled: boolPtr(true), ReceivingEnabled: boolPtr(false)})
	if err != nil {
		t.Fatal(err)
	}
	if result.Items[0].ID != "dom_1" || result.Pagination.Page != 2 {
		t.Fatalf("decoded result = %#v", result)
	}
	if got := transport.request.URL.Path; got != "/api/v1/domains" {
		t.Fatalf("path = %q", got)
	}
	if got := transport.request.URL.Query().Get("sendingEnabled"); got != "true" {
		t.Fatalf("sendingEnabled = %q", got)
	}
	if got := transport.request.URL.Query().Get("receivingEnabled"); got != "false" {
		t.Fatalf("receivingEnabled = %q", got)
	}
	if meta.RequestID != "req_domains" {
		t.Fatalf("request id = %q", meta.RequestID)
	}
	if !bytes.Contains(meta.RawBody, []byte(`"future_field"`)) {
		t.Fatal("raw response did not preserve unknown field")
	}
}

func TestTemplateRollbackUsesCanonicalRouteAndScope(t *testing.T) {
	transport := &typedTransport{response: typedResponse(http.StatusOK, `{"success":true,"data":{"id":"v_2","status":"published"}}`)}
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = transport
	client.MaxRetries = 0
	result, _, err := client.Templates.Rollback(context.Background(), "template/a", "version/2", "rollback-1234", scopedTestOptions())
	if err != nil {
		t.Fatal(err)
	}
	if result.ID != "v_2" {
		t.Fatalf("result = %#v", result)
	}
	if got := transport.request.URL.EscapedPath(); got != "/api/v1/templates/template%2Fa/versions/version%2F2/rollback" {
		t.Fatalf("escaped path = %q", got)
	}
	if got := transport.request.Header.Get("Idempotency-Key"); got != "rollback-1234" {
		t.Fatalf("idempotency key = %q", got)
	}
	if got := transport.request.URL.Query().Get("project_id"); got != "project_1" {
		t.Fatalf("project_id = %q", got)
	}
}

func TestTemplateVersionGetAndCompareExposeCanonicalSelectors(t *testing.T) {
	transport := &typedTransport{response: typedResponse(http.StatusOK, `{"success":true,"data":{"template_id":"template_1","from":1,"to":2,"source_digest_changed":true}}`)}
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = transport
	client.MaxRetries = 0
	comparison, _, err := client.Templates.Compare(context.Background(), "template_1", "latest", "2", scopedTestOptions())
	if err != nil {
		t.Fatal(err)
	}
	if comparison.TemplateID != "template_1" || !comparison.SourceDigestChanged {
		t.Fatalf("comparison = %#v", comparison)
	}
	query := transport.request.URL.Query()
	if query.Get("from") != "latest" || query.Get("to") != "2" || query.Get("project_id") != "project_1" {
		t.Fatalf("compare query = %v", query)
	}

	transport.response = typedResponse(http.StatusOK, `{"success":true,"data":{"id":"version_2","compiler":"mjml","source_digest":"sha256:abc"}}`)
	version, _, err := client.Templates.GetVersion(context.Background(), "template_1", "version_2", scopedTestOptions())
	if err != nil {
		t.Fatal(err)
	}
	if version.Compiler != "mjml" || version.SourceDigest != "sha256:abc" {
		t.Fatalf("version artifacts = %#v", version)
	}
}

func TestAutomationUpdateUsesBodyAndQueryScope(t *testing.T) {
	transport := &typedTransport{response: typedResponse(http.StatusOK, `{"success":true,"data":{"id":"automation_1","name":"Updated flow","status":"draft"}}`)}
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = transport
	client.MaxRetries = 0
	request := UpdateAutomationRequest{
		Scope:       scopeTest(),
		Name:        "Updated flow",
		Description: "Updated description",
		Trigger:     AutomationTrigger{Kind: "manual"},
		Steps: []AutomationStep{{
			ID:       "exit",
			Position: 0,
			Action:   AutomationAction{Kind: "exit"},
		}},
	}
	result, _, err := client.Automations.Update(context.Background(), "automation/1", request, "automation-update-1234")
	if err != nil {
		t.Fatal(err)
	}
	if result.ID != "automation_1" || result.Name != "Updated flow" {
		t.Fatalf("result = %#v", result)
	}
	if transport.request.Method != http.MethodPatch || transport.request.URL.EscapedPath() != "/api/v1/automations/automation%2F1" {
		t.Fatalf("request = %s %s", transport.request.Method, transport.request.URL.RequestURI())
	}
	if transport.request.Header.Get("Idempotency-Key") != "automation-update-1234" {
		t.Fatalf("idempotency key = %q", transport.request.Header.Get("Idempotency-Key"))
	}
	query := transport.request.URL.Query()
	if query.Get("project_id") != "project_1" || query.Get("environment") != "development" || query.Get("tenant_id") != "tenant_1" {
		t.Fatalf("scope query = %v", query)
	}
	var body UpdateAutomationRequest
	if err := json.Unmarshal(transport.body, &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Scope != request.Scope || body.Name != request.Name || body.Description != request.Description || body.Trigger.Kind != "manual" || len(body.Steps) != 1 {
		t.Fatalf("body = %#v", body)
	}
}

func TestAutomationUpdateRejectsMissingScopeBeforeTransport(t *testing.T) {
	transport := &typedTransport{response: typedResponse(http.StatusOK, `{"success":true,"data":{}}`)}
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = transport
	client.MaxRetries = 0
	_, _, err := client.Automations.Update(context.Background(), "automation_1", UpdateAutomationRequest{}, "automation-update-1234")
	if !errors.Is(err, ErrMissingScope) {
		t.Fatalf("error = %v", err)
	}
	if transport.request != nil {
		t.Fatal("missing scope reached transport")
	}
}

func TestEmailTraceAndReplayDecodeTypedEvidence(t *testing.T) {
	transport := &typedTransport{response: typedResponse(http.StatusOK, `{"success":true,"data":{"email_id":"email_1","complete":true,"steps":[{"event_id":"event_1","type":"delivered","status":"ok"}]}}`)}
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = transport
	client.MaxRetries = 0
	trace, _, err := client.Emails.Trace(context.Background(), "email_1", scopedTestOptions())
	if err != nil {
		t.Fatal(err)
	}
	if trace.EmailID != "email_1" || !trace.Complete || len(trace.Steps) != 1 {
		t.Fatalf("trace = %#v", trace)
	}

	transport.response = typedResponse(http.StatusAccepted, `{"success":true,"data":{"id":"replay_1","email_id":"email_1","status":"accepted"}}`)
	plan, _, err := client.Emails.Replay(context.Background(), "email_1", ReplayEmailRequest{Reason: "retry"}, "replay-1234", scopedTestOptions())
	if err != nil {
		t.Fatal(err)
	}
	if plan.ID != "replay_1" || plan.Status != "accepted" {
		t.Fatalf("replay plan = %#v", plan)
	}
}

func TestAuditListDecodesCursorPageEnvelope(t *testing.T) {
	transport := &typedTransport{response: typedResponse(http.StatusOK, `{"success":true,"data":{"data":[{"id":"audit_1","action":"send","sequence":7}],"next_cursor":"next_1"}}`)}
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = transport
	client.MaxRetries = 0
	page, _, err := client.Audit.List(context.Background(), AuditListOptions{ProjectID: "project_1", Environment: "development"})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Data) != 1 || page.Data[0].Sequence != 7 || page.NextCursor == nil || *page.NextCursor != "next_1" {
		t.Fatalf("audit page = %#v", page)
	}
}

func TestDNSProvisionAndDomainHealthUseCanonicalContracts(t *testing.T) {
	transport := &typedTransport{response: typedResponse(http.StatusAccepted, `{"success":true,"data":{"domainId":"domain_1","status":"accepted","dnsRecords":[]}}`)}
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = transport
	client.MaxRetries = 0
	if _, _, err := client.Domains.ProvisionDNS(context.Background(), "domain_1", ProvisionDNSRequest{Force: Bool(true)}, "dns-provision-1234"); err != nil {
		t.Fatal(err)
	}
	if transport.request.Method != http.MethodPost || transport.request.URL.Path != "/api/v1/domains/domain_1/dns-records/provision" || transport.request.Header.Get("Idempotency-Key") != "dns-provision-1234" {
		t.Fatalf("dns provision request = %s %s", transport.request.Method, transport.request.URL.String())
	}

	transport.response = typedResponse(http.StatusOK, `{"success":true,"data":{"data":[],"next_cursor":"next-health"}}`)
	page, _, err := client.DomainHealth.List(context.Background(), DomainHealthListOptions{ProjectID: "project_1", Environment: "development"})
	if err != nil {
		t.Fatal(err)
	}
	if page.NextCursor == nil || *page.NextCursor != "next-health" {
		t.Fatalf("health page = %#v", page)
	}
	if transport.request.Method != http.MethodGet || transport.request.URL.Path != "/api/v1/domain-health" || transport.request.URL.Query().Get("project_id") != "project_1" {
		t.Fatalf("domain health request = %s %s", transport.request.Method, transport.request.URL.String())
	}
}

func TestBroadResourceMethodsUseExpectedPaths(t *testing.T) {
	tests := []struct {
		name   string
		call   func(*Client) error
		path   string
		method string
	}{
		{"webhook replay", func(c *Client) error {
			_, _, err := c.Webhooks.ReplayDelivery(context.Background(), "del_1", "webhook-1234")
			return err
		}, "/api/v1/webhook-deliveries/del_1/replay", http.MethodPost},
		{"inbound attachment", func(c *Client) error {
			_, _, err := c.Inbound.AttachmentURL(context.Background(), "mail_1", "att_1")
			return err
		}, "/api/v1/received-emails/mail_1/attachments/att_1", http.MethodGet},
		{"suppression reactivate", func(c *Client) error {
			_, err := c.Suppressions.Reactivate(context.Background(), "sup_1", "suppress-1234")
			return err
		}, "/api/v1/suppressions/sup_1", http.MethodDelete},
		{"preference unsubscribe", func(c *Client) error {
			_, err := c.Preferences.Unsubscribe(context.Background(), "token/one", "pref-1234")
			return err
		}, "/api/v1/unsubscribe/token%2Fone", http.MethodPost},
		{"audit release", func(c *Client) error {
			_, _, err := c.Audit.ReleaseHold(context.Background(), "hold_1", scopedTestOptions(), "audit-1234")
			return err
		}, "/api/v1/audit-events/holds/hold_1", http.MethodDelete},
		{"automation event", func(c *Client) error {
			_, _, err := c.Automations.IngestEvent(context.Background(), IngestAutomationEventRequest{EventType: "signup"}, scopedTestOptions(), "auto-1234")
			return err
		}, "/api/v1/automation-events", http.MethodPost},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			transport := &typedTransport{response: typedResponse(http.StatusNoContent, "")}
			client := NewClient("https://api.example.test", "secret")
			client.HTTPClient = transport
			client.MaxRetries = 0
			if err := test.call(client); err != nil {
				t.Fatal(err)
			}
			if transport.request.URL.EscapedPath() != test.path {
				t.Fatalf("path = %q, want %q", transport.request.URL.EscapedPath(), test.path)
			}
			if transport.request.Method != test.method {
				t.Fatalf("method = %q, want %q", transport.request.Method, test.method)
			}
		})
	}
}

func TestIdempotentMutationRejectsMissingKeyBeforeTransport(t *testing.T) {
	transport := &typedTransport{response: typedResponse(http.StatusOK, `{"success":true,"data":{}}`)}
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = transport
	if _, _, err := client.Domains.ProvisionDNS(context.Background(), "domain_1", ProvisionDNSRequest{}, ""); !errors.Is(err, ErrInvalidIdempotencyKey) {
		t.Fatalf("error = %v", err)
	}
	if transport.request != nil {
		t.Fatal("request left process despite invalid key")
	}
}

func TestPublicClientOmitsAuthorizationForOneClickRoutes(t *testing.T) {
	transport := &typedTransport{response: typedResponse(http.StatusNoContent, "")}
	client := NewPublicClient("https://api.example.test")
	client.HTTPClient = transport
	client.MaxRetries = 0
	if _, err := client.Preferences.Unsubscribe(context.Background(), "public-token", "public-1234"); err != nil {
		t.Fatal(err)
	}
	if got := transport.request.Header.Get("Authorization"); got != "" {
		t.Fatalf("authorization = %q", got)
	}
}

func TestScopedMutationAddsRequiredQueryAndMissingScopeStopsTransport(t *testing.T) {
	transport := &typedTransport{response: typedResponse(http.StatusOK, `{"success":true,"data":{"id":"template_1"}}`)}
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = transport
	client.MaxRetries = 0
	if _, _, err := client.Templates.Update(context.Background(), "template_1", TemplateTransitionRequest{Event: "publish"}, "template-1234", scopedTestOptions()); err != nil {
		t.Fatal(err)
	}
	if got := transport.request.URL.Query().Get("project_id"); got != "project_1" {
		t.Fatalf("project_id = %q", got)
	}
	if got := transport.request.URL.Query().Get("environment"); got != "development" {
		t.Fatalf("environment = %q", got)
	}
	if got := transport.request.URL.Query().Get("tenant_id"); got != "tenant_1" {
		t.Fatalf("tenant_id = %q", got)
	}

	transport.request = nil
	if _, _, err := client.Templates.Update(context.Background(), "template_1", TemplateTransitionRequest{}, "template-1234"); !errors.Is(err, ErrMissingScope) {
		t.Fatalf("missing scope error = %v", err)
	}
	if transport.request != nil {
		t.Fatal("missing scope reached transport")
	}
}

func TestScopedOptionsValidateRequiredValues(t *testing.T) {
	for _, test := range []struct {
		name          string
		options       ScopedPageOptions
		requireTenant bool
		want          error
	}{
		{"missing project", ScopedPageOptions{Environment: "development"}, false, ErrMissingScope},
		{"invalid environment", ScopedPageOptions{ProjectID: "project_1", Environment: "sandbox"}, false, ErrInvalidScope},
		{"missing tenant for list", ScopedPageOptions{ProjectID: "project_1", Environment: "development"}, true, ErrMissingScope},
		{"limit too high", ScopedPageOptions{ProjectID: "project_1", Environment: "development", Limit: 101}, false, ErrInvalidScope},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.options.Validate(test.requireTenant); !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestSecretBearingMutationDoesNotRetainRawBody(t *testing.T) {
	transport := &typedTransport{response: typedResponse(http.StatusCreated, `{"success":true,"data":{"key":{"id":"key_1"},"token":"clv_secret_once"}}`)}
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = transport
	client.MaxRetries = 0
	result, meta, err := client.APIKeys.Create(context.Background(), CreateAPIKeyRequest{Name: "one-time"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Token != "clv_secret_once" {
		t.Fatalf("token = %q", result.Token)
	}
	if len(meta.RawBody) != 0 {
		t.Fatalf("secret mutation retained raw body: %q", meta.RawBody)
	}
}

func TestRoutesWithoutBackendIdempotencyAcceptNoKey(t *testing.T) {
	tests := []struct {
		name string
		call func(*Client) error
	}{
		{"domain create", func(c *Client) error {
			_, _, err := c.Domains.Create(context.Background(), CreateDomainRequest{Name: "mail.example.com"})
			return err
		}},
		{"api key create", func(c *Client) error {
			_, _, err := c.APIKeys.Create(context.Background(), CreateAPIKeyRequest{Name: "local"})
			return err
		}},
		{"suppression delete", func(c *Client) error {
			_, err := c.Suppressions.Delete(context.Background(), "suppression_1")
			return err
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			transport := &typedTransport{response: typedResponse(http.StatusNoContent, "")}
			client := NewClient("https://api.example.test", "secret")
			client.HTTPClient = transport
			client.MaxRetries = 0
			if err := test.call(client); err != nil {
				t.Fatal(err)
			}
			if got := transport.request.Header.Get("Idempotency-Key"); got != "" {
				t.Fatalf("unexpected idempotency key %q", got)
			}
		})
	}
}

func TestProviderRoutingServicesUseCanonicalPathsAndTypedEnvelopes(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		call   func(*Client) error
		path   string
		method string
		query  func(*testing.T, *http.Request)
	}{
		{"policy get", `{"success":true,"data":{"organizationId":"org_1"}}`, func(c *Client) error { _, _, err := c.Routing.GetPolicy(context.Background()); return err }, "/api/v1/routing/policy", http.MethodGet, nil},
		{"policy put", `{"success":true,"data":{"organizationId":"org_1"}}`, func(c *Client) error {
			_, _, err := c.Routing.PutPolicy(context.Background(), RoutingPolicyRequest{PreferredRegion: "us-east"}, "routing-policy-1234")
			return err
		}, "/api/v1/routing/policy", http.MethodPut, nil},
		{"capabilities", `{"success":true,"data":{"items":[]}}`, func(c *Client) error { _, _, err := c.Routing.ListCapabilities(context.Background()); return err }, "/api/v1/routing/capabilities", http.MethodGet, nil},
		{"pools", `{"success":true,"data":{"items":[]}}`, func(c *Client) error { _, _, err := c.Routing.ListPools(context.Background()); return err }, "/api/v1/routing/pools", http.MethodGet, nil},
		{"pool create", `{"success":true,"data":{"id":"pool_1"}}`, func(c *Client) error {
			_, _, err := c.Routing.CreatePool(context.Background(), CreatePoolRequest{Provider: "ses"}, "routing-pool-1234")
			return err
		}, "/api/v1/routing/pools", http.MethodPost, nil},
		{"pool get", `{"success":true,"data":{"id":"pool_1"}}`, func(c *Client) error { _, _, err := c.Routing.GetPool(context.Background(), "pool_1"); return err }, "/api/v1/routing/pools/pool_1", http.MethodGet, nil},
		{"pool command", `{"success":true,"data":{"id":"transition_1"}}`, func(c *Client) error {
			_, _, err := c.Routing.ApplyPoolCommand(context.Background(), "pool_1", PoolCommandRequest{Action: "warmup"}, "routing-command-1234")
			return err
		}, "/api/v1/routing/pools/pool_1/command", http.MethodPost, nil},
		{"ip command", `{"success":true,"data":{"id":"transition_1"}}`, func(c *Client) error {
			_, _, err := c.Routing.ApplyIPCommand(context.Background(), "pool_1", "ip_1", IPCommandRequest{Action: "hold"}, "routing-ip-1234")
			return err
		}, "/api/v1/routing/pools/pool_1/ips/ip_1/command", http.MethodPost, nil},
		{"audit", `{"success":true,"data":{"items":[]}}`, func(c *Client) error {
			_, _, err := c.Routing.ListAudit(context.Background(), RoutingAuditOptions{EntityType: "pool", EntityID: "pool_1"})
			return err
		}, "/api/v1/routing/audit", http.MethodGet, func(t *testing.T, request *http.Request) {
			if request.URL.Query().Get("entityType") != "pool" || request.URL.Query().Get("entityId") != "pool_1" {
				t.Fatalf("routing audit query = %v", request.URL.Query())
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			transport := &typedTransport{response: typedResponse(http.StatusOK, test.body)}
			client := NewClient("https://api.example.test", "secret")
			client.HTTPClient = transport
			client.MaxRetries = 0
			if err := test.call(client); err != nil {
				t.Fatal(err)
			}
			if transport.request.Method != test.method || transport.request.URL.Path != test.path {
				t.Fatalf("request = %s %s", transport.request.Method, transport.request.URL.String())
			}
			if test.query != nil {
				test.query(t, transport.request)
			}
			if test.method == http.MethodGet && transport.request.Header.Get("Idempotency-Key") != "" {
				t.Fatalf("unexpected idempotency header: %q", transport.request.Header.Get("Idempotency-Key"))
			}
			if test.method != http.MethodGet && transport.request.Header.Get("Idempotency-Key") == "" {
				t.Fatal("missing routing mutation idempotency header")
			}
		})
	}
}

type routeCoverageTransport struct{ request *http.Request }

func (t *routeCoverageTransport) Do(request *http.Request) (*http.Response, error) {
	t.request = request
	if request.Method == http.MethodDelete || request.URL.Path == "/api/v1/unsubscribe/public-token" {
		return typedResponse(http.StatusNoContent, ""), nil
	}
	data := `{}`
	switch {
	case request.URL.Path == "/api/v1/templates/template_1/versions" && request.Method == http.MethodGet:
		data = `[]`
	case request.URL.Path == "/api/v1/domains/domain_1/dns-records":
		data = `{"items":[]}`
	case strings.HasSuffix(request.URL.Path, "/trace"), strings.HasSuffix(request.URL.Path, "/evaluate"):
		data = `{"items":[],"pagination":{}}`
	case request.URL.Path == "/api/v1/domains", request.URL.Path == "/api/v1/emails", request.URL.Path == "/api/v1/api-keys", request.URL.Path == "/api/v1/webhooks", request.URL.Path == "/api/v1/webhook-deliveries", request.URL.Path == "/api/v1/received-emails", request.URL.Path == "/api/v1/suppressions", request.URL.Path == "/api/v1/preference-topics", request.URL.Path == "/api/v1/logs", request.URL.Path == "/api/v1/templates", request.URL.Path == "/api/v1/contacts", request.URL.Path == "/api/v1/segments", request.URL.Path == "/api/v1/broadcasts", request.URL.Path == "/api/v1/automations", request.URL.Path == "/api/v1/audit-events", request.URL.Path == "/api/v1/audit-events/holds":
		data = `{"items":[],"pagination":{}}`
	}
	return typedResponse(http.StatusOK, `{"success":true,"data":`+data+`}`), nil
}

func TestCoreResourceMethodsRemainRouteComplete(t *testing.T) {
	key := "route-1234"
	now := timeNowForTest()
	methods := []struct {
		name string
		call func(*Client) error
	}{
		{"email send", func(c *Client) error {
			_, _, err := c.Emails.Send(context.Background(), SendEmailRequest{}, key)
			return err
		}},
		{"email batch", func(c *Client) error { _, _, err := c.Emails.SendBatch(context.Background(), nil, key); return err }},
		{"email schedule", func(c *Client) error {
			_, _, err := c.Emails.Schedule(context.Background(), "email_1", ScheduleEmailRequest{ScheduledAt: now}, key)
			return err
		}},
		{"email cancel", func(c *Client) error { _, _, err := c.Emails.Cancel(context.Background(), "email_1", key); return err }},
		{"email get", func(c *Client) error { _, _, err := c.Emails.Get(context.Background(), "email_1"); return err }},
		{"email list", func(c *Client) error {
			_, _, err := c.Emails.List(context.Background(), EmailListOptions{})
			return err
		}},
		{"email replay", func(c *Client) error {
			_, _, err := c.Emails.Replay(context.Background(), "email_1", ReplayEmailRequest{}, key, scopedTestOptions())
			return err
		}},
		{"email trace", func(c *Client) error {
			_, _, err := c.Emails.Trace(context.Background(), "email_1", scopedTestOptions())
			return err
		}},
		{"attachment upload", func(c *Client) error {
			_, _, err := c.Emails.UploadAttachment(context.Background(), AttachmentUploadRequest{}, key)
			return err
		}},
		{"domain list", func(c *Client) error {
			_, _, err := c.Domains.List(context.Background(), DomainListOptions{})
			return err
		}},
		{"domain create", func(c *Client) error {
			_, _, err := c.Domains.Create(context.Background(), CreateDomainRequest{}, key)
			return err
		}},
		{"domain get", func(c *Client) error { _, _, err := c.Domains.Get(context.Background(), "domain_1"); return err }},
		{"domain update", func(c *Client) error {
			_, _, err := c.Domains.Update(context.Background(), "domain_1", UpdateDomainRequest{}, key)
			return err
		}},
		{"domain delete", func(c *Client) error { _, err := c.Domains.Delete(context.Background(), "domain_1", key); return err }},
		{"domain verify", func(c *Client) error {
			_, _, err := c.Domains.Verify(context.Background(), "domain_1", VerifyDomainRequest{}, key)
			return err
		}},
		{"dns records", func(c *Client) error { _, _, err := c.Domains.DNSRecords(context.Background(), "domain_1"); return err }},
		{"dns provision", func(c *Client) error {
			_, _, err := c.Domains.ProvisionDNS(context.Background(), "domain_1", ProvisionDNSRequest{}, key)
			return err
		}},
		{"domain health list", func(c *Client) error {
			_, _, err := c.DomainHealth.List(context.Background(), DomainHealthListOptions{ProjectID: "project_1", Environment: "development"})
			return err
		}},
		{"domain health get", func(c *Client) error {
			_, _, err := c.DomainHealth.Get(context.Background(), "domain_1", scopedTestOptions())
			return err
		}},
		{"domain health verify", func(c *Client) error {
			_, _, err := c.DomainHealth.Verify(context.Background(), GetDomainHealthRequest{DomainID: "domain_1"}, key, scopedTestOptions())
			return err
		}},
		{"api key list", func(c *Client) error {
			_, _, err := c.APIKeys.List(context.Background(), APIKeyListOptions{})
			return err
		}},
		{"api key create", func(c *Client) error {
			_, _, err := c.APIKeys.Create(context.Background(), CreateAPIKeyRequest{}, key)
			return err
		}},
		{"api key update", func(c *Client) error {
			_, _, err := c.APIKeys.Update(context.Background(), "key_1", UpdateAPIKeyRequest{}, key)
			return err
		}},
		{"api key revoke", func(c *Client) error { _, err := c.APIKeys.Revoke(context.Background(), "key_1", key); return err }},
		{"webhook list", func(c *Client) error {
			_, _, err := c.Webhooks.List(context.Background(), WebhookListOptions{})
			return err
		}},
		{"webhook create", func(c *Client) error {
			_, _, err := c.Webhooks.Create(context.Background(), CreateWebhookRequest{}, key)
			return err
		}},
		{"webhook get", func(c *Client) error { _, _, err := c.Webhooks.Get(context.Background(), "hook_1"); return err }},
		{"webhook update", func(c *Client) error {
			_, _, err := c.Webhooks.Update(context.Background(), "hook_1", UpdateWebhookRequest{}, key)
			return err
		}},
		{"webhook delete", func(c *Client) error { _, err := c.Webhooks.Delete(context.Background(), "hook_1", key); return err }},
		{"webhook rotate", func(c *Client) error {
			_, _, err := c.Webhooks.RotateSecret(context.Background(), "hook_1", nil, key)
			return err
		}},
		{"delivery list", func(c *Client) error {
			_, _, err := c.Webhooks.Deliveries(context.Background(), DeliveryListOptions{})
			return err
		}},
		{"delivery get", func(c *Client) error {
			_, _, err := c.Webhooks.Delivery(context.Background(), "delivery_1")
			return err
		}},
		{"delivery replay", func(c *Client) error {
			_, _, err := c.Webhooks.ReplayDelivery(context.Background(), "delivery_1", key)
			return err
		}},
		{"logs", func(c *Client) error { _, _, err := c.Logs.List(context.Background(), LogListOptions{}); return err }},
		{"metrics", func(c *Client) error {
			_, _, err := c.Metrics.Email(context.Background(), MetricsOptions{})
			return err
		}},
		{"inbound list", func(c *Client) error {
			_, _, err := c.Inbound.List(context.Background(), InboundListOptions{})
			return err
		}},
		{"inbound get", func(c *Client) error { _, _, err := c.Inbound.Get(context.Background(), "mail_1"); return err }},
		{"inbound attachment", func(c *Client) error {
			_, _, err := c.Inbound.AttachmentURL(context.Background(), "mail_1", "attachment_1")
			return err
		}},
		{"inbound accept", func(c *Client) error {
			_, _, err := c.Inbound.Accept(context.Background(), "cloudflare", nil, key)
			return err
		}},
		{"provider event", func(c *Client) error {
			_, _, err := c.ProviderEvents.Accept(context.Background(), "cloudflare", nil, key)
			return err
		}},
		{"suppression list", func(c *Client) error {
			_, _, err := c.Suppressions.List(context.Background(), SuppressionListOptions{})
			return err
		}},
		{"suppression create", func(c *Client) error {
			_, _, err := c.Suppressions.Create(context.Background(), CreateSuppressionRequest{}, key)
			return err
		}},
		{"suppression delete", func(c *Client) error {
			_, err := c.Suppressions.Delete(context.Background(), "suppression_1", key)
			return err
		}},
		{"topics", func(c *Client) error { _, _, err := c.Preferences.Topics(context.Background()); return err }},
		{"topic create", func(c *Client) error {
			_, _, err := c.Preferences.CreateTopic(context.Background(), CreatePreferenceTopicRequest{}, key)
			return err
		}},
		{"preferences get", func(c *Client) error { _, _, err := c.Preferences.Get(context.Background(), "token_1"); return err }},
		{"preferences update", func(c *Client) error {
			_, _, err := c.Preferences.Update(context.Background(), "token_1", UpdatePreferenceRequest{}, key)
			return err
		}},
		{"preferences unsubscribe", func(c *Client) error {
			_, err := c.Preferences.Unsubscribe(context.Background(), "public-token", key)
			return err
		}},
		{"template list", func(c *Client) error {
			_, _, err := c.Templates.List(context.Background(), TemplateListOptions{TenantID: "tenant_1", ProjectID: "project_1", Environment: "development"})
			return err
		}},
		{"template create", func(c *Client) error {
			_, _, err := c.Templates.Create(context.Background(), CreateTemplateRequest{Scope: scopeTest()}, key)
			return err
		}},
		{"template get", func(c *Client) error {
			_, _, err := c.Templates.Get(context.Background(), "template_1", scopedTestOptions())
			return err
		}},
		{"template update", func(c *Client) error {
			_, _, err := c.Templates.Update(context.Background(), "template_1", TemplateTransitionRequest{}, key, scopedTestOptions())
			return err
		}},
		{"template versions", func(c *Client) error {
			_, _, err := c.Templates.Versions(context.Background(), "template_1", scopedTestOptions())
			return err
		}},
		{"template version create", func(c *Client) error {
			_, _, err := c.Templates.CreateVersion(context.Background(), "template_1", CreateTemplateVersionRequest{Scope: scopeTest()}, key)
			return err
		}},
		{"template publish", func(c *Client) error {
			_, _, err := c.Templates.Publish(context.Background(), "template_1", "version_1", key, scopedTestOptions())
			return err
		}},
		{"template version get", func(c *Client) error {
			_, _, err := c.Templates.GetVersion(context.Background(), "template_1", "version_1", scopedTestOptions())
			return err
		}},
		{"template compare", func(c *Client) error {
			_, _, err := c.Templates.Compare(context.Background(), "template_1", "1", "latest", scopedTestOptions())
			return err
		}},
		{"template rollback", func(c *Client) error {
			_, _, err := c.Templates.Rollback(context.Background(), "template_1", "version_1", key, scopedTestOptions())
			return err
		}},
		{"contact list", func(c *Client) error {
			_, _, err := c.Contacts.List(context.Background(), ContactListOptions{TenantID: "tenant_1", ProjectID: "project_1", Environment: "development"})
			return err
		}},
		{"contact create", func(c *Client) error {
			_, _, err := c.Contacts.Create(context.Background(), CreateContactRequest{Scope: scopeTest()}, key)
			return err
		}},
		{"contact get", func(c *Client) error {
			_, _, err := c.Contacts.Get(context.Background(), "contact_1", scopedTestOptions())
			return err
		}},
		{"contact transition", func(c *Client) error {
			_, _, err := c.Contacts.Transition(context.Background(), "contact_1", ContactTransitionRequest{}, key, scopedTestOptions())
			return err
		}},
		{"contact resubscribe", func(c *Client) error {
			_, _, err := c.Contacts.Resubscribe(context.Background(), "contact_1", key, scopedTestOptions())
			return err
		}},
		{"segment list", func(c *Client) error {
			_, _, err := c.Segments.List(context.Background(), SegmentListOptions{TenantID: "tenant_1", ProjectID: "project_1", Environment: "development"})
			return err
		}},
		{"segment create", func(c *Client) error {
			_, _, err := c.Segments.Create(context.Background(), CreateSegmentRequest{Scope: scopeTest()}, key)
			return err
		}},
		{"segment get", func(c *Client) error {
			_, _, err := c.Segments.Get(context.Background(), "segment_1", scopedTestOptions())
			return err
		}},
		{"segment archive", func(c *Client) error {
			_, _, err := c.Segments.Archive(context.Background(), "segment_1", key, scopedTestOptions())
			return err
		}},
		{"segment evaluate", func(c *Client) error {
			_, _, err := c.Segments.Evaluate(context.Background(), "segment_1", scopedTestOptions())
			return err
		}},
		{"broadcast list", func(c *Client) error {
			_, _, err := c.Broadcasts.List(context.Background(), BroadcastListOptions{TenantID: "tenant_1", ProjectID: "project_1", Environment: "development"})
			return err
		}},
		{"broadcast create", func(c *Client) error {
			_, _, err := c.Broadcasts.Create(context.Background(), CreateBroadcastRequest{Scope: scopeTest()}, key)
			return err
		}},
		{"broadcast get", func(c *Client) error {
			_, _, err := c.Broadcasts.Get(context.Background(), "broadcast_1", scopedTestOptions())
			return err
		}},
		{"broadcast schedule", func(c *Client) error {
			_, _, err := c.Broadcasts.Schedule(context.Background(), "broadcast_1", ScheduleBroadcastRequest{ScheduledAt: now}, key, scopedTestOptions())
			return err
		}},
		{"broadcast cancel", func(c *Client) error {
			_, _, err := c.Broadcasts.Cancel(context.Background(), "broadcast_1", key, scopedTestOptions())
			return err
		}},
		{"automation list", func(c *Client) error {
			_, _, err := c.Automations.List(context.Background(), AutomationListOptions{TenantID: "tenant_1", ProjectID: "project_1", Environment: "development"})
			return err
		}},
		{"automation create", func(c *Client) error {
			_, _, err := c.Automations.Create(context.Background(), CreateAutomationRequest{Scope: scopeTest()}, key)
			return err
		}},
		{"automation get", func(c *Client) error {
			_, _, err := c.Automations.Get(context.Background(), "automation_1", scopedTestOptions())
			return err
		}},
		{"automation activate", func(c *Client) error {
			_, _, err := c.Automations.Activate(context.Background(), "automation_1", key, scopedTestOptions())
			return err
		}},
		{"automation pause", func(c *Client) error {
			_, _, err := c.Automations.Pause(context.Background(), "automation_1", key, scopedTestOptions())
			return err
		}},
		{"automation run", func(c *Client) error {
			_, _, err := c.Automations.StartRun(context.Background(), "automation_1", StartAutomationRunRequest{}, key, scopedTestOptions())
			return err
		}},
		{"automation get run", func(c *Client) error {
			_, _, err := c.Automations.GetRun(context.Background(), "automation_1", "run_1", scopedTestOptions())
			return err
		}},
		{"automation event", func(c *Client) error {
			_, _, err := c.Automations.IngestEvent(context.Background(), IngestAutomationEventRequest{}, scopedTestOptions(), key)
			return err
		}},
		{"audit list", func(c *Client) error {
			_, _, err := c.Audit.List(context.Background(), AuditListOptions{ProjectID: "project_1", Environment: "development"})
			return err
		}},
		{"audit append", func(c *Client) error {
			_, _, err := c.Audit.Append(context.Background(), AppendAuditEventRequest{}, key, scopedTestOptions())
			return err
		}},
		{"audit get", func(c *Client) error {
			_, _, err := c.Audit.Get(context.Background(), "audit_1", scopedTestOptions())
			return err
		}},
		{"audit holds", func(c *Client) error {
			_, _, err := c.Audit.Holds(context.Background(), AuditHoldListOptions{ProjectID: "project_1", Environment: "development"})
			return err
		}},
		{"audit create hold", func(c *Client) error {
			_, _, err := c.Audit.CreateHold(context.Background(), CreateAuditHoldRequest{}, key, scopedTestOptions())
			return err
		}},
		{"audit release hold", func(c *Client) error {
			_, _, err := c.Audit.ReleaseHold(context.Background(), "hold_1", scopedTestOptions(), key)
			return err
		}},
	}
	for _, method := range methods {
		t.Run(method.name, func(t *testing.T) {
			transport := &routeCoverageTransport{}
			client := NewClient("https://api.example.test", "secret")
			client.HTTPClient = transport
			client.MaxRetries = 0
			if err := method.call(client); err != nil {
				t.Fatalf("%s: %v", method.name, err)
			}
			if transport.request == nil || !strings.HasPrefix(transport.request.URL.Path, "/api/v1/") {
				t.Fatalf("request path = %v", transport.request)
			}
		})
	}
}

func timeNowForTest() time.Time { return time.Date(2030, time.January, 1, 0, 0, 0, 0, time.UTC) }

func TestTypedErrorAndContextCancellation(t *testing.T) {
	transport := &typedTransport{response: typedResponse(http.StatusForbidden, `{"success":false,"error":{"code":40301,"type":"forbidden","message":"missing scope"},"requestId":"req_denied"}`)}
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = transport
	client.MaxRetries = 0
	_, _, err := client.APIKeys.List(context.Background(), APIKeyListOptions{})
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusForbidden || apiErr.Detail == nil || apiErr.Detail.Message != "missing scope" {
		t.Fatalf("error = %#v", err)
	}
	if apiErr.Meta.RequestID != "req_denied" {
		t.Fatalf("request id = %q", apiErr.Meta.RequestID)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	transport.response = typedResponse(http.StatusOK, `{"success":true,"data":{}}`)
	if _, _, err := client.Domains.List(canceled, DomainListOptions{}); err == nil {
		t.Fatal("canceled context unexpectedly succeeded")
	}
}

func boolPtr(value bool) *bool { return &value }

func scopedTestOptions() ScopedPageOptions {
	return ScopedPageOptions{TenantID: "tenant_1", ProjectID: "project_1", Environment: "development"}
}

func scopeTest() Scope {
	return Scope{OrganizationID: "org_1", TenantID: "tenant_1", ProjectID: "project_1", Environment: "development"}
}
