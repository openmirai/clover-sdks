package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestPlatformSendUsesAccountEnvironmentRoute(t *testing.T) {
	var request *http.Request
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(incoming *http.Request) (*http.Response, error) {
		request = incoming
		return httpResponse(http.StatusAccepted, `{"success":true,"data":{"id":"message-1"}}`), nil
	})}
	result, _, err := client.PlatformSend(context.Background(), PlatformScope{AccountID: "account/1", EnvironmentID: "environment/1"}, json.RawMessage(`{"subject":"hello"}`), "send-1234")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result), `"id":"message-1"`) {
		t.Fatalf("result = %s", result)
	}
	if request.URL.EscapedPath() != "/api/v1/platform/accounts/account%2F1/environments/environment%2F1/messages" {
		t.Fatalf("path = %s", request.URL.EscapedPath())
	}
	if request.URL.RawQuery != "" {
		t.Fatalf("unexpected legacy scope query: %s", request.URL.RawQuery)
	}
	if request.Header.Get("Idempotency-Key") != "send-1234" {
		t.Fatalf("idempotency key = %q", request.Header.Get("Idempotency-Key"))
	}
}

func TestPlatformTemplateAndWebhookRoutesDoNotUseLegacyScope(t *testing.T) {
	var requests []*http.Request
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(incoming *http.Request) (*http.Response, error) {
		requests = append(requests, incoming)
		return httpResponse(http.StatusOK, `{"success":true,"data":{"items":[]}}`), nil
	})}
	scope := PlatformScope{AccountID: "account-1", EnvironmentID: "environment-1"}
	if _, _, err := client.PlatformListTemplates(context.Background(), scope, nil); err != nil {
		t.Fatal(err)
	}
	if _, _, err := client.PlatformReplayWebhookDelivery(context.Background(), scope, "delivery/1"); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 2 {
		t.Fatalf("requests = %d", len(requests))
	}
	if requests[0].URL.EscapedPath() != "/api/v1/platform/accounts/account-1/environments/environment-1/templates" {
		t.Fatalf("template path = %s", requests[0].URL.EscapedPath())
	}
	if requests[1].URL.EscapedPath() != "/api/v1/platform/accounts/account-1/environments/environment-1/webhook-deliveries/delivery%2F1/replay" {
		t.Fatalf("replay path = %s", requests[1].URL.EscapedPath())
	}
	for _, request := range requests {
		if request.URL.Query().Get("project_id") != "" || request.URL.Query().Get("tenant_id") != "" {
			t.Fatalf("legacy scope query present: %s", request.URL.RawQuery)
		}
	}
}

func TestPlatformScopeRejectsLegacyMissingIDs(t *testing.T) {
	client := NewClient("https://api.example.test", "secret")
	if _, _, err := client.PlatformListMessages(context.Background(), PlatformScope{EnvironmentID: "environment-1"}, nil); err == nil || !strings.Contains(err.Error(), "account") {
		t.Fatalf("error = %v", err)
	}
	if _, _, err := client.PlatformListMessages(context.Background(), PlatformScope{AccountID: "account-1"}, nil); err == nil || !strings.Contains(err.Error(), "environment") {
		t.Fatalf("error = %v", err)
	}
}
