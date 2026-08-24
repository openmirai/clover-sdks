package main

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestPlatformMessagesCommandUsesCleanScopePath(t *testing.T) {
	var request *http.Request
	var body string
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(incoming *http.Request) (*http.Response, error) {
		request = incoming
		bytes, _ := io.ReadAll(incoming.Body)
		body = string(bytes)
		return httpResponse(http.StatusAccepted, `{"success":true,"data":{"id":"message-1"}}`), nil
	})}
	if err := runPlatformMessages(context.Background(), client, []string{
		"send", "-account-id", "account-1", "-environment-id", "environment-1",
		"-json", `{"subject":"hello"}`, "-idempotency-key", "message-1234",
	}); err != nil {
		t.Fatal(err)
	}
	if request.Method != http.MethodPost || request.URL.EscapedPath() != "/api/v1/platform/accounts/account-1/environments/environment-1/messages" {
		t.Fatalf("request = %s %s", request.Method, request.URL.EscapedPath())
	}
	if request.URL.RawQuery != "" || strings.Contains(request.URL.RequestURI(), "project_id") {
		t.Fatalf("legacy scope leaked into request: %s", request.URL.RequestURI())
	}
	if request.Header.Get("Idempotency-Key") != "message-1234" || body != `{"subject":"hello"}` {
		t.Fatalf("headers/body = %q / %s", request.Header.Get("Idempotency-Key"), body)
	}
}

func TestPlatformUsageExportCommandUsesFactsRoute(t *testing.T) {
	var request *http.Request
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(incoming *http.Request) (*http.Response, error) {
		request = incoming
		return httpResponse(http.StatusOK, `{"success":true,"data":{"items":[]}}`), nil
	})}
	if err := runPlatformUsage(context.Background(), client, []string{
		"export", "-account-id", "account-1", "-environment-id", "environment-1", "-from", "2030-01-01T00:00:00Z", "-to", "2030-01-02T00:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}
	if request.Method != http.MethodGet || request.URL.EscapedPath() != "/api/v1/platform/accounts/account-1/environments/environment-1/usage/facts" {
		t.Fatalf("request = %s %s", request.Method, request.URL.EscapedPath())
	}
	query := request.URL.Query()
	if query.Get("from") == "" || query.Get("to") == "" || query.Get("project_id") != "" || query.Get("tenant_id") != "" {
		t.Fatalf("query = %s", request.URL.RawQuery)
	}
}

func TestPlatformSMTPCredentialCommandEscapesCredentialID(t *testing.T) {
	var request *http.Request
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(incoming *http.Request) (*http.Response, error) {
		request = incoming
		return httpResponse(http.StatusOK, `{"success":true,"data":{}}`), nil
	})}
	if err := runPlatformSMTP(context.Background(), client, []string{
		"credentials", "revoke", "credential/1", "-account-id", "account-1", "-environment-id", "environment-1", "-idempotency-key", "revoke-1234",
	}); err != nil {
		t.Fatal(err)
	}
	if request.Method != http.MethodPost || request.URL.EscapedPath() != "/api/v1/platform/accounts/account-1/environments/environment-1/smtp-credentials/credential%2F1/revoke" {
		t.Fatalf("request = %s %s", request.Method, request.URL.EscapedPath())
	}
}

func TestPlatformRequiredMutationCommandsForwardIdempotencyKey(t *testing.T) {
	tests := []struct {
		name string
		args []string
		run  func(context.Context, *Client, []string) error
	}{
		{
			name: "webhook create",
			args: []string{"create", "-account-id", "account-1", "-environment-id", "environment-1", "-json", `{}`},
			run:  runPlatformWebhooks,
		},
		{
			name: "preference topic create",
			args: []string{"topics", "create", "-account-id", "account-1", "-environment-id", "environment-1", "-json", `{}`},
			run:  runPlatformPreferences,
		},
		{
			name: "suppression create",
			args: []string{"create", "-account-id", "account-1", "-environment-id", "environment-1", "-json", `{}`},
			run:  runPlatformSuppressions,
		},
		{
			name: "contact create",
			args: []string{"create", "-account-id", "account-1", "-environment-id", "environment-1", "-json", `{}`},
			run:  runPlatformContacts,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var requests []*http.Request
			newClient := func() *Client {
				client := NewClient("https://api.example.test", "secret")
				client.HTTPClient = &http.Client{Transport: roundTripFunc(func(incoming *http.Request) (*http.Response, error) {
					requests = append(requests, incoming)
					return httpResponse(http.StatusCreated, `{"success":true,"data":{}}`), nil
				})}
				return client
			}
			if err := test.run(context.Background(), newClient(), test.args); err == nil || len(requests) != 0 {
				t.Fatalf("missing key error=%v requests=%d", err, len(requests))
			}
			requests = nil
			args := append(append([]string(nil), test.args...), "-idempotency-key", "mutation-1234")
			if err := test.run(context.Background(), newClient(), args); err != nil {
				t.Fatal(err)
			}
			if len(requests) != 1 || requests[0].Header.Get("Idempotency-Key") != "mutation-1234" {
				t.Fatalf("requests=%d key=%q", len(requests), requests[0].Header.Get("Idempotency-Key"))
			}
		})
	}
}
