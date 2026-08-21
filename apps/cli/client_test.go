package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func httpResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func TestSendUsesIdempotencyAndBoundedRetry(t *testing.T) {
	calls := 0
	client := NewClient("https://api.example.test", "secret")
	client.MaxRetries = 1
	client.Sleep = func(context.Context, time.Duration) error { return nil }
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return httpResponse(503, `{"type":"about:blank","title":"Busy","status":503,"code":"BUSY","request_id":"req_12345678","vendor":{"x":1}}`), nil
		}
		response := httpResponse(202, `{"id":"e1","status":"queued","extra":true}`)
		if request.Header.Get("Idempotency-Key") != "idem-1234" {
			t.Errorf("missing idempotency header")
		}
		if request.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("missing auth header")
		}
		return response, nil
	})}
	result, _, err := client.Send(context.Background(), map[string]any{"subject": "hello"}, "idem-1234")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result), `"extra":true`) {
		t.Fatalf("result = %s", result)
	}
	if calls != 2 {
		t.Fatalf("calls = %d", calls)
	}
}

func TestRetryAfterZeroIsHonored(t *testing.T) {
	first := httpResponse(503, `{"type":"about:blank","title":"Busy","status":503,"code":"BUSY"}`)
	first.Header.Set("Retry-After", "0")
	responses := []*http.Response{first, httpResponse(200, `{}`)}
	client := NewClient("https://api.example.test", "secret")
	client.MaxRetries = 1
	client.RetryBaseDelay = time.Second
	var delays []time.Duration
	client.Sleep = func(_ context.Context, delay time.Duration) error {
		delays = append(delays, delay)
		return nil
	}
	call := 0
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		response := responses[call]
		call++
		return response, nil
	})}
	if _, _, err := client.Get(context.Background(), "e1"); err != nil {
		t.Fatal(err)
	}
	if len(delays) != 1 || delays[0] != 0 {
		t.Fatalf("retry delays = %#v", delays)
	}
}

func TestReplayHeaderValueIsCaseInsensitive(t *testing.T) {
	problemResponse := httpResponse(400, `{"type":"about:blank","title":"Bad request"}`)
	problemResponse.Header.Set("Idempotency-Replayed", "TrUe")
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return problemResponse, nil })}
	_, metadata, err := client.Get(context.Background(), "e1")
	if err == nil {
		t.Fatal("expected problem response")
	}
	if !metadata.Replayed {
		t.Fatal("mixed-case replay metadata was not recognized")
	}
}

func TestFollowLogsOffline(t *testing.T) {
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return httpResponse(http.StatusOK, `{"success":true,"data":{"items":[{"id":"e1"},{"id":"e2"}]}}`), nil
	})}
	var events []string
	if err := client.FollowLogs(context.Background(), nil, 100*time.Millisecond, func(event json.RawMessage) error {
		events = append(events, string(event))
		if len(events) == 2 {
			return errors.New("stop after first page")
		}
		return nil
	}); err == nil || err.Error() != "stop after first page" {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("events = %v", events)
	}
}

func TestSendBatchAndScheduleUseCanonicalPaths(t *testing.T) {
	requests := []string{}
	client := NewClient("https://api.example.test", "secret")
	client.Sleep = func(context.Context, time.Duration) error { return nil }
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests = append(requests, request.URL.EscapedPath())
		return httpResponse(202, `{"id":"e1"}`), nil
	})}
	if _, _, err := client.SendBatch(context.Background(), []map[string]any{{"subject": "hello"}}, "idem-batch"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := client.Schedule(context.Background(), "e/1", "2030-01-01T00:00:00Z", "idem-schedule"); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 2 || requests[0] != "/api/v1/emails/batch" || requests[1] != "/api/v1/emails/e%2F1/schedule" {
		t.Fatalf("paths = %#v", requests)
	}
}

func TestNewClientRejectsInvalidConfiguration(t *testing.T) {
	for _, values := range [][2]string{{"", "secret"}, {"ftp://api.example.test", "secret"}, {"https://", "secret"}, {"https://user:pass@api.example.test", "secret"}, {"https://api.example.test?token=leak", "secret"}, {"https://api.example.test", "  "}} {
		t.Run(values[0]+"/"+values[1], func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected invalid configuration to panic")
				}
			}()
			NewClient(values[0], values[1])
		})
	}
}

