package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
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

func TestStreamEventsOffline(t *testing.T) {
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("data: {\"id\":\"e1\"}\n\ndata: {\"id\":\"e2\"}\n\n"))}, nil
	})}
	var events []string
	if err := client.StreamEvents(context.Background(), "/v1/events/stream", func(event json.RawMessage) error { events = append(events, string(event)); return nil }); err != nil {
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

func TestSendBatchStripsScheduledAt(t *testing.T) {
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
	if strings.Contains(body, "scheduled_at") {
		t.Fatalf("scheduled_at forwarded: %s", body)
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

func TestStreamEventsReportsBodyOverflow(t *testing.T) {
	client := NewClient("https://api.example.test", "secret")
	client.MaxResponseBodyBytes = 8
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return httpResponse(200, "data: {\"id\":\"e1\"}\n\n"), nil
	})}
	err := client.StreamEvents(context.Background(), "/v1/events/stream", func(json.RawMessage) error { return nil })
	apiErr, ok := err.(*APIError)
	if !ok || !strings.Contains(apiErr.Message, "stream response body exceeds") {
		t.Fatalf("error = %#v", err)
	}
}
