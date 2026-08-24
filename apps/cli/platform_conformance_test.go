package main

import (
	"context"
	"os"
	"strings"
	"testing"
)

// TestPlatformBackendConformance is intentionally opt-in. Set the URL, API
// key, account ID, and environment ID to exercise a deployed Phase 1 backend;
// ordinary unit and race runs never make a network request.
func TestPlatformBackendConformance(t *testing.T) {
	baseURL := strings.TrimSpace(os.Getenv("CLOVER_PHASE1_CONFORMANCE_URL"))
	if baseURL == "" {
		t.Skip("set CLOVER_PHASE1_CONFORMANCE_URL to run the deployed backend conformance check")
	}
	apiKey := strings.TrimSpace(os.Getenv("CLOVER_API_KEY"))
	accountID := strings.TrimSpace(os.Getenv("CLOVER_ACCOUNT_ID"))
	environmentID := strings.TrimSpace(os.Getenv("CLOVER_ENVIRONMENT_ID"))
	if apiKey == "" || accountID == "" || environmentID == "" {
		t.Fatal("CLOVER_API_KEY, CLOVER_ACCOUNT_ID, and CLOVER_ENVIRONMENT_ID are required when CLOVER_PHASE1_CONFORMANCE_URL is set")
	}
	client, err := NewClientWithError(baseURL, apiKey)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = client.PlatformListMessages(context.Background(), PlatformScope{AccountID: accountID, EnvironmentID: environmentID}, nil)
	if err != nil {
		t.Fatalf("platform messages conformance request failed: %v", err)
	}
}