func TestNewClientWithErrorReturnsConfigurationErrors(t *testing.T) {
	if _, err := NewClientWithError("", "secret"); err == nil || !strings.Contains(err.Error(), "CLOVER_BASE_URL") {
		t.Fatalf("unexpected base URL error: %v", err)
	}
	if _, err := NewClientWithError("https://api.example.test", ""); err == nil || !strings.Contains(err.Error(), "CLOVER_API_KEY") {
		t.Fatalf("unexpected API key error: %v", err)
	}
}

func TestIdempotencyKeyBoundaries(t *testing.T) {
	client := NewClient("https://api.example.test", "secret")
	for _, key := range []string{strings.Repeat("a", 7), strings.Repeat("a", 129), "a bad-key", "_" + strings.Repeat("a", 8)} {
		if _, _, err := client.Send(context.Background(), map[string]any{"subject": "hello"}, key); err == nil {
			t.Fatalf("accepted invalid key %q", key)
		}
	}
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return httpResponse(202, `{}`), nil })}
	for _, key := range []string{strings.Repeat("a", 8), strings.Repeat("a", 128)} {
		if _, _, err := client.Send(context.Background(), map[string]any{"subject": "hello"}, key); err != nil {
			t.Fatalf("valid key rejected: %v", err)
		}
	}
}

func TestSendBatchPreservesScheduledAt(t *testing.T) {
	var body string
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		bytes, _ := io.ReadAll(request.Body)
		body = string(bytes)
		return httpResponse(202, `{}`), nil
	})}
	if _, _, err := client.SendBatch(context.Background(), []map[string]any{{"subject": "hello", "scheduled_at": "2030-01-01"}}, "batch-1234"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "scheduled_at") {
		t.Fatalf("scheduled_at was stripped: %s", body)
	}
}

func TestOversizedResponseFailsBeforeDecode(t *testing.T) {
	client := NewClient("https://api.example.test", "secret")
	client.MaxResponseBodyBytes = 8
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return httpResponse(200, `{"data":"too long"}`), nil
	})}
	_, _, err := client.Get(context.Background(), "e1")
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.Status != 200 || !strings.Contains(apiErr.Message, "exceeds the configured limit") {
		t.Fatalf("error = %#v", err)
	}
}

func TestFollowLogsReportsBodyOverflow(t *testing.T) {
	client := NewClient("https://api.example.test", "secret")
	client.MaxResponseBodyBytes = 8
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return httpResponse(http.StatusOK, `{"success":true,"data":{"items":[{"id":"e1"}]}}`), nil
	})}
	err := client.FollowLogs(context.Background(), nil, 100*time.Millisecond, func(json.RawMessage) error { return nil })
	apiErr, ok := err.(*APIError)
	if !ok || !strings.Contains(apiErr.Message, "response body exceeds") {
		t.Fatalf("error = %#v", err)
	}
}

func TestStreamEventsRejectsLegacySSEPath(t *testing.T) {
	client := NewClient("https://api.example.test", "secret")
	if err := client.StreamEvents(context.Background(), "/v1/events/stream", func(json.RawMessage) error { return nil }); err == nil || !strings.Contains(err.Error(), "SSE streaming is unsupported") {
		t.Fatalf("error = %v", err)
	}
}

