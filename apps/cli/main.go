package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := runContext(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "clover-cli:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	return runContext(context.Background(), args)
}

func runContext(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: clover-cli <accounts|environments|messages|send|send-batch|schedule|get|list|cancel|domains|templates|webhooks|logs|api-keys|inbound|provider-events|received|preferences|suppressions|contacts|segments|automations|domain-health|smtp|routing|usage|dev>")
	}
	if args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		fmt.Println("usage: clover-cli <accounts|environments|messages|send|send-batch|schedule|get|list|cancel|domains|templates|webhooks|logs|api-keys|inbound|provider-events|received|preferences|suppressions|contacts|segments|automations|domain-health|smtp|routing|usage|dev>")
		return nil
	}
	if args[0] == "dev" {
		return runDev(args[1:])
	}
	baseURL := os.Getenv("CLOVER_BASE_URL")
	apiKey := os.Getenv("CLOVER_API_KEY")
	client, err := NewClientWithError(baseURL, apiKey)
	if err != nil {
		return err
	}
	switch args[0] {
	case "accounts", "account":
		return runPlatformAccounts(ctx, client, args[1:])
	case "environments", "environment":
		return runPlatformEnvironments(ctx, client, args[1:])
	case "messages", "message":
		return runPlatformMessages(ctx, client, args[1:])
	case "domains", "domain":
		return runPlatformDomains(ctx, client, args[1:])
	case "templates", "template":
		return runPlatformTemplates(ctx, client, args[1:])
	case "webhooks", "webhook":
		return runPlatformWebhooks(ctx, client, args[1:])
	case "api-keys", "keys":
		return runAPIKeys(ctx, client, args[1:])
	case "inbound":
		return runInbound(ctx, client, args[1:])
	case "provider-events":
		return runProviderEvents(ctx, client, args[1:])
	case "received", "received-emails":
		return runPlatformReceived(ctx, client, args[1:])
	case "preferences", "preference":
		return runPlatformPreferences(ctx, client, args[1:])
	case "suppressions", "suppression":
		return runPlatformSuppressions(ctx, client, args[1:])
	case "contacts", "contact":
		return runPlatformContacts(ctx, client, args[1:])
	case "segments", "segment":
		return runPlatformSegments(ctx, client, args[1:])
	case "automations", "automation":
		return runPlatformAutomations(ctx, client, args[1:])
	case "domain-health":
		return runPlatformDomainHealth(ctx, client, args[1:])
	case "smtp", "smtp-credentials":
		if args[0] == "smtp-credentials" {
			return runPlatformSMTP(ctx, client, append([]string{"credentials"}, args[1:]...))
		}
		return runPlatformSMTP(ctx, client, args[1:])
	case "routing", "provider-routing":
		return runPlatformRouting(ctx, client, args[1:])
	case "usage":
		return runPlatformUsage(ctx, client, args[1:])
	case "emails":
		return runEmailReliability(ctx, client, args[1:])
	case "logs":
		return runPlatformLogs(ctx, client, args[1:])
	case "stream":
		return runPlatformLogs(ctx, client, args[1:])
	case "send":
		return runPlatformMessages(ctx, client, append([]string{"send"}, args[1:]...))
	case "send-batch":
		return runPlatformMessages(ctx, client, append([]string{"send-batch"}, args[1:]...))
	case "schedule":
		return runPlatformMessages(ctx, client, append([]string{"schedule"}, args[1:]...))
	case "get":
		return runPlatformMessages(ctx, client, append([]string{"get"}, args[1:]...))
	case "list":
		return runPlatformMessages(ctx, client, append([]string{"list"}, args[1:]...))
	case "cancel":
		return runPlatformMessages(ctx, client, append([]string{"cancel"}, args[1:]...))
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}
func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
