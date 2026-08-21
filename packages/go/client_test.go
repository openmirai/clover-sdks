package clover

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type fakeDoer struct {
	responses []*http.Response
	requests  []*http.Request
	bodies    [][]byte
}

func (f *fakeDoer) Do(r *http.Request) (*http.Response, error) {
	f.requests = append(f.requests, r)
	if r.Body != nil {
		body, _ := io.ReadAll(r.Body)
		f.bodies = append(f.bodies, body)
		r.Body = io.NopCloser(bytes.NewReader(body))
	}
	return f.responses[len(f.requests)-1], nil
}
func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func TestSendAndRetry(t *testing.T) {
	fake := &fakeDoer{responses: []*http.Response{response(503, `{"type":"about:blank","title":"Busy","status":503,"code":"BUSY","request_id":"req_12345678","vendor":{"x":1}}`), response(202, `{"success":true,"data":{"id":"e1","status":"queued","extra":true},"requestId":"req_12345678"}`)}}
	c := NewClient("https://api.example.test", "secret")
	c.HTTPClient = fake
	c.Sleep = func(context.Context, time.Duration) error { return nil }
	result, _, err := c.Send(context.Background(), JSON{"subject": "hi"}, "idem-1234")
	if err != nil {
		t.Fatal(err)
	}
	if result["extra"] != true {
		t.Fatalf("unknown response field lost: %#v", result)
	}
	if got := fake.requests[0].Header.Get("Authorization"); got != "Bearer secret" {
		t.Errorf("authorization = %q", got)
	}
	if got := fake.requests[0].Header.Get("Idempotency-Key"); got != "idem-1234" {
		t.Errorf("idempotency = %q", got)
	}
}

func TestReplayHeaderValueIsCaseInsensitive(t *testing.T) {
	problemResponse := response(400, `{"type":"about:blank","title":"Bad request"}`)
	problemResponse.Header.Set("Idempotency-Replayed", "TrUe")
	c := NewClient("https://api.example.test", "secret")
	c.HTTPClient = &fakeDoer{responses: []*http.Response{problemResponse}}
	_, metadata, err := c.Get(context.Background(), "e1")
	if err == nil {
		t.Fatal("expected problem response")
	}
	if !metadata.Replayed {
		t.Fatal("mixed-case replay metadata was not recognized")
	}
}

func TestProblemAndBoundedRetry(t *testing.T) {
	body := `{"type":"about:blank","title":"Busy","status":503,"code":"BUSY","request_id":"req_12345678","vendor":{"x":1}}`
	fake := &fakeDoer{responses: []*http.Response{response(503, body), response(503, body)}}
	c := NewClient("https://api.example.test", "secret")
	c.HTTPClient = fake
	c.MaxRetries = 1
	c.Sleep = func(context.Context, time.Duration) error { return nil }
	_, _, err := c.Get(context.Background(), "e1")
	apiErr, ok := err.(*Error)
	if !ok || apiErr.Problem == nil {
		t.Fatalf("error = %#v", err)
	}
	if string(apiErr.Problem.Extra["vendor"]) != `{"x":1}` {
		t.Fatalf("extra = %s", apiErr.Problem.Extra["vendor"])
	}
	if len(fake.requests) != 2 {
		t.Fatalf("calls = %d", len(fake.requests))
	}
}

func TestRetryAfterZeroIsHonored(t *testing.T) {
	first := response(503, `{"type":"about:blank","title":"Busy","status":503,"code":"BUSY"}`)
	first.Header.Set("Retry-After", "0")
	fake := &fakeDoer{responses: []*http.Response{first, response(200, `{"success":true,"data":{}}`)}}
	c := NewClient("https://api.example.test", "secret")
	c.HTTPClient = fake
	c.MaxRetries = 1
	c.RetryBaseDelay = time.Second
	var delays []time.Duration
	c.Sleep = func(_ context.Context, delay time.Duration) error {
		delays = append(delays, delay)
		return nil
	}
	if _, _, err := c.Get(context.Background(), "e1"); err != nil {
		t.Fatal(err)
	}
	if len(delays) != 1 || delays[0] != 0 {
		t.Fatalf("retry delays = %#v", delays)
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
		if _, _, err := client.Send(context.Background(), JSON{"subject": "hello"}, key); err == nil {
			t.Fatalf("accepted invalid key %q", key)
		}
	}
	fake := &fakeDoer{responses: []*http.Response{response(202, `{}`), response(202, `{}`)}}
	client.HTTPClient = fake
	for _, key := range []string{strings.Repeat("a", 8), strings.Repeat("a", 128)} {
		if _, _, err := client.Send(context.Background(), JSON{"subject": "hello"}, key); err != nil {
			t.Fatalf("valid key rejected: %v", err)
		}
	}
}

func TestSendBatchPreservesScheduledAt(t *testing.T) {
	fake := &fakeDoer{responses: []*http.Response{response(202, `{}`)}}
	client := NewClient("https://api.example.test", "secret")
	client.HTTPClient = fake
	if _, _, err := client.SendBatch(context.Background(), []JSON{{"subject": "hello", "scheduled_at": "2030-01-01"}}, "batch-1234"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(fake.bodies[0]), "scheduled_at") {
		t.Fatalf("scheduled_at was dropped: %s", fake.bodies[0])
	}
}

func TestOversizedResponseFailsBeforeDecode(t *testing.T) {
	fake := &fakeDoer{responses: []*http.Response{response(200, `{"data":"too long"}`)}}
	client := NewClient("https://api.example.test", "secret")
	client.MaxResponseBodyBytes = 8
	client.HTTPClient = fake
	_, _, err := client.Get(context.Background(), "e1")
	apiErr, ok := err.(*Error)
	if !ok || apiErr.Status != 200 || !strings.Contains(apiErr.Message, "exceeds the configured limit") {
		t.Fatalf("error = %#v", err)
	}
}