func TestPhase1ResourceMethodsUseDocumentedPathsAndScope(t *testing.T) {
	type observed struct {
		method string
		path   string
		query  string
		header http.Header
		body   string
	}
	var requests []observed
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(request.Body)
		requests = append(requests, observed{method: request.Method, path: request.URL.EscapedPath(), query: request.URL.RawQuery, header: request.Header.Clone(), body: string(body)})
		return httpResponse(200, `{"success":true,"data":{"items":[]}}`), nil
	})}
	scope := Scope{ProjectID: "project-1", Environment: "staging", TenantID: "tenant-1"}
	calls := []struct {
		name string
		call func() error
	}{
		{name: "domain list", call: func() error {
			_, _, err := client.ListDomains(context.Background(), url.Values{"status": {"verified"}})
			return err
		}},
		{name: "domain get", call: func() error { _, _, err := client.GetDomain(context.Background(), "domain-1"); return err }},
		{name: "domain create", call: func() error {
			_, _, err := client.CreateDomain(context.Background(), json.RawMessage(`{"name":"example.test"}`), "domain-1234")
			return err
		}},
		{name: "domain configure", call: func() error {
			_, _, err := client.ConfigureDomain(context.Background(), "domain-1", json.RawMessage(`{"sending_enabled":true}`), "config-1234")
			return err
		}},
		{name: "domain verify", call: func() error {
			_, _, err := client.VerifyDomain(context.Background(), "domain-1", json.RawMessage(`{}`), "verify-1234")
			return err
		}},
		{name: "domain DNS", call: func() error { _, _, err := client.ListDomainDNSRecords(context.Background(), "domain-1"); return err }},
		{name: "template list", call: func() error { _, _, err := client.ListTemplates(context.Background(), nil, scope); return err }},
		{name: "template get", call: func() error { _, _, err := client.GetTemplate(context.Background(), "template-1", scope); return err }},
		{name: "template create", call: func() error {
			_, _, err := client.CreateTemplate(context.Background(), json.RawMessage(`{"name":"welcome","scope":{}}`), "template-1234", scope)
			return err
		}},
		{name: "template transition", call: func() error {
			_, _, err := client.TransitionTemplate(context.Background(), "template-1", json.RawMessage(`{"event":"publish"}`), "transition-1234", scope)
			return err
		}},
		{name: "template versions", call: func() error {
			_, _, err := client.ListTemplateVersions(context.Background(), "template-1", scope)
			return err
		}},
		{name: "template version create", call: func() error {
			_, _, err := client.CreateTemplateVersion(context.Background(), "template-1", json.RawMessage(`{"subject":"Hi","scope":{}}`), "version-1234", scope)
			return err
		}},
		{name: "template publish", call: func() error {
			_, _, err := client.PublishTemplateVersion(context.Background(), "template-1", "version-1", "publish-1234", scope)
			return err
		}},
		{name: "webhook list", call: func() error { _, _, err := client.ListWebhooks(context.Background(), nil); return err }},
		{name: "webhook create", call: func() error {
			_, _, err := client.CreateWebhook(context.Background(), json.RawMessage(`{"url":"https://example.test/hook"}`), "webhook-1234")
			return err
		}},
		{name: "webhook get", call: func() error { _, _, err := client.GetWebhook(context.Background(), "webhook-1"); return err }},
		{name: "webhook update", call: func() error {
			_, _, err := client.UpdateWebhook(context.Background(), "webhook-1", json.RawMessage(`{"enabled":false}`), "update-1234")
			return err
		}},
		{name: "webhook delete", call: func() error {
			_, _, err := client.DeleteWebhook(context.Background(), "webhook-1", "delete-1234")
			return err
		}},
		{name: "webhook rotate", call: func() error {
			_, _, err := client.RotateWebhookSecret(context.Background(), "webhook-1", json.RawMessage(`{}`), "rotate-1234")
			return err
		}},
		{name: "delivery list", call: func() error { _, _, err := client.ListWebhookDeliveries(context.Background(), nil); return err }},
		{name: "delivery get", call: func() error { _, _, err := client.GetWebhookDelivery(context.Background(), "delivery-1"); return err }},
		{name: "delivery replay", call: func() error {
			_, _, err := client.ReplayWebhookDelivery(context.Background(), "delivery-1", "replay-1234")
			return err
		}},
		{name: "logs list", call: func() error { _, _, err := client.ListLogs(context.Background(), nil); return err }},
	}
	for _, test := range calls {
		t.Run(test.name, func(t *testing.T) {
			if err := test.call(); err != nil {
				t.Fatal(err)
			}
		})
	}
	if len(requests) != len(calls) {
		t.Fatalf("requests = %d, calls = %d", len(requests), len(calls))
	}
	for _, request := range requests {
		if request.header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("authorization header missing for %#v", request)
		}
		requiresKey := strings.Contains(request.path, "/templates/") || strings.Contains(request.path, "/segments/") || strings.Contains(request.path, "/domains/domain-1/verify")
		if requiresKey && request.method != http.MethodGet {
			if request.header.Get("Idempotency-Key") == "" {
				t.Fatalf("mutation missing idempotency header for %#v", request)
			}
		}
	}
	var found bool
	for _, request := range requests {
		if request.path == "/api/v1/templates/template-1" && strings.Contains(request.query, "project_id=project-1") {
			found = request.header.Get("X-Environment") == "staging" && strings.Contains(request.query, "environment=staging") && strings.Contains(request.query, "tenant_id=tenant-1")
		}
	}
	if !found {
		t.Fatal("scoped template request did not include query and X-Environment scope")
	}
}

