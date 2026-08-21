package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type pageFlagValues struct {
	page   *int
	limit  *int
	cursor *string
}

func addPageFlags(flags *flag.FlagSet, includePage bool) pageFlagValues {
	values := pageFlagValues{limit: flags.Int("limit", 0, "page size"), cursor: flags.String("cursor", "", "pagination cursor")}
	if includePage {
		values.page = flags.Int("page", 0, "page number")
	}
	return values
}

func (values pageFlagValues) query() url.Values {
	query := url.Values{}
	if values.page != nil && *values.page > 0 {
		query.Set("page", strconv.Itoa(*values.page))
	}
	if values.limit != nil && *values.limit > 0 {
		query.Set("limit", strconv.Itoa(*values.limit))
	}
	if values.cursor != nil && *values.cursor != "" {
		query.Set("cursor", *values.cursor)
	}
	return query
}

type providerHeaderFlags struct {
	signature *string
	timestamp *string
	eventID   *string
}

func addProviderHeaderFlags(flags *flag.FlagSet) providerHeaderFlags {
	return providerHeaderFlags{
		signature: flags.String("provider-signature", os.Getenv("CLOVER_PROVIDER_SIGNATURE"), "X-Provider-Signature callback header"),
		timestamp: flags.String("provider-timestamp", os.Getenv("CLOVER_PROVIDER_TIMESTAMP"), "X-Provider-Timestamp callback header"),
		eventID:   flags.String("provider-event-id", os.Getenv("CLOVER_PROVIDER_EVENT_ID"), "X-Provider-Event-ID callback header"),
	}
}

func (values providerHeaderFlags) headers() (http.Header, error) {
	signature := strings.TrimSpace(*values.signature)
	timestamp := strings.TrimSpace(*values.timestamp)
	eventID := strings.TrimSpace(*values.eventID)
	if signature == "" && timestamp == "" && eventID == "" {
		return nil, nil
	}
	if signature == "" || timestamp == "" || eventID == "" {
		return nil, errors.New("provider-signature, provider-timestamp, and provider-event-id must be supplied together")
	}
	headers := make(http.Header)
	headers.Set("X-Provider-Signature", signature)
	headers.Set("X-Provider-Timestamp", timestamp)
	headers.Set("X-Provider-Event-ID", eventID)
	return headers, nil
}

func runAPIKeys(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli api-keys <list|create|update|revoke>")
	}
	switch args[0] {
	case "list":
		flags := flag.NewFlagSet("api-keys list", flag.ContinueOnError)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return errors.New("usage: clover-cli api-keys list")
		}
		result, _, err := client.ListAPIKeys(ctx)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "create":
		flags := flag.NewFlagSet("api-keys create", flag.ContinueOnError)
		bodyFlags := addJSONInputFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		body, err := bodyFlags.read(true, DefaultMaxRequestBodyBytes)
		if err != nil {
			return err
		}
		result, _, err := client.CreateAPIKey(ctx, body)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "update":
		flags := flag.NewFlagSet("api-keys update", flag.ContinueOnError)
		bodyFlags := addJSONInputFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli api-keys update <id> -json '{...}'"); err != nil {
			return err
		}
		body, err := bodyFlags.read(true, DefaultMaxRequestBodyBytes)
		if err != nil {
			return err
		}
		result, _, err := client.UpdateAPIKey(ctx, flags.Arg(0), body)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "revoke", "delete":
		flags := flag.NewFlagSet("api-keys revoke", flag.ContinueOnError)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli api-keys revoke <id>"); err != nil {
			return err
		}
		result, _, err := client.RevokeAPIKey(ctx, flags.Arg(0))
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown api-keys command %q", args[0])
	}
}

func runReceivedEmails(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli received <list|get|attachment>")
	}
	switch args[0] {
	case "list":
		flags := flag.NewFlagSet("received list", flag.ContinueOnError)
		page := addPageFlags(flags, true)
		domainID := flags.String("domain-id", "", "domain UUID filter")
		parseStatus := flags.String("parse-status", "", "parse status filter")
		receivedAfter := flags.String("received-after", "", "RFC3339 lower bound")
		receivedBefore := flags.String("received-before", "", "RFC3339 upper bound")
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		query := page.query()
		if *domainID != "" {
			query.Set("domain_id", *domainID)
		}
		if *parseStatus != "" {
			query.Set("parse_status", *parseStatus)
		}
		if *receivedAfter != "" {
			query.Set("received_after", *receivedAfter)
		}
		if *receivedBefore != "" {
			query.Set("received_before", *receivedBefore)
		}
		result, _, err := client.ListReceivedEmails(ctx, query)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "get":
		flags := flag.NewFlagSet("received get", flag.ContinueOnError)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli received get <id>"); err != nil {
			return err
		}
		result, _, err := client.GetReceivedEmail(ctx, flags.Arg(0))
		if err != nil {
			return err
		}
		return printJSON(result)
	case "attachment":
		flags := flag.NewFlagSet("received attachment", flag.ContinueOnError)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 2, "usage: clover-cli received attachment <email-id> <attachment-id>"); err != nil {
			return err
		}
		result, _, err := client.GetReceivedEmailAttachment(ctx, flags.Arg(0), flags.Arg(1))
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown received command %q", args[0])
	}
}

