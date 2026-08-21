package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFlagArgsFirstPreservesPositionalArgumentsAndValues(t *testing.T) {
	got := flagArgsFirst([]string{"domain-1", "-json", `{"name":"example.test"}`, "-idempotency-key", "domain-1234"})
	want := []string{"-json", `{"name":"example.test"}`, "-idempotency-key", "domain-1234", "domain-1"}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("flagArgsFirst = %#v, want %#v", got, want)
	}
}

func TestHelpDoesNotRequireConnectionConfiguration(t *testing.T) {
	t.Setenv("CLOVER_BASE_URL", "")
	t.Setenv("CLOVER_API_KEY", "")
	if err := runContext(context.Background(), []string{"help"}); err != nil {
		t.Fatal(err)
	}
}

func TestDomainConfigureCommandUsesSwaggerRequest(t *testing.T) {
	var request *http.Request
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, incoming *http.Request) {
		request = incoming
		bytes, _ := io.ReadAll(incoming.Body)
		body = string(bytes)
		response.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(response, `{"success":true,"data":{"id":"domain-1"}}`)
	}))
	defer server.Close()
	client := NewClient(server.URL, "secret")
	if err := runDomains(context.Background(), client, []string{"configure", "domain-1", "-json", `{"sendingEnabled":true}`, "-idempotency-key", "domain-config-1"}); err != nil {
		t.Fatal(err)
	}
	if request.Method != http.MethodPatch || request.URL.EscapedPath() != "/api/v1/domains/domain-1" {
		t.Fatalf("request = %s %s", request.Method, request.URL.RequestURI())
	}
	if request.Header.Get("Idempotency-Key") != "domain-config-1" {
		t.Fatal("idempotency header missing")
	}
	if body != `{"sendingEnabled":true}` {
		t.Fatalf("body = %s", body)
	}
}

func TestTemplateListCommandSendsScopeQueryAndHeader(t *testing.T) {
	var request *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, incoming *http.Request) {
		request = incoming
		response.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(response, `{"success":true,"data":{"items":[]}}`)
	}))
	defer server.Close()
	client := NewClient(server.URL, "secret")
	if err := runTemplates(context.Background(), client, []string{"list", "-project-id", "project-1", "-environment", "production", "-tenant-id", "tenant-1"}); err != nil {
		t.Fatal(err)
	}
	query := request.URL.Query()
	if query.Get("project_id") != "project-1" || query.Get("environment") != "production" || query.Get("tenant_id") != "tenant-1" {
		t.Fatalf("scope query = %s", request.URL.RawQuery)
	}
	if request.Header.Get("X-Environment") != "production" {
		t.Fatal("X-Environment header missing")
	}
}

func TestDomainListUsesLiveSwaggerFilterNames(t *testing.T) {
	var request *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, incoming *http.Request) {
		request = incoming
		response.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(response, `{"success":true,"data":{}}`)
	}))
	defer server.Close()
	client := NewClient(server.URL, "secret")
	if err := runDomains(context.Background(), client, []string{"list", "-sending-enabled", "true", "-receiving-enabled", "false"}); err != nil {
		t.Fatal(err)
	}
	query := request.URL.Query()
	if query.Get("sendingEnabled") != "true" || query.Get("receivingEnabled") != "false" {
		t.Fatalf("domain query = %s", request.URL.RawQuery)
	}
	if query.Get("sending_enabled") != "" || query.Get("receiving_enabled") != "" {
		t.Fatalf("stale snake_case filters present: %s", request.URL.RawQuery)
	}
}

func TestProviderCallbackCommandSendsVerificationHeaders(t *testing.T) {
	var request *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, incoming *http.Request) {
		request = incoming
		response.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(response, `{"success":true,"data":{}}`)
	}))
	defer server.Close()
	client := NewClient(server.URL, "secret")
	if err := runProviderEvents(context.Background(), client, []string{
		"ses", "-json", `{"event_id":"event-1"}`,
		"-provider-signature", "v1=signature",
		"-provider-timestamp", "1700000000",
		"-provider-event-id", "event-1",
	}); err != nil {
		t.Fatal(err)
	}
	if request.URL.EscapedPath() != "/api/v1/provider-events/ses" {
		t.Fatalf("path = %s", request.URL.EscapedPath())
	}
	if request.Header.Get("X-Provider-Signature") != "v1=signature" || request.Header.Get("X-Provider-Timestamp") != "1700000000" || request.Header.Get("X-Provider-Event-ID") != "event-1" {
		t.Fatalf("provider headers = %#v", request.Header)
	}
}