func TestFollowLogsPollsCursorAndDeduplicates(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var paths []string
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		paths = append(paths, request.URL.RequestURI())
		if len(paths) == 1 {
			response := httpResponse(200, `{"success":true,"data":{"items":[{"id":"log-1"}]}}`)
			response.Header.Set("X-Next-Cursor", "cursor-2")
			return response, nil
		}
		response := httpResponse(200, `{"success":true,"data":{"items":[{"id":"log-1"},{"id":"log-2"}]}}`)
		response.Header.Set("X-Next-Cursor", "cursor-3")
		return response, nil
	})}
	var events []string
	err := client.FollowLogs(ctx, url.Values{"operation": {"send"}}, 100*time.Millisecond, func(event json.RawMessage) error {
		events = append(events, string(event))
		if len(events) == 2 {
			cancel()
		}
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
	if len(events) != 2 || !strings.Contains(events[0], "log-1") || !strings.Contains(events[1], "log-2") {
		t.Fatalf("events = %v", events)
	}
	if len(paths) != 2 || !strings.Contains(paths[1], "cursor=cursor-2") || !strings.Contains(paths[1], "operation=send") {
		t.Fatalf("paths = %v", paths)
	}
}

func TestFollowLogsRequiresBoundedInterval(t *testing.T) {
	client := NewClient("https://api.example.test", "secret")
	err := client.FollowLogs(context.Background(), nil, 99*time.Millisecond, func(json.RawMessage) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "at least 100ms") {
		t.Fatalf("error = %v", err)
	}
}

func TestRequestBodyLimitAndSecretRedaction(t *testing.T) {
	client := NewClient("https://api.example.test", "secret-token")
	client.MaxRequestBodyBytes = 8
	if _, _, err := client.CreateWebhook(context.Background(), json.RawMessage(`{"too":"long"}`), "webhook-1234"); err == nil || !strings.Contains(err.Error(), "request body exceeds") {
		t.Fatalf("oversized request error = %v", err)
	}
	client.MaxRequestBodyBytes = DefaultMaxRequestBodyBytes
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return httpResponse(400, `{"success":false,"error":{"message":"token secret-token is invalid","raw_body":"provider-secret","headers":{"authorization":"Bearer provider-secret"}}}`), nil
	})}
	_, _, err := client.CreateWebhook(context.Background(), json.RawMessage(`{"url":"https://example.test"}`), "webhook-1234")
	if err == nil || strings.Contains(err.Error(), "secret-token") || !strings.Contains(err.Error(), "[REDACTED]") {
		t.Fatalf("redacted error = %v", err)
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	problem, ok := apiErr.Problem["message"].(string)
	if !ok || strings.Contains(problem, "secret-token") {
		t.Fatalf("stored problem retained secret: %#v", apiErr.Problem)
	}
	if raw, ok := apiErr.Problem["raw_body"]; !ok || raw != "[REDACTED]" {
		t.Fatalf("stored raw body was not redacted: %#v", apiErr.Problem)
	}
	if headers, ok := apiErr.Problem["headers"].(string); !ok || headers != "[REDACTED]" {
		t.Fatalf("stored headers were not redacted: %#v", apiErr.Problem)
	}
}

func TestScopedPhase1ResourcesCarryQueryAndEnvironmentHeader(t *testing.T) {
	var requests []*http.Request
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests = append(requests, request)
		return httpResponse(200, `{"success":true,"data":{"items":[]}}`), nil
	})}
	scope := Scope{ProjectID: "project-1", Environment: "preview", TenantID: "tenant-1"}
	if _, _, err := client.ListContacts(context.Background(), nil, scope); err != nil {
		t.Fatal(err)
	}
	if _, _, err := client.GetSegment(context.Background(), "segment-1", scope); err != nil {
		t.Fatal(err)
	}
	if _, _, err := client.ListBroadcasts(context.Background(), nil, scope); err != nil {
		t.Fatal(err)
	}
	if _, _, err := client.GetAutomation(context.Background(), "automation-1", scope); err != nil {
		t.Fatal(err)
	}
	if _, _, err := client.GetEmailTrace(context.Background(), "email-1", Scope{ProjectID: "project-1", Environment: "preview"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := client.ListAuditEvents(context.Background(), nil, Scope{ProjectID: "project-1", Environment: "preview"}); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 6 {
		t.Fatalf("requests = %d", len(requests))
	}
	for _, request := range requests {
		query := request.URL.Query()
		if query.Get("project_id") != "project-1" || query.Get("environment") != "preview" {
			t.Fatalf("scope query missing from %s: %s", request.URL.Path, request.URL.RawQuery)
		}
		if request.Header.Get("X-Environment") != "preview" {
			t.Fatalf("scope header missing from %s", request.URL.Path)
		}
	}
	if got := requests[0].URL.Query().Get("tenant_id"); got != "tenant-1" {
		t.Fatalf("tenant query = %q", got)
	}
	if got := requests[4].URL.Query().Get("tenant_id"); got != "" {
		t.Fatalf("optional tenant query = %q", got)
	}
}

func TestOptionalMutationsDoNotRequireIdempotency(t *testing.T) {
	var header string
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		header = request.Header.Get("Idempotency-Key")
		return httpResponse(200, `{"success":true,"data":{}}`), nil
	})}
	if _, _, err := client.CreateWebhook(context.Background(), json.RawMessage(`{"url":"https://example.test"}`), ""); err != nil {
		t.Fatal(err)
	}
	if header != "" {
		t.Fatalf("unexpected idempotency header %q", header)
	}
}