func runInbound(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli inbound <provider> -json '<payload>'")
	}
	flags := flag.NewFlagSet("inbound", flag.ContinueOnError)
	bodyFlags := addJSONInputFlags(flags)
	providerHeaders := addProviderHeaderFlags(flags)
	if err := flags.Parse(flagArgsFirst(args)); err != nil {
		return err
	}
	if err := requireArgs(flags, 1, "usage: clover-cli inbound <provider> -json '<payload>'"); err != nil {
		return err
	}
	body, err := bodyFlags.read(true, DefaultMaxRequestBodyBytes)
	if err != nil {
		return err
	}
	headers, err := providerHeaders.headers()
	if err != nil {
		return err
	}
	result, _, err := client.AcceptInboundWithHeaders(ctx, flags.Arg(0), body, headers)
	if err != nil {
		return err
	}
	return printJSON(result)
}

func runProviderEvents(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli provider-events <provider> -json '<payload>'")
	}
	flags := flag.NewFlagSet("provider-events", flag.ContinueOnError)
	bodyFlags := addJSONInputFlags(flags)
	providerHeaders := addProviderHeaderFlags(flags)
	if err := flags.Parse(flagArgsFirst(args)); err != nil {
		return err
	}
	if err := requireArgs(flags, 1, "usage: clover-cli provider-events <provider> -json '<payload>'"); err != nil {
		return err
	}
	body, err := bodyFlags.read(true, DefaultMaxRequestBodyBytes)
	if err != nil {
		return err
	}
	headers, err := providerHeaders.headers()
	if err != nil {
		return err
	}
	result, _, err := client.AcceptProviderEventWithHeaders(ctx, flags.Arg(0), body, headers)
	if err != nil {
		return err
	}
	return printJSON(result)
}

func runPreferences(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli preferences <get|update> <token>")
	}
	flags := flag.NewFlagSet("preferences", flag.ContinueOnError)
	bodyFlags := addJSONInputFlags(flags)
	if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
		return err
	}
	if err := requireArgs(flags, 1, "usage: clover-cli preferences get <token>"); err != nil {
		return err
	}
	switch args[0] {
	case "get":
		result, _, err := client.GetPreferenceCenter(ctx, flags.Arg(0))
		if err != nil {
			return err
		}
		return printJSON(result)
	case "update":
		body, err := bodyFlags.read(true, DefaultMaxRequestBodyBytes)
		if err != nil {
			return err
		}
		result, _, err := client.UpdatePreferenceCenter(ctx, flags.Arg(0), body)
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown preferences command %q", args[0])
	}
}

func runSuppressions(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli suppressions <list|create|delete>")
	}
	switch args[0] {
	case "list":
		flags := flag.NewFlagSet("suppressions list", flag.ContinueOnError)
		page := addPageFlags(flags, true)
		active := flags.String("active", "", "active filter (true or false)")
		reason := flags.String("reason", "", "suppression reason")
		addressHash := flags.String("address-sha256", "", "address SHA-256 filter")
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		query := page.query()
		if err := setBooleanQuery(query, "active", *active); err != nil {
			return err
		}
		if *reason != "" {
			query.Set("reason", *reason)
		}
		if *addressHash != "" {
			query.Set("address_sha256", *addressHash)
		}
		result, _, err := client.ListSuppressions(ctx, query)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "create":
		flags := flag.NewFlagSet("suppressions create", flag.ContinueOnError)
		bodyFlags := addJSONInputFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		body, err := bodyFlags.read(true, DefaultMaxRequestBodyBytes)
		if err != nil {
			return err
		}
		result, _, err := client.CreateSuppression(ctx, body)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "delete":
		flags := flag.NewFlagSet("suppressions delete", flag.ContinueOnError)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli suppressions delete <id>"); err != nil {
			return err
		}
		result, _, err := client.DeleteSuppression(ctx, flags.Arg(0))
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown suppressions command %q", args[0])
	}
}

func runDomainHealth(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli domain-health <list|get|verify>")
	}
	switch args[0] {
	case "list":
		flags := flag.NewFlagSet("domain-health list", flag.ContinueOnError)
		page := addPageFlags(flags, true)
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		scope, err := scopeFlags.scope(false)
		if err != nil {
			return err
		}
		result, _, err := client.ListDomainHealth(ctx, page.query(), scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "get":
		flags := flag.NewFlagSet("domain-health get", flag.ContinueOnError)
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli domain-health get <domain-id>"); err != nil {
			return err
		}
		scope, err := scopeFlags.scope(false)
		if err != nil {
			return err
		}
		result, _, err := client.GetDomainHealth(ctx, flags.Arg(0), scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "verify":
		flags := flag.NewFlagSet("domain-health verify", flag.ContinueOnError)
		bodyFlags := addJSONInputFlags(flags)
		key := flags.String("idempotency-key", "", "required idempotency key")
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requiredKey(*key); err != nil {
			return err
		}
		body, err := bodyFlags.read(true, DefaultMaxRequestBodyBytes)
		if err != nil {
			return err
		}
		scope, err := scopeFlags.scope(false)
		if err != nil {
			return err
		}
		result, _, err := client.VerifyDomainHealth(ctx, body, *key, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown domain-health command %q", args[0])
	}
}
