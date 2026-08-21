package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
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
		return fmt.Errorf("usage: clover-cli <send|send-batch|schedule|get|list|cancel|domains|templates|webhooks|logs|api-keys|inbound|provider-events|received|preferences|suppressions|contacts|segments|broadcasts|automations|audit|domain-health|delivery-policies|delivery-routes|smtp-credentials|routing|dev>")
	}
	if args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		fmt.Println("usage: clover-cli <send|send-batch|schedule|get|list|cancel|domains|templates|webhooks|logs|api-keys|inbound|provider-events|received|preferences|suppressions|contacts|segments|broadcasts|automations|audit|domain-health|delivery-policies|delivery-routes|smtp-credentials|routing|dev>")
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
	case "domains", "domain":
		return runDomains(ctx, client, args[1:])
	case "templates", "template":
		return runTemplates(ctx, client, args[1:])
	case "webhooks", "webhook":
		return runWebhooks(ctx, client, args[1:])
	case "api-keys", "keys":
		return runAPIKeys(ctx, client, args[1:])
	case "inbound":
		return runInbound(ctx, client, args[1:])
	case "provider-events":
		return runProviderEvents(ctx, client, args[1:])
	case "received", "received-emails":
		return runReceivedEmails(ctx, client, args[1:])
	case "preferences", "preference":
		return runPreferences(ctx, client, args[1:])
	case "suppressions", "suppression":
		return runSuppressions(ctx, client, args[1:])
	case "contacts", "contact":
		return runContacts(ctx, client, args[1:])
	case "segments", "segment":
		return runSegments(ctx, client, args[1:])
	case "broadcasts", "broadcast":
		return runBroadcasts(ctx, client, args[1:])
	case "automations", "automation":
		return runAutomations(ctx, client, args[1:])
	case "audit", "audit-events":
		return runAudit(ctx, client, args[1:])
	case "domain-health":
		return runDomainHealth(ctx, client, args[1:])
	case "delivery-policies":
		return runDeliveryPolicies(ctx, client, args[1:])
	case "delivery-routes":
		return runDeliveryRoutes(ctx, client, args[1:])
	case "smtp-credentials":
		return runSMTPCredentials(ctx, client, args[1:])
	case "routing", "provider-routing":
		return runRouting(ctx, client, args[1:])
	case "emails":
		return runEmailReliability(ctx, client, args[1:])
	case "logs":
		return runLogs(ctx, client, args[1:], false)
	case "stream":
		return runLogs(ctx, client, args[1:], true)
	case "send":
		flags := flag.NewFlagSet("send", flag.ContinueOnError)
		from := flags.String("from", "", "sender address")
		to := flags.String("to", "", "recipient address")
		subject := flags.String("subject", "", "subject")
		text := flags.String("text", "", "plain text")
		key := flags.String("idempotency-key", "", "required idempotency key")
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		result, _, err := client.Send(ctx, map[string]any{"from": map[string]any{"address": *from}, "to": []any{map[string]any{"address": *to}}, "subject": *subject, "text": *text}, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "send-batch":
		flags := flag.NewFlagSet("send-batch", flag.ContinueOnError)
		itemsJSON := flags.String("items", "[]", "JSON array of email payloads")
		key := flags.String("idempotency-key", "", "required idempotency key")
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		var items []map[string]any
		if err := json.Unmarshal([]byte(*itemsJSON), &items); err != nil {
			return fmt.Errorf("items must be a JSON array: %w", err)
		}
		result, _, err := client.SendBatch(ctx, items, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "schedule":
		flags := flag.NewFlagSet("schedule", flag.ContinueOnError)
		scheduledAt := flags.String("scheduled-at", "", "RFC3339 schedule time")
		key := flags.String("idempotency-key", "", "required idempotency key")
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if flags.NArg() != 1 || *scheduledAt == "" {
			return fmt.Errorf("usage: clover-cli schedule <email-id> -scheduled-at <RFC3339> -idempotency-key <key>")
		}
		result, _, err := client.Schedule(ctx, flags.Arg(0), *scheduledAt, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "get":
		if len(args) != 2 {
			return fmt.Errorf("usage: clover-cli get <email-id>")
		}
		result, _, err := client.Get(ctx, args[1])
		if err != nil {
			return err
		}
		return printJSON(result)
	case "list":
		flags := flag.NewFlagSet("list", flag.ContinueOnError)
		cursor := flags.String("cursor", "", "pagination cursor")
		status := flags.String("status", "", "email status filter")
		limit := flags.Int("limit", 0, "page size")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		query := url.Values{}
		if *cursor != "" {
			query.Set("cursor", *cursor)
		}
		if *status != "" {
			query.Set("status", *status)
		}
		if *limit > 0 {
			query.Set("limit", fmt.Sprint(*limit))
		}
		result, _, err := client.List(ctx, query)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "cancel":
		flags := flag.NewFlagSet("cancel", flag.ContinueOnError)
		key := flags.String("idempotency-key", "", "required idempotency key")
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if flags.NArg() != 1 {
			return fmt.Errorf("usage: clover-cli cancel <email-id> -idempotency-key <key>")
		}
		result, _, err := client.Cancel(ctx, flags.Arg(0), *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}
func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