func TestOptionalKeyDoesNotEnableMutationRetry(t *testing.T) {
	calls := 0
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		if request.Header.Get("Idempotency-Key") != "optional-1234" {
			t.Fatalf("optional idempotency header = %q", request.Header.Get("Idempotency-Key"))
		}
		return httpResponse(http.StatusServiceUnavailable, `{"title":"busy"}`), nil
	})}
	_, _, err := client.CreateWebhook(context.Background(), json.RawMessage(`{"url":"https://example.test"}`), " optional-1234 ")
	if err == nil {
		t.Fatal("expected service unavailable error")
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want one non-retried mutation", calls)
	}
}

func TestExpandedPhase1RoutesAndScope(t *testing.T) {
	type observed struct {
		method string
		path   string
		query  url.Values
		header http.Header
		body   string
	}
	var requests []observed
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(request.Body)
		requests = append(requests, observed{method: request.Method, path: request.URL.EscapedPath(), query: request.URL.Query(), header: request.Header.Clone(), body: string(body)})
		return httpResponse(200, `{"success":true,"data":{}}`), nil
	})}
	scope := Scope{ProjectID: "project-1", Environment: "staging", TenantID: "tenant-1"}
	platformScope := Scope{ProjectID: "project-1", Environment: "staging"}
	headers := http.Header{"X-Provider-Signature": []string{"v1=abc"}, "X-Provider-Timestamp": []string{"1700000000"}, "X-Provider-Event-ID": []string{"evt-1"}}
	run := func(name, method, path string, scoped bool, call func() error) {
		t.Run(name, func(t *testing.T) {
			before := len(requests)
			if err := call(); err != nil {
				t.Fatal(err)
			}
			if len(requests) != before+1 {
				t.Fatalf("request count changed from %d to %d", before, len(requests))
			}
			request := requests[before]
			if request.method != method || request.path != path {
				t.Fatalf("request = %s %s, want %s %s", request.method, request.path, method, path)
			}
			if scoped {
				if request.query.Get("project_id") != "project-1" || request.query.Get("environment") != "staging" {
					t.Fatalf("scope query = %s", request.query.Encode())
				}
				if request.header.Get("X-Environment") != "staging" {
					t.Fatal("X-Environment header missing")
				}
			}
		})
	}

	run("domain DNS provision", http.MethodPost, "/api/v1/domains/domain-1/dns-records/provision", false, func() error {
		_, _, err := client.ProvisionDomainDNS(context.Background(), "domain-1", json.RawMessage(`{}`), "dns-1234")
		return err
	})
	run("template version get", http.MethodGet, "/api/v1/templates/template-1/versions/latest", true, func() error {
		_, _, err := client.GetTemplateVersion(context.Background(), "template-1", "latest", scope)
		return err
	})
	run("template version compare", http.MethodGet, "/api/v1/templates/template-1/versions/compare", true, func() error {
		_, _, err := client.CompareTemplateVersions(context.Background(), "template-1", "1", "latest", scope)
		return err
	})
	if got := requests[2].query.Get("from"); got != "1" {
		t.Fatalf("compare from = %q", got)
	}
	if got := requests[2].query.Get("to"); got != "latest" {
		t.Fatalf("compare to = %q", got)
	}
	run("template version rollback", http.MethodPost, "/api/v1/templates/template-1/versions/version-1/rollback", true, func() error {
		_, _, err := client.RollbackTemplateVersion(context.Background(), "template-1", "version-1", "rollback-1234", scope)
		return err
	})
	run("domain health list", http.MethodGet, "/api/v1/domain-health", true, func() error {
		_, _, err := client.ListDomainHealth(context.Background(), nil, platformScope)
		return err
	})
	run("domain health verify", http.MethodPost, "/api/v1/domain-health", true, func() error {
		_, _, err := client.VerifyDomainHealth(context.Background(), json.RawMessage(`{"domain_id":"domain-1"}`), "health-1234", platformScope)
		return err
	})
	run("domain health get", http.MethodGet, "/api/v1/domain-health/domain-1", true, func() error {
		_, _, err := client.GetDomainHealth(context.Background(), "domain-1", platformScope)
		return err
	})
	run("API key list", http.MethodGet, "/api/v1/api-keys", false, func() error {
		_, _, err := client.ListAPIKeys(context.Background())
		return err
	})
	run("API key create", http.MethodPost, "/api/v1/api-keys", false, func() error {
		_, _, err := client.CreateAPIKey(context.Background(), json.RawMessage(`{"name":"cli"}`))
		return err
	})
	run("API key update", http.MethodPatch, "/api/v1/api-keys/key-1", false, func() error {
		_, _, err := client.UpdateAPIKey(context.Background(), "key-1", json.RawMessage(`{"enabled":false}`))
		return err
	})
	run("API key revoke", http.MethodDelete, "/api/v1/api-keys/key-1", false, func() error {
		_, _, err := client.RevokeAPIKey(context.Background(), "key-1")
		return err
	})
	run("inbound", http.MethodPost, "/api/v1/inbound/ses", false, func() error {
		_, _, err := client.AcceptInboundWithHeaders(context.Background(), "ses", json.RawMessage(`{"message_id":"m-1"}`), headers)
		return err
	})
	if got := requests[11].header.Get("X-Provider-Event-ID"); got != "evt-1" {
		t.Fatalf("provider event ID = %q", got)
	}
	run("provider event", http.MethodPost, "/api/v1/provider-events/ses", false, func() error {
		_, _, err := client.AcceptProviderEventWithHeaders(context.Background(), "ses", json.RawMessage(`{"event_id":"e-1"}`), headers)
		return err
	})
	run("received email list", http.MethodGet, "/api/v1/received-emails", false, func() error {
		_, _, err := client.ListReceivedEmails(context.Background(), url.Values{"cursor": []string{"next"}})
		return err
	})
	run("preference get", http.MethodGet, "/api/v1/preferences/token-1", false, func() error {
		_, _, err := client.GetPreferenceCenter(context.Background(), "token-1")
		return err
	})
	run("preference update", http.MethodPost, "/api/v1/preferences/token-1", false, func() error {
		_, _, err := client.UpdatePreferenceCenter(context.Background(), "token-1", json.RawMessage(`{"subscriptions":[]}`))
		return err
	})
	run("suppression list", http.MethodGet, "/api/v1/suppressions", false, func() error {
		_, _, err := client.ListSuppressions(context.Background(), url.Values{"active": []string{"true"}})
		return err
	})
	run("suppression create", http.MethodPost, "/api/v1/suppressions", false, func() error {
		_, _, err := client.CreateSuppression(context.Background(), json.RawMessage(`{"address":"user@example.test"}`))
		return err
	})
	run("suppression delete", http.MethodDelete, "/api/v1/suppressions/suppression-1", false, func() error {
		_, _, err := client.DeleteSuppression(context.Background(), "suppression-1")
		return err
	})
	run("contact create", http.MethodPost, "/api/v1/contacts", true, func() error {
		_, _, err := client.CreateContact(context.Background(), json.RawMessage(`{"scope":{}}`), "contact-1234", scope)
		return err
	})
	run("contact transition", http.MethodPatch, "/api/v1/contacts/contact-1", true, func() error {
		_, _, err := client.TransitionContact(context.Background(), "contact-1", json.RawMessage(`{"event":"subscribe"}`), "contact-transition-1", scope)
		return err
	})
	run("segment create", http.MethodPost, "/api/v1/segments", true, func() error {
		_, _, err := client.CreateSegment(context.Background(), json.RawMessage(`{"scope":{}}`), "segment-1234", scope)
		return err
	})
	run("segment archive", http.MethodPatch, "/api/v1/segments/segment-1", true, func() error {
		_, _, err := client.ArchiveSegment(context.Background(), "segment-1", "segment-archive-1", scope)
		return err
	})
	run("segment evaluate", http.MethodPost, "/api/v1/segments/segment-1/evaluate", true, func() error {
		_, _, err := client.EvaluateSegment(context.Background(), "segment-1", "", scope)
		return err
	})
	run("broadcast create", http.MethodPost, "/api/v1/broadcasts", true, func() error {
		_, _, err := client.CreateBroadcast(context.Background(), json.RawMessage(`{"scope":{}}`), "broadcast-1234", scope)
		return err
	})
	run("broadcast schedule", http.MethodPost, "/api/v1/broadcasts/broadcast-1/schedule", true, func() error {
		_, _, err := client.ScheduleBroadcast(context.Background(), "broadcast-1", json.RawMessage(`{"scheduled_at":"2030-01-01T00:00:00Z"}`), "broadcast-schedule-1", scope)
		return err
	})
	run("broadcast cancel", http.MethodPost, "/api/v1/broadcasts/broadcast-1/cancel", true, func() error {
		_, _, err := client.CancelBroadcast(context.Background(), "broadcast-1", "broadcast-cancel-1", scope)
		return err
	})
	run("automation create", http.MethodPost, "/api/v1/automations", true, func() error {
		_, _, err := client.CreateAutomation(context.Background(), json.RawMessage(`{"scope":{}}`), "automation-1234", scope)
		return err
	})
	run("automation activate", http.MethodPost, "/api/v1/automations/automation-1/activate", true, func() error {
		_, _, err := client.TransitionAutomation(context.Background(), "automation-1", "activate", "automation-activate-1", scope)
		return err
	})
	run("automation run", http.MethodPost, "/api/v1/automations/automation-1/runs", true, func() error {
		_, _, err := client.StartAutomationRun(context.Background(), "automation-1", json.RawMessage(`{"contact_id":"contact-1"}`), "automation-run-1", scope)
		return err
	})
	run("automation event", http.MethodPost, "/api/v1/automation-events", true, func() error {
		_, _, err := client.IngestAutomationEvent(context.Background(), json.RawMessage(`{"event_type":"contact.updated"}`), "automation-event-1", scope)
		return err
	})
	run("audit append", http.MethodPost, "/api/v1/audit-events", true, func() error {
		_, _, err := client.AppendAuditEvent(context.Background(), json.RawMessage(`{"action":"test"}`), "audit-1234", platformScope)
		return err
	})
	run("audit hold create", http.MethodPost, "/api/v1/audit-events/holds", true, func() error {
		_, _, err := client.CreateAuditHold(context.Background(), json.RawMessage(`{"name":"hold","reason":"test"}`), "hold-1234", platformScope)
		return err
	})
	run("audit hold release", http.MethodDelete, "/api/v1/audit-events/holds/hold-1", true, func() error {
		_, _, err := client.ReleaseAuditHold(context.Background(), "hold-1", "hold-release-1", platformScope)
		return err
	})
	run("email replay", http.MethodPost, "/api/v1/emails/email-1/replay", true, func() error {
		_, _, err := client.ReplayEmail(context.Background(), "email-1", json.RawMessage(`{"reason":"test"}`), "email-replay-1", platformScope)
		return err
	})
	run("delivery policy list", http.MethodGet, "/api/v1/delivery-policies", true, func() error {
		_, _, err := client.ListDeliveryPolicies(context.Background(), platformScope)
		return err
	})
	run("delivery policy create", http.MethodPost, "/api/v1/delivery-policies", true, func() error {
		_, _, err := client.CreateDeliveryPolicy(context.Background(), json.RawMessage(`{"scope":{}}`), "policy-1234", platformScope)
		return err
	})
	run("delivery policy update", http.MethodPatch, "/api/v1/delivery-policies/policy-1", true, func() error {
		_, _, err := client.UpdateDeliveryPolicy(context.Background(), "policy-1", json.RawMessage(`{"scope":{}}`), "policy-update-1", platformScope)
		return err
	})
	run("delivery route list", http.MethodGet, "/api/v1/delivery-routes", true, func() error {
		_, _, err := client.ListDeliveryRoutes(context.Background(), platformScope)
		return err
	})
	run("delivery route create", http.MethodPost, "/api/v1/delivery-routes", true, func() error {
		_, _, err := client.CreateDeliveryRoute(context.Background(), json.RawMessage(`{"scope":{}}`), "route-1234", platformScope)
		return err
	})
	run("SMTP list", http.MethodGet, "/api/v1/smtp-credentials", true, func() error {
		_, _, err := client.ListSMTPCredentials(context.Background(), nil, platformScope)
		return err
	})
	run("SMTP create", http.MethodPost, "/api/v1/smtp-credentials", true, func() error {
		_, _, err := client.CreateSMTPCredential(context.Background(), json.RawMessage(`{"name":"cli"}`), "smtp-1234", platformScope)
		return err
	})
	run("SMTP get", http.MethodGet, "/api/v1/smtp-credentials/smtp-1", true, func() error {
		_, _, err := client.GetSMTPCredential(context.Background(), "smtp-1", platformScope)
		return err
	})
	run("SMTP revoke", http.MethodDelete, "/api/v1/smtp-credentials/smtp-1", true, func() error {
		_, _, err := client.RevokeSMTPCredential(context.Background(), "smtp-1", "smtp-revoke-1", platformScope)
		return err
	})

	if len(requests) == 0 {
		t.Fatal("no requests observed")
	}
}