func TestProviderCallbackCommandRejectsPartialVerificationHeaders(t *testing.T) {
	client := NewClient("https://api.example.test", "secret")
	err := runInbound(context.Background(), client, []string{"ses", "-json", `{}`, "-provider-event-id", "event-1"})
	if err == nil || !strings.Contains(err.Error(), "must be supplied together") {
		t.Fatalf("error = %v", err)
	}
}

func TestScopedMutationBodyOmitsOptionalTenant(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, incoming *http.Request) {
		bytes, _ := io.ReadAll(incoming.Body)
		body = string(bytes)
		response.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(response, `{"success":true,"data":{}}`)
	}))
	defer server.Close()
	client := NewClient(server.URL, "secret")
	if err := runDeliveryPolicies(context.Background(), client, []string{
		"create", "-json", `{"name":"default"}`,
		"-organization-id", "organization-1", "-project-id", "project-1", "-environment", "staging",
		"-idempotency-key", "policy-1234",
	}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(body, "tenant_id") || !strings.Contains(body, `"organization_id":"organization-1"`) || !strings.Contains(body, `"project_id":"project-1"`) || !strings.Contains(body, `"environment":"staging"`) {
		t.Fatalf("scope body = %s", body)
	}
}

func TestMergeBodyScopeRejectsJSONNull(t *testing.T) {
	if _, err := mergeBodyScope([]byte("null"), Scope{ProjectID: "project-1", Environment: "staging"}); err == nil {
		t.Fatal("expected JSON null to be rejected")
	}
}

func TestRoutingWarmupCommandUsesDocumentedLifecycleAction(t *testing.T) {
	var request *http.Request
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, incoming *http.Request) {
		request = incoming
		bytes, _ := io.ReadAll(incoming.Body)
		body = string(bytes)
		response.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(response, `{"success":true,"data":{}}`)
	}))
	defer server.Close()
	client := NewClient(server.URL, "secret")
	if err := runRouting(context.Background(), client, []string{"pools", "warmup", "pool-1", "-idempotency-key", "warmup-1234"}); err != nil {
		t.Fatal(err)
	}
	if request.Method != http.MethodPost || request.URL.EscapedPath() != "/api/v1/routing/pools/pool-1/command" {
		t.Fatalf("request = %s %s", request.Method, request.URL.RequestURI())
	}
	if request.Header.Get("Idempotency-Key") != "warmup-1234" || body != `{"action":"start_warmup"}` {
		t.Fatalf("headers/body = %q / %s", request.Header.Get("Idempotency-Key"), body)
	}
}

func TestAutomationUpdateCommandUsesScopedPatch(t *testing.T) {
	var request *http.Request
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, incoming *http.Request) {
		request = incoming
		bytes, _ := io.ReadAll(incoming.Body)
		body = string(bytes)
		response.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(response, `{"success":true,"data":{"id":"automation-1"}}`)
	}))
	defer server.Close()
	client := NewClient(server.URL, "secret")
	if err := runAutomations(context.Background(), client, []string{
		"update", "automation/1", "-json", `{"name":"Updated flow","description":"Description","trigger":{"kind":"manual"},"steps":[{"id":"exit","position":0,"action":{"kind":"exit"}}]}`,
		"-organization-id", "organization-1", "-project-id", "project-1", "-environment", "staging", "-tenant-id", "tenant-1", "-idempotency-key", "automation-update-1",
	}); err != nil {
		t.Fatal(err)
	}
	if request.Method != http.MethodPatch || request.URL.EscapedPath() != "/api/v1/automations/automation%2F1" {
		t.Fatalf("request = %s %s", request.Method, request.URL.RequestURI())
	}
	query := request.URL.Query()
	if query.Get("project_id") != "project-1" || query.Get("environment") != "staging" || query.Get("tenant_id") != "tenant-1" {
		t.Fatalf("scope query = %s", request.URL.RawQuery)
	}
	if request.Header.Get("X-Environment") != "staging" || request.Header.Get("Idempotency-Key") != "automation-update-1" {
		t.Fatalf("headers = %#v", request.Header)
	}
	if !strings.Contains(body, `"organization_id":"organization-1"`) || !strings.Contains(body, `"name":"Updated flow"`) {
		t.Fatalf("body = %s", body)
	}
}
