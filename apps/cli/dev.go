package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	devWebhookIDHeader        = "X-Clover-Webhook-Id"
	devWebhookTimestampHeader = "X-Clover-Webhook-Timestamp"
	devWebhookSignatureHeader = "X-Clover-Webhook-Signature"
	devMaxBodyBytes           = 4 << 20
	devReplayWindow           = 5 * time.Minute
)

type devReceiver struct {
	secret []byte
	now    func() time.Time
	out    io.Writer
	mu     sync.Mutex
	seen   map[string]time.Time
}

func runDev(args []string) error {
	flags := flag.NewFlagSet("dev", flag.ContinueOnError)
	listen := flags.String("listen", "127.0.0.1:8788", "local webhook listen address")
	webhookPath := flags.String("path", "/webhooks/clover", "local webhook path")
	secret := flags.String("secret", os.Getenv("CLOVER_WEBHOOK_SECRET"), "webhook signing secret")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*secret) == "" {
		return errors.New("webhook secret is required via --secret or CLOVER_WEBHOOK_SECRET")
	}
	if len(*secret) > 4096 {
		return errors.New("webhook secret exceeds 4096 bytes")
	}
	if !validWebhookPath(*webhookPath) {
		return errors.New("webhook path must be one absolute path without a query or fragment")
	}

	receiver := newDevReceiver([]byte(*secret), time.Now, os.Stdout)
	mux := http.NewServeMux()
	mux.Handle(*webhookPath, receiver)
	mux.HandleFunc("GET /-/health", func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(response, `{"ok":true}`)
	})
	server := &http.Server{
		Addr:              *listen,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	fmt.Fprintf(os.Stderr, "clover dev listening on http://%s%s\n", *listen, *webhookPath)
	return server.ListenAndServe()
}

func validWebhookPath(value string) bool {
	return strings.HasPrefix(value, "/") && value != "/" && !strings.ContainsAny(value, "?#") && !strings.Contains(value, "..")
}

func newDevReceiver(secret []byte, now func() time.Time, out io.Writer) *devReceiver {
	return &devReceiver{secret: append([]byte(nil), secret...), now: now, out: out, seen: make(map[string]time.Time)}
}

func (receiver *devReceiver) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		response.Header().Set("Allow", http.MethodPost)
		http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(request.Body, devMaxBodyBytes+1))
	if err != nil {
		http.Error(response, "request body could not be read", http.StatusBadRequest)
		return
	}
	if len(body) > devMaxBodyBytes {
		http.Error(response, "request body is too large", http.StatusRequestEntityTooLarge)
		return
	}
	if !json.Valid(body) {
		http.Error(response, "request body must be valid JSON", http.StatusBadRequest)
		return
	}
	deliveryID := strings.TrimSpace(request.Header.Get(devWebhookIDHeader))
	rawTimestamp := strings.TrimSpace(request.Header.Get(devWebhookTimestampHeader))
	signature := strings.TrimSpace(request.Header.Get(devWebhookSignatureHeader))
	timestamp, err := strconv.ParseInt(rawTimestamp, 10, 64)
	if err != nil || deliveryID == "" || strings.ContainsAny(deliveryID, "\r\n") {
		http.Error(response, "webhook signature headers are invalid", http.StatusUnauthorized)
		return
	}
	observed := time.Unix(timestamp, 0)
	now := receiver.now().UTC()
	if absoluteDuration(now.Sub(observed)) > devReplayWindow || !verifyDevSignature(receiver.secret, timestamp, deliveryID, body, signature) {
		http.Error(response, "webhook signature is invalid or expired", http.StatusUnauthorized)
		return
	}
	if receiver.replayed(deliveryID, observed, now) {
		http.Error(response, "webhook delivery was already received", http.StatusConflict)
		return
	}

	var compact bytes.Buffer
	if err := json.Compact(&compact, body); err != nil {
		http.Error(response, "request body must be valid JSON", http.StatusBadRequest)
		return
	}
	compact.WriteByte('\n')
	if _, err := receiver.out.Write(compact.Bytes()); err != nil {
		http.Error(response, "event output failed", http.StatusInternalServerError)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func (receiver *devReceiver) replayed(deliveryID string, observed, now time.Time) bool {
	receiver.mu.Lock()
	defer receiver.mu.Unlock()
	for key, seenAt := range receiver.seen {
		if now.Sub(seenAt) > devReplayWindow {
			delete(receiver.seen, key)
		}
	}
	if _, ok := receiver.seen[deliveryID]; ok {
		return true
	}
	receiver.seen[deliveryID] = observed
	return false
}

func verifyDevSignature(secret []byte, timestamp int64, deliveryID string, body []byte, provided string) bool {
	prefix, encoded, ok := strings.Cut(strings.TrimSpace(provided), "=")
	if !ok || prefix != "v1" || len(encoded) != sha256.Size*2 {
		return false
	}
	candidate, err := hex.DecodeString(encoded)
	if err != nil || len(candidate) != sha256.Size {
		return false
	}
	value := strconv.FormatInt(timestamp, 10) + "." + deliveryID + "." + string(body)
	digest := hmac.New(sha256.New, secret)
	_, _ = digest.Write([]byte(value))
	return subtle.ConstantTimeCompare(candidate, digest.Sum(nil)) == 1
}

func absoluteDuration(value time.Duration) time.Duration {
	if value < 0 {
		return -value
	}
	return value
}