func TestRoutingRoutesUseDocumentedPathsAndCommandKeys(t *testing.T) {
	type expectedRequest struct {
		method string
		path   string
		key    string
		call   func() error
	}
	var requests []*http.Request
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests = append(requests, request)
		return httpResponse(http.StatusOK, `{"success":true,"data":{}}`), nil
	})}
	routingBody := json.RawMessage(`{"allowedRegions":["eu"],"requiredResidency":"eu"}`)
	poolBody := json.RawMessage(`{"provider":"ses","region":"eu","residency":"eu","warmup":{"startAt":"2030-01-01T00:00:00Z","initialDailyCapacity":10,"maxDailyCapacity":100},"ips":[{"address":"192.0.2.1"}]}`)
	commandBody := json.RawMessage(`{"action":"start_warmup"}`)
	expected := []expectedRequest{
		{http.MethodGet, "/api/v1/routing/policy", "", func() error { _, _, err := client.GetRoutingPolicy(context.Background()); return err }},
		{http.MethodPut, "/api/v1/routing/policy", "policy-1234", func() error {
			_, _, err := client.PutRoutingPolicy(context.Background(), routingBody, "policy-1234")
			return err
		}},
		{http.MethodGet, "/api/v1/routing/capabilities", "", func() error { _, _, err := client.ListRoutingCapabilities(context.Background()); return err }},
		{http.MethodGet, "/api/v1/routing/pools", "", func() error { _, _, err := client.ListRoutingPools(context.Background()); return err }},
		{http.MethodPost, "/api/v1/routing/pools", "pool-1234", func() error {
			_, _, err := client.CreateRoutingPool(context.Background(), poolBody, "pool-1234")
			return err
		}},
		{http.MethodGet, "/api/v1/routing/pools/pool-1", "", func() error { _, _, err := client.GetRoutingPool(context.Background(), "pool-1"); return err }},
		{http.MethodPost, "/api/v1/routing/pools/pool-1/command", "warmup-1234", func() error {
			_, _, err := client.ApplyRoutingPoolCommand(context.Background(), "pool-1", commandBody, "warmup-1234")
			return err
		}},
		{http.MethodPost, "/api/v1/routing/pools/pool-1/ips/ip-1/command", "ip-12345", func() error {
			_, _, err := client.ApplyRoutingIPCommand(context.Background(), "pool-1", "ip-1", json.RawMessage(`{"action":"activate"}`), "ip-12345")
			return err
		}},
		{http.MethodGet, "/api/v1/routing/audit", "", func() error {
			_, _, err := client.ListRoutingAudit(context.Background(), url.Values{"entityType": []string{"dedicated_pool"}, "entityId": []string{"pool-1"}})
			return err
		}},
	}
	for index, item := range expected {
		if err := item.call(); err != nil {
			t.Fatalf("call %d: %v", index, err)
		}
		request := requests[index]
		if request.Method != item.method || request.URL.EscapedPath() != item.path {
			t.Fatalf("request %d = %s %s, want %s %s", index, request.Method, request.URL.EscapedPath(), item.method, item.path)
		}
		if request.Header.Get("Idempotency-Key") != item.key {
			t.Fatalf("request %d idempotency key = %q, want %q", index, request.Header.Get("Idempotency-Key"), item.key)
		}
	}
	if got := requests[len(requests)-1].URL.Query(); got.Get("entityType") != "dedicated_pool" || got.Get("entityId") != "pool-1" {
		t.Fatalf("routing audit query = %s", got.Encode())
	}
}

func TestEmptySuccessBodyIsSafeJSON(t *testing.T) {
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return httpResponse(http.StatusNoContent, ""), nil
	})}
	result, _, err := client.RevokeAPIKey(context.Background(), "key-1")
	if err != nil {
		t.Fatal(err)
	}
	if string(result) != "{}" {
		t.Fatalf("result = %q", result)
	}
}
