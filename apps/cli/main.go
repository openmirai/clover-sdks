package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "clover-cli:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: clover-cli <send|send-batch|schedule|get|list|cancel|stream>")
	}
	baseURL := os.Getenv("CLOVER_BASE_URL")
	apiKey := os.Getenv("CLOVER_API_KEY")
	client := NewClient(baseURL, apiKey)
	ctx := context.Background()
	switch args[0] {
	case "send":
		flags := flag.NewFlagSet("send", flag.ContinueOnError)
		from := flags.String("from", "", "sender address")
		to := flags.String("to", "", "recipient address")
		subject := flags.String("subject", "", "subject")
		text := flags.String("text", "", "plain text")
		key := flags.String("idempotency-key", "", "required idempotency key")
		if err := flags.Parse(args[1:]); err != nil {
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
		if err := flags.Parse(args[1:]); err != nil {
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
		if err := flags.Parse(args[1:]); err != nil {
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
		if err := flags.Parse(args[1:]); err != nil {
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
	case "stream":
		path := "/v1/events/stream"
		if len(args) == 2 {
			path = args[1]
		}
		return client.StreamEvents(ctx, path, func(event json.RawMessage) error { return printJSON(event) })
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}
func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
