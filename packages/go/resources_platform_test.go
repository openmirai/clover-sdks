package clover

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func platformScopeTest() PlatformScope {
	return PlatformScope{AccountID: "acct/one", EnvironmentID: "env/blue"}
}

func TestPlatformMessageUsesAccountEnvironmentPathAndKey(t *testing.T) {
	fake := &fakeDoer{responses: []*http.Response{response(http.StatusAccepted, `{"success":true,"requestId":"req_platform","data":{"id":"msg_1","status":"queued"}}`)}}
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = fake
	client.MaxRetries = 0
	accepted, meta, err := client.Platform.Messages.Send(context.Background(), platformScopeTest(), SendMessageRequest{
		From:    PlatformAddress{Address: "sender@example.com"},
		To:      []PlatformAddress{{Address: "recipient@example.com"}},
		Subject: "hello",
		Text:    "body",
	}, "message-1234")
	if err != nil {
		t.Fatal(err)
	}
	if accepted.ID != "msg_1" || meta.RequestID != "req_platform" {
		t.Fatalf("accepted=%#v meta=%#v", accepted, meta)
	}
	request := fake.requests[0]
	if request.URL.EscapedPath() != "/api/v1/platform/accounts/acct%2Fone/environments/env%2Fblue/messages" {
		t.Fatalf("path=%q", request.URL.EscapedPath())
	}
	if request.URL.Query().Get("tenant_id") != "" || request.URL.Query().Get("project_id") != "" {
		t.Fatalf("legacy scope leaked into query: %v", request.URL.Query())
	}
	if request.Header.Get("Idempotency-Key") != "message-1234" || !strings.HasPrefix(request.Header.Get("X-Request-ID"), "req_") {
		t.Fatalf("headers=%v", request.Header)
	}
	var body SendMessageRequest
	if err := json.Unmarshal(fake.bodies[0], &body); err != nil {
		t.Fatal(err)
	}
	if body.ScheduledAt != "" || body.Subject != "hello" || len(body.To) != 1 {
		t.Fatalf("body=%#v", body)
	}
}

func TestPlatformTemplateVersionEscapesEveryPathSegment(t *testing.T) {
	fake := &fakeDoer{responses: []*http.Response{response(http.StatusOK, `{"success":true,"data":{"id":"version_1","templateId":"template/one","number":1}}`)}}
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = fake
	client.MaxRetries = 0
	version, _, err := client.Platform.Templates.GetVersion(context.Background(), platformScopeTest(), "template/one", "version/latest")
	if err != nil {
		t.Fatal(err)
	}
	if version.ID != "version_1" {
		t.Fatalf("version=%#v", version)
	}
	if got := fake.requests[0].URL.EscapedPath(); got != "/api/v1/platform/accounts/acct%2Fone/environments/env%2Fblue/templates/template%2Fone/versions/version%2Flatest" {
		t.Fatalf("path=%q", got)
	}
}

func TestPlatformInboundCallbackAddsProviderHeaders(t *testing.T) {
	fake := &fakeDoer{responses: []*http.Response{response(http.StatusAccepted, `{"success":true,"request_id":"req_inbound","data":{"id":"in_1","created":true}}`)}}
	client := NewPublicClient("https://api.example.test")
	client.HTTPClient = fake
	client.MaxRetries = 0
	headers := make(http.Header)
	headers.Set("X-Provider-Signature", "sig")
	headers.Set("X-Provider-Timestamp", "now")
	headers.Set("X-Provider-Event-ID", "evt")
	result, meta, err := client.Platform.Inbound.AcceptProviderMessage(context.Background(), "provider/one", JSON{"event": "received"}, headers)
	if err != nil {
		t.Fatal(err)
	}
	if result["id"] != "in_1" || meta.RequestID != "req_inbound" {
		t.Fatalf("result=%#v meta=%#v", result, meta)
	}
	request := fake.requests[0]
	if request.URL.EscapedPath() != "/api/v1/inbound/provider%2Fone" || request.Header.Get("X-Provider-Signature") != "sig" {
		t.Fatalf("request=%s headers=%v", request.URL, request.Header)
	}
	if request.Header.Get("Authorization") != "" {
		t.Fatal("public provider callback unexpectedly sent bearer authorization")
	}
}

