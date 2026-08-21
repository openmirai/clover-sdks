package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestDevReceiverVerifiesPrintsAndRejectsReplay(t *testing.T) {
	now := time.Unix(2_000, 0).UTC()
	body := []byte(`{"type":"email.delivered","data":{"id":"email-1"}}`)
	secret := []byte("local-webhook-secret")
	deliveryID := "whd_123"
	signature := testDevSignature(secret, now.Unix(), deliveryID, body)
	var output bytes.Buffer
	receiver := newDevReceiver(secret, func() time.Time { return now }, &output)

	request := httptest.NewRequest(http.MethodPost, "/webhooks/clover", bytes.NewReader(body))
	request.Header.Set(devWebhookIDHeader, deliveryID)
	request.Header.Set(devWebhookTimestampHeader, strconv.FormatInt(now.Unix(), 10))
	request.Header.Set(devWebhookSignatureHeader, signature)
	response := httptest.NewRecorder()
	receiver.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || strings.TrimSpace(output.String()) != string(body) {
		t.Fatalf("status=%d output=%q", response.Code, output.String())
	}

	replay := httptest.NewRequest(http.MethodPost, "/webhooks/clover", bytes.NewReader(body))
	replay.Header = request.Header.Clone()
	replayResponse := httptest.NewRecorder()
	receiver.ServeHTTP(replayResponse, replay)
	if replayResponse.Code != http.StatusConflict {
		t.Fatalf("replay status=%d", replayResponse.Code)
	}
}

func TestDevReceiverFailsClosed(t *testing.T) {
	now := time.Unix(2_000, 0).UTC()
	receiver := newDevReceiver([]byte("secret"), func() time.Time { return now }, &bytes.Buffer{})
	for name, configure := range map[string]func(*http.Request){
		"missing signature": func(*http.Request) {},
		"stale timestamp": func(request *http.Request) {
			request.Header.Set(devWebhookIDHeader, "whd_stale")
			request.Header.Set(devWebhookTimestampHeader, "1")
			request.Header.Set(devWebhookSignatureHeader, testDevSignature([]byte("secret"), 1, "whd_stale", []byte(`{}`)))
		},
		"wrong signature": func(request *http.Request) {
			request.Header.Set(devWebhookIDHeader, "whd_wrong")
			request.Header.Set(devWebhookTimestampHeader, strconv.FormatInt(now.Unix(), 10))
			request.Header.Set(devWebhookSignatureHeader, testDevSignature([]byte("other"), now.Unix(), "whd_wrong", []byte(`{}`)))
		},
	} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/webhooks/clover", strings.NewReader(`{}`))
			configure(request)
			response := httptest.NewRecorder()
			receiver.ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status=%d", response.Code)
			}
		})
	}
}

func TestDevRequiresSecretBeforeAPIConfiguration(t *testing.T) {
	t.Setenv("CLOVER_WEBHOOK_SECRET", "")
	t.Setenv("CLOVER_BASE_URL", "")
	t.Setenv("CLOVER_API_KEY", "")
	if err := run([]string{"dev"}); err == nil || !strings.Contains(err.Error(), "webhook secret is required") {
		t.Fatalf("error=%v", err)
	}
}

func testDevSignature(secret []byte, timestamp int64, deliveryID string, body []byte) string {
	value := strconv.FormatInt(timestamp, 10) + "." + deliveryID + "." + string(body)
	digest := hmac.New(sha256.New, secret)
	_, _ = digest.Write([]byte(value))
	return "v1=" + hex.EncodeToString(digest.Sum(nil))
}