func TestPlatformScopeAndRequestBodyBoundsFailBeforeTransport(t *testing.T) {
	fake := &fakeDoer{responses: []*http.Response{response(http.StatusAccepted, `{}`)}}
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = fake
	client.MaxRetries = 0
	_, _, err := client.Platform.Messages.Send(context.Background(), PlatformScope{AccountID: "acct", EnvironmentID: ""}, SendMessageRequest{}, "message-1234")
	if !errors.Is(err, ErrMissingPlatformScope) || len(fake.requests) != 0 {
		t.Fatalf("scope error=%v requests=%d", err, len(fake.requests))
	}
	client.MaxRequestBodyBytes = 8
	_, _, err = client.Platform.Messages.Send(context.Background(), PlatformScope{AccountID: "acct", EnvironmentID: "env"}, SendMessageRequest{Subject: strings.Repeat("x", 32)}, "message-1234")
	if !errors.Is(err, ErrRequestBodyTooLarge) || len(fake.requests) != 0 {
		t.Fatalf("body error=%v requests=%d", err, len(fake.requests))
	}
}

func TestPlatformRequiredMutationsRejectMissingIdempotencyKey(t *testing.T) {
	tests := []struct {
		name string
		call func(*Client) error
	}{
		{
			name: "webhook create",
			call: func(client *Client) error {
				_, _, err := client.Platform.Webhooks.Create(context.Background(), platformScopeTest(), CreateScopedWebhookRequest{})
				return err
			},
		},
		{
			name: "preference topic create",
			call: func(client *Client) error {
				_, _, err := client.Platform.Preferences.CreateTopic(context.Background(), platformScopeTest(), CreatePlatformPreferenceTopicRequest{})
				return err
			},
		},
		{
			name: "suppression create",
			call: func(client *Client) error {
				_, _, err := client.Platform.Suppressions.Create(context.Background(), platformScopeTest(), CreatePlatformSuppressionRequest{})
				return err
			},
		},
		{
			name: "contact create",
			call: func(client *Client) error {
				_, _, err := client.Platform.Contacts.Create(context.Background(), platformScopeTest(), CreatePlatformContactRequest{})
				return err
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeDoer{responses: []*http.Response{response(http.StatusCreated, `{"success":true,"data":{}}`)}}
			client := NewClient("https://api.example.test", "secret")
			client.HTTPClient = fake
			client.MaxRetries = 0
			if err := test.call(client); !errors.Is(err, ErrInvalidIdempotencyKey) || len(fake.requests) != 0 {
				t.Fatalf("error=%v requests=%d", err, len(fake.requests))
			}
		})
	}
}

func TestPlatformRequiredMutationForwardsIdempotencyKey(t *testing.T) {
	fake := &fakeDoer{responses: []*http.Response{response(http.StatusCreated, `{"success":true,"data":{}}`)}}
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = fake
	client.MaxRetries = 0
	if _, _, err := client.Platform.Contacts.Create(context.Background(), platformScopeTest(), CreatePlatformContactRequest{}, "contact-1234"); err != nil {
		t.Fatal(err)
	}
	if got := fake.requests[0].Header.Get("Idempotency-Key"); got != "contact-1234" {
		t.Fatalf("Idempotency-Key=%q", got)
	}
}

func TestCursorPageAcceptsBothCursorSpellings(t *testing.T) {
	var snake CursorPage[MessageTimelineEntry]
	if err := json.Unmarshal([]byte(`{"items":[],"next_cursor":"snake"}`), &snake); err != nil || snake.NextCursor != "snake" {
		t.Fatalf("snake=%#v err=%v", snake, err)
	}
	var camel CursorPage[UsageFact]
	if err := json.Unmarshal([]byte(`{"items":[],"nextCursor":"camel"}`), &camel); err != nil || camel.NextCursor != "camel" {
		t.Fatalf("camel=%#v err=%v", camel, err)
	}
}
