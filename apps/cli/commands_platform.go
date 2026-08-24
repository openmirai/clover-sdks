package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type platformScopeFlags struct {
	accountID     *string
	environmentID *string
}

func addPlatformScopeFlags(flags *flag.FlagSet) platformScopeFlags {
	return platformScopeFlags{
		accountID:     flags.String("account-id", strings.TrimSpace(os.Getenv("CLOVER_ACCOUNT_ID")), "account UUID"),
		environmentID: flags.String("environment-id", strings.TrimSpace(os.Getenv("CLOVER_ENVIRONMENT_ID")), "environment UUID"),
	}
}

func (values platformScopeFlags) scope() (PlatformScope, error) {
	return platformScopeFromValues(*values.accountID, *values.environmentID)
}

func platformQuery(page, limit *int, cursor *string) url.Values {
	query := url.Values{}
	if page != nil && *page > 0 {
		query.Set("page", strconv.Itoa(*page))
	}
	if limit != nil && *limit > 0 {
		query.Set("limit", strconv.Itoa(*limit))
	}
	if cursor != nil && strings.TrimSpace(*cursor) != "" {
		query.Set("cursor", strings.TrimSpace(*cursor))
	}
	return query
}

func parsePlatformBody(input jsonInputFlags, required bool) (json.RawMessage, error) {
	return input.read(required, DefaultMaxRequestBodyBytes)
}

func platformKey(flags *flag.FlagSet, required bool) *string {
	message := "optional idempotency key"
	if required {
		message = "required idempotency key"
	}
	return flags.String("idempotency-key", "", message)
}

func validatePlatformKey(key string, required bool) error {
	if required {
		return requiredKey(key)
	}
	return optionalKey(key)
}

func requirePlatformArgs(flags *flag.FlagSet, count int, usage string) error {
	return requireArgs(flags, count, usage)
}

func runPlatformAccounts(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli accounts <list|create|get|update>")
	}
	switch args[0] {
	case "list":
		flags := flag.NewFlagSet("accounts list", flag.ContinueOnError)
		page := flags.Int("page", 0, "page number")
		limit := flags.Int("limit", 0, "page size")
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return errors.New("usage: clover-cli accounts list [-page N] [-limit N]")
		}
		result, _, err := client.PlatformListAccounts(ctx, platformQuery(page, limit, nil))
		if err != nil {
			return err
		}
		return printJSON(result)
	case "create":
		flags := flag.NewFlagSet("accounts create", flag.ContinueOnError)
		bodyFlags := addJSONInputFlags(flags)
		key := platformKey(flags, true)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := validatePlatformKey(*key, true); err != nil {
			return err
		}
		body, err := parsePlatformBody(bodyFlags, true)
		if err != nil {
			return err
		}
		result, _, err := client.PlatformCreateAccount(ctx, body, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "get", "update":
		flags := flag.NewFlagSet("accounts "+args[0], flag.ContinueOnError)
		bodyFlags := addJSONInputFlags(flags)
		key := platformKey(flags, args[0] == "update")
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requirePlatformArgs(flags, 1, "usage: clover-cli accounts get|update <account-id>"); err != nil {
			return err
		}
		if args[0] == "get" {
			result, _, err := client.PlatformGetAccount(ctx, flags.Arg(0))
			if err != nil {
				return err
			}
			return printJSON(result)
		}
		if err := validatePlatformKey(*key, true); err != nil {
			return err
		}
		body, err := parsePlatformBody(bodyFlags, true)
		if err != nil {
			return err
		}
		result, _, err := client.PlatformUpdateAccount(ctx, flags.Arg(0), body, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown accounts command %q", args[0])
	}
}

func runPlatformEnvironments(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli environments <list|create> -account-id <account-id>")
	}
	flags := flag.NewFlagSet("environments "+args[0], flag.ContinueOnError)
	accountID := flags.String("account-id", strings.TrimSpace(os.Getenv("CLOVER_ACCOUNT_ID")), "account UUID")
	page := flags.Int("page", 0, "page number")
	limit := flags.Int("limit", 0, "page size")
	bodyFlags := addJSONInputFlags(flags)
	key := platformKey(flags, args[0] == "create")
	if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
		return err
	}
	if strings.TrimSpace(*accountID) == "" {
		return errors.New("account-id is required")
	}
	switch args[0] {
	case "list":
		if flags.NArg() != 0 {
			return errors.New("usage: clover-cli environments list -account-id <account-id>")
		}
		result, _, err := client.PlatformListEnvironments(ctx, *accountID, platformQuery(page, limit, nil))
		if err != nil {
			return err
		}
		return printJSON(result)
	case "create":
		if err := validatePlatformKey(*key, true); err != nil {
			return err
		}
		body, err := parsePlatformBody(bodyFlags, true)
		if err != nil {
			return err
		}
		result, _, err := client.PlatformCreateEnvironment(ctx, *accountID, body, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown environments command %q", args[0])
	}
}

func runPlatformMessages(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli messages <send|send-batch|get|list|schedule|cancel>")
	}
	flags := flag.NewFlagSet("messages "+args[0], flag.ContinueOnError)
	scopeFlags := addPlatformScopeFlags(flags)
	bodyFlags := addJSONInputFlags(flags)
	key := platformKey(flags, args[0] != "get" && args[0] != "list")
	from := flags.String("from", "", "sender address for convenience send mode")
	to := flags.String("to", "", "recipient address for convenience send mode")
	subject := flags.String("subject", "", "subject for convenience send mode")
	text := flags.String("text", "", "plain-text body for convenience send mode")
	items := flags.String("items", "", "JSON array of message payloads for send-batch")
	scheduledAt := flags.String("scheduled-at", "", "RFC3339 schedule time")
	page := flags.Int("page", 0, "page number")
	limit := flags.Int("limit", 0, "page size")
	status := flags.String("status", "", "message status filter")
	if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
		return err
	}
	scope, err := scopeFlags.scope()
	if err != nil {
		return err
	}
	if args[0] == "get" || args[0] == "schedule" || args[0] == "cancel" {
		if flags.NArg() != 1 {
			return errors.New("usage: clover-cli messages <get|schedule|cancel> <message-id>")
		}
	}
	if args[0] != "get" && args[0] != "list" {
		if err := validatePlatformKey(*key, true); err != nil {
			return err
		}
	}
	switch args[0] {
	case "send":
		body, err := parsePlatformBody(bodyFlags, false)
		if err != nil {
			return err
		}
		if strings.TrimSpace(*bodyFlags.raw) == "" && strings.TrimSpace(*bodyFlags.file) == "" {
			if strings.TrimSpace(*from) == "" || strings.TrimSpace(*to) == "" || strings.TrimSpace(*subject) == "" {
				return errors.New("send requires -json/-file or -from, -to, and -subject")
			}
			payload := map[string]any{"from": map[string]any{"address": *from}, "to": []any{map[string]any{"address": *to}}, "subject": *subject}
			if strings.TrimSpace(*text) != "" {
				payload["text"] = *text
			}
			body, err = json.Marshal(payload)
			if err != nil {
				return err
			}
		}
		result, _, err := client.PlatformSend(ctx, scope, body, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "send-batch":
		body, err := parsePlatformBody(bodyFlags, false)
		if err != nil {
			return err
		}
		if strings.TrimSpace(*bodyFlags.raw) == "" && strings.TrimSpace(*bodyFlags.file) == "" {
			if strings.TrimSpace(*items) == "" {
				return errors.New("send-batch requires -json/-file or -items")
			}
			var values []json.RawMessage
			if err := json.Unmarshal([]byte(*items), &values); err != nil {
				return fmt.Errorf("items must be a JSON array: %w", err)
			}
			body, err = json.Marshal(map[string]any{"items": values})
			if err != nil {
				return err
			}
		}
		result, _, err := client.PlatformSendBatch(ctx, scope, body, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "get":
		result, _, err := client.PlatformGetMessage(ctx, scope, flags.Arg(0))
		if err != nil {
			return err
		}
		return printJSON(result)
	case "list":
		query := platformQuery(page, limit, nil)
		if strings.TrimSpace(*status) != "" {
			query.Set("status", strings.TrimSpace(*status))
		}
		result, _, err := client.PlatformListMessages(ctx, scope, query)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "schedule":
		body, err := parsePlatformBody(bodyFlags, false)
		if err != nil {
			return err
		}
		if strings.TrimSpace(*bodyFlags.raw) == "" && strings.TrimSpace(*bodyFlags.file) == "" {
			if strings.TrimSpace(*scheduledAt) == "" {
				return errors.New("schedule requires -scheduled-at or -json/-file")
			}
			body, err = json.Marshal(map[string]string{"scheduled_at": *scheduledAt})
			if err != nil {
				return err
			}
		}
		result, _, err := client.PlatformScheduleMessage(ctx, scope, flags.Arg(0), body, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "cancel":
		result, _, err := client.PlatformCancelMessage(ctx, scope, flags.Arg(0), *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown messages command %q", args[0])
	}
}

func runPlatformTemplates(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli templates <list|create|get|update|archive|unarchive|versions>")
	}
	if args[0] == "versions" {
		return runPlatformTemplateVersions(ctx, client, args[1:])
	}
	flags := flag.NewFlagSet("templates "+args[0], flag.ContinueOnError)
	scopeFlags := addPlatformScopeFlags(flags)
	bodyFlags := addJSONInputFlags(flags)
	key := platformKey(flags, args[0] == "create" || args[0] == "update" || args[0] == "archive" || args[0] == "unarchive")
	page := flags.Int("page", 0, "page number")
	limit := flags.Int("limit", 0, "page size")
	status := flags.String("status", "", "template status filter")
	if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
		return err
	}
	scope, err := scopeFlags.scope()
	if err != nil {
		return err
	}
	if args[0] != "list" && args[0] != "create" {
		if err := requirePlatformArgs(flags, 1, "usage: clover-cli templates <get|update|archive|unarchive> <template-id>"); err != nil {
			return err
		}
	}
	needsKey := args[0] == "create" || args[0] == "update" || args[0] == "archive" || args[0] == "unarchive"
	if needsKey {
		if err := validatePlatformKey(*key, true); err != nil {
			return err
		}
	}
	switch args[0] {
	case "list":
		query := platformQuery(page, limit, nil)
		if strings.TrimSpace(*status) != "" {
			query.Set("status", *status)
		}
		result, _, err := client.PlatformListTemplates(ctx, scope, query)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "create", "update":
		body, err := parsePlatformBody(bodyFlags, true)
		if err != nil {
			return err
		}
		var result json.RawMessage
		if args[0] == "create" {
			result, _, err = client.PlatformCreateTemplate(ctx, scope, body, *key)
		} else {
			result, _, err = client.PlatformUpdateTemplate(ctx, scope, flags.Arg(0), body, *key)
		}
		if err != nil {
			return err
		}
		return printJSON(result)
	case "get":
		result, _, err := client.PlatformGetTemplate(ctx, scope, flags.Arg(0))
		if err != nil {
			return err
		}
		return printJSON(result)
	case "archive", "unarchive":
		action := args[0]
		result, _, err := client.PlatformTransitionTemplate(ctx, scope, flags.Arg(0), action, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown templates command %q", args[0])
	}
}

func runPlatformTemplateVersions(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli templates versions <list|create|get|compare|publish|render|rollback> <template-id>")
	}
	flags := flag.NewFlagSet("templates versions "+args[0], flag.ContinueOnError)
	scopeFlags := addPlatformScopeFlags(flags)
	bodyFlags := addJSONInputFlags(flags)
	key := platformKey(flags, args[0] == "create" || args[0] == "publish" || args[0] == "rollback")
	from := flags.String("from", "", "source version")
	to := flags.String("to", "", "target version")
	if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
		return err
	}
	scope, err := scopeFlags.scope()
	if err != nil {
		return err
	}
	if flags.NArg() < 1 {
		return errors.New("template-id is required")
	}
	templateID := flags.Arg(0)
	if args[0] == "compare" {
		if strings.TrimSpace(*from) == "" || strings.TrimSpace(*to) == "" {
			return errors.New("compare requires -from and -to")
		}
	}
	if args[0] == "get" || args[0] == "publish" || args[0] == "render" || args[0] == "rollback" {
		if flags.NArg() != 2 {
			return errors.New("version-ref is required")
		}
	}
	if args[0] == "create" || args[0] == "publish" || args[0] == "rollback" {
		if err := validatePlatformKey(*key, true); err != nil {
			return err
		}
	}
	switch args[0] {
	case "list":
		result, _, err := client.PlatformListTemplateVersions(ctx, scope, templateID)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "create":
		body, err := parsePlatformBody(bodyFlags, true)
		if err != nil {
			return err
		}
		result, _, err := client.PlatformCreateTemplateVersion(ctx, scope, templateID, body, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "get":
		result, _, err := client.PlatformGetTemplateVersion(ctx, scope, templateID, flags.Arg(1))
		if err != nil {
			return err
		}
		return printJSON(result)
	case "compare":
		result, _, err := client.PlatformCompareTemplateVersions(ctx, scope, templateID, *from, *to)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "publish", "rollback":
		versionRef := flags.Arg(1)
		var result json.RawMessage
		if args[0] == "publish" {
			result, _, err = client.PlatformPublishTemplateVersion(ctx, scope, templateID, versionRef, *key)
		} else {
			result, _, err = client.PlatformRollbackTemplateVersion(ctx, scope, templateID, versionRef, *key)
		}
		if err != nil {
			return err
		}
		return printJSON(result)
	case "render":
		body, err := parsePlatformBody(bodyFlags, false)
		if err != nil {
			return err
		}
		result, _, err := client.PlatformRenderTemplateVersion(ctx, scope, templateID, flags.Arg(1), body)
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown template versions command %q", args[0])
	}
}

func runPlatformWebhooks(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli webhooks <list|create|get|update|delete|rotate|deliveries>")
	}
	if args[0] == "deliveries" {
		return runPlatformWebhookDeliveries(ctx, client, args[1:])
	}
	flags := flag.NewFlagSet("webhooks "+args[0], flag.ContinueOnError)
	scopeFlags := addPlatformScopeFlags(flags)
	bodyFlags := addJSONInputFlags(flags)
	page := flags.Int("page", 0, "page number")
	limit := flags.Int("limit", 0, "page size")
	key := platformKey(flags, args[0] == "create")
	cursor := flags.String("cursor", "", "pagination cursor")
	enabled := flags.String("enabled", "", "enabled filter (true or false)")
	if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
		return err
	}
	scope, err := scopeFlags.scope()
	if err != nil {
		return err
	}
	if args[0] != "list" && args[0] != "create" {
		if err := requirePlatformArgs(flags, 1, "usage: clover-cli webhooks <get|update|delete|rotate> <webhook-id>"); err != nil {
			return err
		}
	}
	switch args[0] {
	case "list":
		query := platformQuery(page, limit, cursor)
		if strings.TrimSpace(*enabled) != "" {
			if err := setBooleanQuery(query, "enabled", *enabled); err != nil {
				return err
			}
		}
		result, _, err := client.PlatformListWebhooks(ctx, scope, query)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "create", "update", "rotate":
		if args[0] == "create" {
			if err := validatePlatformKey(*key, true); err != nil {
				return err
			}
		}
		body, err := parsePlatformBody(bodyFlags, args[0] != "rotate")
		if err != nil {
			return err
		}
		var result json.RawMessage
		if args[0] == "create" {
			result, _, err = client.PlatformCreateWebhook(ctx, scope, body, *key)
		} else if args[0] == "update" {
			result, _, err = client.PlatformUpdateWebhook(ctx, scope, flags.Arg(0), body)
		} else {
			result, _, err = client.PlatformRotateWebhookSecret(ctx, scope, flags.Arg(0), body)
		}
		if err != nil {
			return err
		}
		return printJSON(result)
	case "get":
		result, _, err := client.PlatformGetWebhook(ctx, scope, flags.Arg(0))
		if err != nil {
			return err
		}
		return printJSON(result)
	case "delete":
		result, _, err := client.PlatformDeleteWebhook(ctx, scope, flags.Arg(0))
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown webhooks command %q", args[0])
	}
}

func runPlatformWebhookDeliveries(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli webhooks deliveries <list|get|replay>")
	}
	flags := flag.NewFlagSet("webhooks deliveries "+args[0], flag.ContinueOnError)
	scopeFlags := addPlatformScopeFlags(flags)
	page := flags.Int("page", 0, "page number")
	limit := flags.Int("limit", 0, "page size")
	cursor := flags.String("cursor", "", "pagination cursor")
	endpointID := flags.String("endpoint-id", "", "webhook endpoint UUID filter")
	eventType := flags.String("event-type", "", "event type filter")
	status := flags.String("status", "", "delivery status filter")
	if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
		return err
	}
	scope, err := scopeFlags.scope()
	if err != nil {
		return err
	}
	if args[0] != "list" && flags.NArg() != 1 {
		return errors.New("delivery-id is required")
	}
	switch args[0] {
	case "list":
		query := platformQuery(page, limit, cursor)
		if *endpointID != "" {
			query.Set("endpoint_id", *endpointID)
		}
		if *eventType != "" {
			query.Set("event_type", *eventType)
		}
		if *status != "" {
			query.Set("status", *status)
		}
		result, _, err := client.PlatformListWebhookDeliveries(ctx, scope, query)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "get":
		result, _, err := client.PlatformGetWebhookDelivery(ctx, scope, flags.Arg(0))
		if err != nil {
			return err
		}
		return printJSON(result)
	case "replay":
		result, _, err := client.PlatformReplayWebhookDelivery(ctx, scope, flags.Arg(0))
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown webhook deliveries command %q", args[0])
	}
}

func runPlatformLogs(ctx context.Context, client *Client, args []string) error {
	flags := flag.NewFlagSet("logs", flag.ContinueOnError)
	scopeFlags := addPlatformScopeFlags(flags)
	queryText := flags.String("q", "", "message or reference search")
	messageKind := flags.String("message-kind", "", "message kind filter")
	messageRef := flags.String("message-ref", "", "message reference filter")
	eventType := flags.String("event-type", "", "event type filter")
	status := flags.String("status", "", "status filter")
	occurredAfter := flags.String("occurred-after", "", "RFC3339 lower bound")
	occurredBefore := flags.String("occurred-before", "", "RFC3339 upper bound")
	page := flags.Int("page", 0, "page number")
	limit := flags.Int("limit", 0, "page size")
	cursor := flags.String("cursor", "", "pagination cursor")
	if err := flags.Parse(flagArgsFirst(args)); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("usage: clover-cli logs [-account-id id] [-environment-id id]")
	}
	scope, err := scopeFlags.scope()
	if err != nil {
		return err
	}
	query := platformQuery(page, limit, cursor)
	for key, value := range map[string]string{"q": *queryText, "message_kind": *messageKind, "message_ref": *messageRef, "event_type": *eventType, "status": *status, "occurred_after": *occurredAfter, "occurred_before": *occurredBefore} {
		if strings.TrimSpace(value) != "" {
			query.Set(key, value)
		}
	}
	result, _, err := client.PlatformListMessageTimeline(ctx, scope, query)
	if err != nil {
		return err
	}
	return printJSON(result)
}

func runPlatformPreferences(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli preferences <topics|token|get|update>")
	}
	if args[0] == "get" || args[0] == "update" {
		// Public preference-center tokens remain intentionally separate from the
		// account/environment control-plane resources.
		return runPreferences(ctx, client, args)
	}
	flags := flag.NewFlagSet("preferences "+args[0], flag.ContinueOnError)
	scopeFlags := addPlatformScopeFlags(flags)
	bodyFlags := addJSONInputFlags(flags)
	key := platformKey(flags, args[0] == "topics" && len(args) > 1 && args[1] == "create")
	if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
		return err
	}
	scope, err := scopeFlags.scope()
	if err != nil {
		return err
	}
	switch args[0] {
	case "topics":
		if len(args) < 2 {
			return errors.New("usage: clover-cli preferences topics <list|create>")
		}
		if args[1] == "list" {
			result, _, err := client.PlatformListPreferenceTopics(ctx, scope)
			if err != nil {
				return err
			}
			return printJSON(result)
		}
		if args[1] != "create" {
			return fmt.Errorf("unknown preferences topics command %q", args[1])
		}
		if err := validatePlatformKey(*key, true); err != nil {
			return err
		}
		body, err := parsePlatformBody(bodyFlags, true)
		if err != nil {
			return err
		}
		result, _, err := client.PlatformCreatePreferenceTopic(ctx, scope, body, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "token", "tokens":
		if args[0] == "tokens" && len(args) > 1 {
			return fmt.Errorf("unknown preferences tokens command %q", args[1])
		}
		body, err := parsePlatformBody(bodyFlags, true)
		if err != nil {
			return err
		}
		result, _, err := client.PlatformCreatePreferenceToken(ctx, scope, body)
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown preferences command %q", args[0])
	}
}

func runPlatformSuppressions(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli suppressions <list|create|delete|reactivate>")
	}
	flags := flag.NewFlagSet("suppressions "+args[0], flag.ContinueOnError)
	scopeFlags := addPlatformScopeFlags(flags)
	bodyFlags := addJSONInputFlags(flags)
	page := flags.Int("page", 0, "page number")
	limit := flags.Int("limit", 0, "page size")
	cursor := flags.String("cursor", "", "pagination cursor")
	active := flags.String("active", "", "active filter")
	reason := flags.String("reason", "", "suppression reason")
	emailHash := flags.String("email-sha256", "", "email SHA-256 filter")
	key := platformKey(flags, args[0] == "create")
	if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
		return err
	}
	scope, err := scopeFlags.scope()
	if err != nil {
		return err
	}
	if args[0] != "list" && args[0] != "create" && flags.NArg() != 1 {
		return errors.New("suppression-id is required")
	}
	switch args[0] {
	case "list":
		query := platformQuery(page, limit, cursor)
		if *active != "" {
			if err := setBooleanQuery(query, "active", *active); err != nil {
				return err
			}
		}
		for key, value := range map[string]string{"reason": *reason, "email_sha256": *emailHash} {
			if strings.TrimSpace(value) != "" {
				query.Set(key, value)
			}
		}
		result, _, err := client.PlatformListSuppressions(ctx, scope, query)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "create":
		if err := validatePlatformKey(*key, true); err != nil {
			return err
		}
		body, err := parsePlatformBody(bodyFlags, true)
		if err != nil {
			return err
		}
		result, _, err := client.PlatformCreateSuppression(ctx, scope, body, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "delete", "reactivate":
		var result json.RawMessage
		if args[0] == "delete" {
			result, _, err = client.PlatformDeleteSuppression(ctx, scope, flags.Arg(0))
		} else {
			result, _, err = client.PlatformReactivateSuppression(ctx, scope, flags.Arg(0))
		}
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown suppressions command %q", args[0])
	}
}

func runPlatformReceived(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli received <list|get|attachment>")
	}
	flags := flag.NewFlagSet("received "+args[0], flag.ContinueOnError)
	scopeFlags := addPlatformScopeFlags(flags)
	page := flags.Int("page", 0, "page number")
	limit := flags.Int("limit", 0, "page size")
	cursor := flags.String("cursor", "", "pagination cursor")
	parseStatus := flags.String("parse-status", "", "parse status filter")
	domainID := flags.String("domain-id", "", "domain UUID filter")
	receivedAfter := flags.String("received-after", "", "RFC3339 lower bound")
	receivedBefore := flags.String("received-before", "", "RFC3339 upper bound")
	if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
		return err
	}
	scope, err := scopeFlags.scope()
	if err != nil {
		return err
	}
	if args[0] != "list" && flags.NArg() < 1 {
		return errors.New("message-id is required")
	}
	switch args[0] {
	case "list":
		query := platformQuery(page, limit, cursor)
		for key, value := range map[string]string{"parse_status": *parseStatus, "domain_id": *domainID, "received_after": *receivedAfter, "received_before": *receivedBefore} {
			if strings.TrimSpace(value) != "" {
				query.Set(key, value)
			}
		}
		result, _, err := client.PlatformListReceivedMessages(ctx, scope, query)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "get":
		if flags.NArg() != 1 {
			return errors.New("usage: clover-cli received get <message-id>")
		}
		result, _, err := client.PlatformGetReceivedMessage(ctx, scope, flags.Arg(0))
		if err != nil {
			return err
		}
		return printJSON(result)
	case "attachment":
		if flags.NArg() != 2 {
			return errors.New("usage: clover-cli received attachment <message-id> <attachment-id>")
		}
		result, _, err := client.PlatformGetReceivedAttachment(ctx, scope, flags.Arg(0), flags.Arg(1))
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown received command %q", args[0])
	}
}

func runPlatformSMTP(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli smtp <credentials|submissions>")
	}
	flags := flag.NewFlagSet("smtp "+args[0], flag.ContinueOnError)
	scopeFlags := addPlatformScopeFlags(flags)
	bodyFlags := addJSONInputFlags(flags)
	page := flags.Int("page", 0, "page number")
	limit := flags.Int("limit", 0, "page size")
	cursor := flags.String("cursor", "", "pagination cursor")
	status := flags.String("status", "", "submission status filter")
	includeRevoked := flags.String("include-revoked", "", "include revoked credentials")
	credentialID := flags.String("credential-id", "", "credential UUID filter")
	key := platformKey(flags, true)
	commandArgs := args[1:]
	if len(args) > 1 {
		commandArgs = args[2:]
	}
	if err := flags.Parse(flagArgsFirst(commandArgs)); err != nil {
		return err
	}
	scope, err := scopeFlags.scope()
	if err != nil {
		return err
	}
	if args[0] == "credentials" || args[0] == "credential" {
		if len(args) < 2 {
			return errors.New("usage: clover-cli smtp credentials <list|create|get|delete|revoke|rotate>")
		}
		action := args[1]
		if action == "list" {
			query := platformQuery(page, limit, cursor)
			if *includeRevoked != "" {
				if err := setBooleanQuery(query, "includeRevoked", *includeRevoked); err != nil {
					return err
				}
			}
			result, _, err := client.PlatformListSMTPCredentials(ctx, scope, query)
			if err != nil {
				return err
			}
			return printJSON(result)
		}
		if action == "create" {
			if err := validatePlatformKey(*key, true); err != nil {
				return err
			}
			body, err := parsePlatformBody(bodyFlags, true)
			if err != nil {
				return err
			}
			result, _, err := client.PlatformCreateSMTPCredential(ctx, scope, body, *key)
			if err != nil {
				return err
			}
			return printJSON(result)
		}
		if flags.NArg() < 1 {
			return errors.New("credential-id is required")
		}
		id := flags.Arg(0)
		if action != "get" {
			if err := validatePlatformKey(*key, true); err != nil {
				return err
			}
		}
		var result json.RawMessage
		switch action {
		case "get":
			result, _, err = client.PlatformGetSMTPCredential(ctx, scope, id)
		case "delete":
			result, _, err = client.PlatformDeleteSMTPCredential(ctx, scope, id, *key)
		case "revoke":
			result, _, err = client.PlatformRevokeSMTPCredential(ctx, scope, id, *key)
		case "rotate":
			result, _, err = client.PlatformRotateSMTPCredential(ctx, scope, id, *key)
		default:
			return fmt.Errorf("unknown smtp credentials command %q", action)
		}
		if err != nil {
			return err
		}
		return printJSON(result)
	}
	if args[0] != "submissions" {
		return fmt.Errorf("unknown smtp command %q", args[0])
	}
	if len(args) < 2 {
		return errors.New("usage: clover-cli smtp submissions <list|get>")
	}
	switch args[1] {
	case "list":
		query := platformQuery(page, limit, cursor)
		for key, value := range map[string]string{"status": *status, "credentialId": *credentialID} {
			if strings.TrimSpace(value) != "" {
				query.Set(key, value)
			}
		}
		result, _, err := client.PlatformListSMTPSubmissions(ctx, scope, query)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "get":
		if flags.NArg() != 1 {
			return errors.New("usage: clover-cli smtp submissions get <submission-id>")
		}
		result, _, err := client.PlatformGetSMTPSubmission(ctx, scope, flags.Arg(0))
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown smtp submissions command %q", args[1])
	}
}

func runPlatformContacts(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli contacts <list|create|get|update|transition>")
	}
	flags := flag.NewFlagSet("contacts "+args[0], flag.ContinueOnError)
	scopeFlags := addPlatformScopeFlags(flags)
	bodyFlags := addJSONInputFlags(flags)
	page := flags.Int("page", 0, "page number")
	limit := flags.Int("limit", 0, "page size")
	key := platformKey(flags, args[0] == "create")
	if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
		return err
	}
	scope, err := scopeFlags.scope()
	if err != nil {
		return err
	}
	if args[0] != "list" && args[0] != "create" && flags.NArg() != 1 {
		return errors.New("contact-id is required")
	}
	switch args[0] {
	case "list":
		result, _, err := client.PlatformListContacts(ctx, scope, platformQuery(page, limit, nil))
		if err != nil {
			return err
		}
		return printJSON(result)
	case "create":
		if err := validatePlatformKey(*key, true); err != nil {
			return err
		}
		body, err := parsePlatformBody(bodyFlags, true)
		if err != nil {
			return err
		}
		result, _, err := client.PlatformCreateContact(ctx, scope, body, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "get":
		result, _, err := client.PlatformGetContact(ctx, scope, flags.Arg(0))
		if err != nil {
			return err
		}
		return printJSON(result)
	case "update", "transition":
		body, err := parsePlatformBody(bodyFlags, true)
		if err != nil {
			return err
		}
		var result json.RawMessage
		if args[0] == "update" {
			result, _, err = client.PlatformUpdateContact(ctx, scope, flags.Arg(0), body)
		} else {
			result, _, err = client.PlatformTransitionContact(ctx, scope, flags.Arg(0), body)
		}
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown contacts command %q", args[0])
	}
}

func runPlatformSegments(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli segments <list|create|get|preview>")
	}
	flags := flag.NewFlagSet("segments "+args[0], flag.ContinueOnError)
	scopeFlags := addPlatformScopeFlags(flags)
	bodyFlags := addJSONInputFlags(flags)
	page := flags.Int("page", 0, "page number")
	limit := flags.Int("limit", 0, "page size")
	if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
		return err
	}
	scope, err := scopeFlags.scope()
	if err != nil {
		return err
	}
	if args[0] != "list" && args[0] != "create" && flags.NArg() != 1 {
		return errors.New("segment-id is required")
	}
	switch args[0] {
	case "list":
		result, _, err := client.PlatformListSegments(ctx, scope, platformQuery(page, limit, nil))
		if err != nil {
			return err
		}
		return printJSON(result)
	case "create":
		body, err := parsePlatformBody(bodyFlags, true)
		if err != nil {
			return err
		}
		result, _, err := client.PlatformCreateSegment(ctx, scope, body)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "get":
		result, _, err := client.PlatformGetSegment(ctx, scope, flags.Arg(0))
		if err != nil {
			return err
		}
		return printJSON(result)
	case "preview":
		result, _, err := client.PlatformPreviewSegment(ctx, scope, flags.Arg(0))
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown segments command %q", args[0])
	}
}

func runPlatformAutomations(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli automations <list|create|get|transition|run|run-get>")
	}
	flags := flag.NewFlagSet("automations "+args[0], flag.ContinueOnError)
	scopeFlags := addPlatformScopeFlags(flags)
	bodyFlags := addJSONInputFlags(flags)
	page := flags.Int("page", 0, "page number")
	limit := flags.Int("limit", 0, "page size")
	key := platformKey(flags, args[0] == "run")
	if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
		return err
	}
	scope, err := scopeFlags.scope()
	if err != nil {
		return err
	}
	if args[0] != "list" && args[0] != "create" && flags.NArg() < 1 {
		return errors.New("automation-id is required")
	}
	switch args[0] {
	case "list":
		result, _, err := client.PlatformListAutomations(ctx, scope, platformQuery(page, limit, nil))
		if err != nil {
			return err
		}
		return printJSON(result)
	case "create":
		body, err := parsePlatformBody(bodyFlags, true)
		if err != nil {
			return err
		}
		result, _, err := client.PlatformCreateAutomation(ctx, scope, body)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "get":
		result, _, err := client.PlatformGetAutomation(ctx, scope, flags.Arg(0))
		if err != nil {
			return err
		}
		return printJSON(result)
	case "transition":
		body, err := parsePlatformBody(bodyFlags, true)
		if err != nil {
			return err
		}
		result, _, err := client.PlatformTransitionAutomation(ctx, scope, flags.Arg(0), body)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "run":
		if err := validatePlatformKey(*key, true); err != nil {
			return err
		}
		body, err := parsePlatformBody(bodyFlags, false)
		if err != nil {
			return err
		}
		result, _, err := client.PlatformStartAutomationRun(ctx, scope, flags.Arg(0), body, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "run-get":
		if flags.NArg() != 1 {
			return errors.New("usage: clover-cli automations run-get <run-id>")
		}
		result, _, err := client.PlatformGetAutomationRun(ctx, scope, flags.Arg(0))
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown automations command %q", args[0])
	}
}

func runPlatformRouting(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli routing <policy|pools|providers|resolve>")
	}
	if args[0] == "pools" || args[0] == "providers" {
		return runPlatformRoutingResource(ctx, client, args[0], args[1:])
	}
	flags := flag.NewFlagSet("routing "+args[0], flag.ContinueOnError)
	scopeFlags := addPlatformScopeFlags(flags)
	bodyFlags := addJSONInputFlags(flags)
	key := platformKey(flags, args[0] == "put")
	if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
		return err
	}
	scope, err := scopeFlags.scope()
	if err != nil {
		return err
	}
	switch args[0] {
	case "policy":
		if flags.NArg() != 1 {
			return errors.New("usage: clover-cli routing policy <get|put>")
		}
		action := flags.Arg(0)
		if action == "get" {
			result, _, err := client.PlatformGetRoutingPolicy(ctx, scope)
			if err != nil {
				return err
			}
			return printJSON(result)
		}
		if action != "put" {
			return fmt.Errorf("unknown routing policy command %q", action)
		}
		if err := validatePlatformKey(*key, true); err != nil {
			return err
		}
		body, err := parsePlatformBody(bodyFlags, true)
		if err != nil {
			return err
		}
		result, _, err := client.PlatformPutRoutingPolicy(ctx, scope, body, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "resolve":
		body, err := parsePlatformBody(bodyFlags, true)
		if err != nil {
			return err
		}
		result, _, err := client.PlatformResolveRouting(ctx, scope, body)
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown routing command %q", args[0])
	}
}

func runPlatformRoutingResource(ctx context.Context, client *Client, resource string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: clover-cli routing %s <list|get|put|routes|ip>", resource)
	}
	flags := flag.NewFlagSet("routing "+resource+" "+args[0], flag.ContinueOnError)
	scopeFlags := addPlatformScopeFlags(flags)
	bodyFlags := addJSONInputFlags(flags)
	key := platformKey(flags, args[0] == "put" || args[0] == "ip" || args[0] == "routes")
	if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
		return err
	}
	scope, err := scopeFlags.scope()
	if err != nil {
		return err
	}
	if args[0] == "list" {
		var result json.RawMessage
		if resource == "pools" {
			result, _, err = client.PlatformListRoutingPools(ctx, scope)
		} else {
			result, _, err = client.PlatformListRoutingProviders(ctx, scope)
		}
		if err != nil {
			return err
		}
		return printJSON(result)
	}
	if args[0] == "get" || args[0] == "put" {
		if flags.NArg() != 1 {
			return errors.New("routing resource id is required")
		}
		id := flags.Arg(0)
		if args[0] == "get" {
			var result json.RawMessage
			if resource == "pools" {
				result, _, err = client.PlatformGetRoutingPool(ctx, scope, id)
			} else {
				return errors.New("routing providers get is not a supported backend operation")
			}
			if err != nil {
				return err
			}
			return printJSON(result)
		}
		if err := validatePlatformKey(*key, true); err != nil {
			return err
		}
		body, err := parsePlatformBody(bodyFlags, true)
		if err != nil {
			return err
		}
		var result json.RawMessage
		if resource == "pools" {
			result, _, err = client.PlatformPutRoutingPool(ctx, scope, id, body, *key)
		} else {
			result, _, err = client.PlatformPutRoutingProvider(ctx, scope, id, body, *key)
		}
		if err != nil {
			return err
		}
		return printJSON(result)
	}
	if resource == "pools" && args[0] == "ip" {
		if flags.NArg() != 2 {
			return errors.New("usage: clover-cli routing pools ip <pool-id> <ip-id> -json '{...}'")
		}
		if err := validatePlatformKey(*key, true); err != nil {
			return err
		}
		body, err := parsePlatformBody(bodyFlags, true)
		if err != nil {
			return err
		}
		result, _, err := client.PlatformPutRoutingIP(ctx, scope, flags.Arg(0), flags.Arg(1), body, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	}
	if resource == "providers" && args[0] == "routes" {
		if flags.NArg() != 1 {
			return errors.New("usage: clover-cli routing providers routes <provider-id> [-json '{...}']")
		}
		if err := validatePlatformKey(*key, true); err != nil {
			return err
		}
		body, err := parsePlatformBody(bodyFlags, true)
		if err != nil {
			return err
		}
		result, _, err := client.PlatformPutRoutingProviderRoutes(ctx, scope, flags.Arg(0), body, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	}
	return fmt.Errorf("unknown routing %s command %q", resource, args[0])
}

func runPlatformDomainHealth(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli domain-health <list|check|reports|receiving|verify>")
	}
	flags := flag.NewFlagSet("domain-health "+args[0], flag.ContinueOnError)
	scopeFlags := addPlatformScopeFlags(flags)
	bodyFlags := addJSONInputFlags(flags)
	key := platformKey(flags, args[0] == "verify")
	limit := flags.Int("limit", 0, "page size")
	cursor := flags.String("cursor", "", "pagination cursor")
	domainFilter := flags.String("domain-id", "", "domain UUID for health reports")
	from := flags.String("from", "", "RFC3339 lower bound")
	to := flags.String("to", "", "RFC3339 upper bound")
	if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
		return err
	}
	scope, err := scopeFlags.scope()
	if err != nil {
		return err
	}
	if args[0] == "list" {
		if strings.TrimSpace(*domainFilter) != "" {
			query := platformQuery(nil, limit, cursor)
			if strings.TrimSpace(*from) != "" {
				query.Set("from", *from)
			}
			if strings.TrimSpace(*to) != "" {
				query.Set("to", *to)
			}
			result, _, err := client.PlatformListDomainHealthReports(ctx, scope, *domainFilter, query)
			if err != nil {
				return err
			}
			return printJSON(result)
		}
		result, _, err := client.PlatformListDomains(ctx, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	}
	if flags.NArg() < 1 {
		return errors.New("domain-id is required")
	}
	domainID := flags.Arg(0)
	switch args[0] {
	case "check":
		result, _, err := client.PlatformCheckDomainHealth(ctx, scope, domainID)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "reports":
		query := platformQuery(nil, limit, cursor)
		if strings.TrimSpace(*from) != "" {
			query.Set("from", *from)
		}
		if strings.TrimSpace(*to) != "" {
			query.Set("to", *to)
		}
		result, _, err := client.PlatformListDomainHealthReports(ctx, scope, domainID, query)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "verify":
		if err := validatePlatformKey(*key, true); err != nil {
			return err
		}
		result, _, err := client.PlatformVerifyDomain(ctx, scope, domainID, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "receiving":
		if flags.NArg() < 2 {
			return errors.New("usage: clover-cli domain-health receiving <get|set|delete> <domain-id>")
		}
		action := domainID
		domainID = flags.Arg(1)
		var result json.RawMessage
		switch action {
		case "get":
			result, _, err = client.PlatformGetDomainReceiving(ctx, scope, domainID)
		case "set":
			body, bodyErr := parsePlatformBody(bodyFlags, true)
			if bodyErr != nil {
				return bodyErr
			}
			result, _, err = client.PlatformSetDomainReceiving(ctx, scope, domainID, body)
		case "delete":
			result, _, err = client.PlatformDeleteDomainReceiving(ctx, scope, domainID)
		default:
			return fmt.Errorf("unknown receiving command %q", action)
		}
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown domain-health command %q", args[0])
	}
}

func runPlatformDomains(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli domains <list|start|verify|health|receiving>")
	}
	if args[0] == "health" {
		return runPlatformDomainHealth(ctx, client, args[1:])
	}
	flags := flag.NewFlagSet("domains "+args[0], flag.ContinueOnError)
	scopeFlags := addPlatformScopeFlags(flags)
	bodyFlags := addJSONInputFlags(flags)
	key := platformKey(flags, args[0] == "start" || args[0] == "verify")
	if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
		return err
	}
	scope, err := scopeFlags.scope()
	if err != nil {
		return err
	}
	switch args[0] {
	case "list":
		result, _, err := client.PlatformListDomains(ctx, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "start", "create":
		if err := validatePlatformKey(*key, true); err != nil {
			return err
		}
		body, err := parsePlatformBody(bodyFlags, true)
		if err != nil {
			return err
		}
		result, _, err := client.PlatformStartDomain(ctx, scope, body, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "verify":
		if flags.NArg() != 1 {
			return errors.New("usage: clover-cli domains verify <domain-id> -idempotency-key <key>")
		}
		if err := validatePlatformKey(*key, true); err != nil {
			return err
		}
		result, _, err := client.PlatformVerifyDomain(ctx, scope, flags.Arg(0), *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "receiving":
		if flags.NArg() < 2 {
			return errors.New("usage: clover-cli domains receiving <get|set|delete> <domain-id>")
		}
		action, domainID := flags.Arg(0), flags.Arg(1)
		var result json.RawMessage
		switch action {
		case "get":
			result, _, err = client.PlatformGetDomainReceiving(ctx, scope, domainID)
		case "set":
			body, bodyErr := parsePlatformBody(bodyFlags, true)
			if bodyErr != nil {
				return bodyErr
			}
			result, _, err = client.PlatformSetDomainReceiving(ctx, scope, domainID, body)
		case "delete":
			result, _, err = client.PlatformDeleteDomainReceiving(ctx, scope, domainID)
		default:
			return fmt.Errorf("unknown receiving command %q", action)
		}
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown domains command %q", args[0])
	}
}

func runPlatformUsage(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli usage <vocabulary|facts|export|correct|reconciliations>")
	}
	if args[0] == "reconciliations" || args[0] == "reconciliation" {
		return runPlatformReconciliations(ctx, client, args[1:])
	}
	flags := flag.NewFlagSet("usage "+args[0], flag.ContinueOnError)
	scopeFlags := addPlatformScopeFlags(flags)
	bodyFlags := addJSONInputFlags(flags)
	page := flags.Int("page", 0, "page number")
	limit := flags.Int("limit", 0, "page size")
	cursor := flags.String("cursor", "", "pagination cursor")
	factType := flags.String("fact-type", "", "fact type filter")
	family := flags.String("family", "", "fact family filter")
	sourceKind := flags.String("source-kind", "", "source kind filter")
	from := flags.String("from", "", "RFC3339 lower bound")
	to := flags.String("to", "", "RFC3339 upper bound")
	if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
		return err
	}
	scope, err := scopeFlags.scope()
	if err != nil {
		return err
	}
	switch args[0] {
	case "vocabulary":
		result, _, err := client.PlatformGetUsageVocabulary(ctx, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "facts", "export":
		query := platformQuery(page, limit, cursor)
		for key, value := range map[string]string{"fact_type": *factType, "family": *family, "source_kind": *sourceKind, "from": *from, "to": *to} {
			if strings.TrimSpace(value) != "" {
				query.Set(key, value)
			}
		}
		result, _, err := client.PlatformListUsageFacts(ctx, scope, query)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "correct":
		if flags.NArg() != 1 {
			return errors.New("usage: clover-cli usage correct <fact-id> -json '{...}'")
		}
		body, err := parsePlatformBody(bodyFlags, true)
		if err != nil {
			return err
		}
		result, _, err := client.PlatformCorrectUsageFact(ctx, scope, flags.Arg(0), body)
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown usage command %q", args[0])
	}
}

func runPlatformReconciliations(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli usage reconciliations <list|create|get|finish|items>")
	}
	flags := flag.NewFlagSet("usage reconciliations "+args[0], flag.ContinueOnError)
	scopeFlags := addPlatformScopeFlags(flags)
	bodyFlags := addJSONInputFlags(flags)
	page := flags.Int("page", 0, "page number")
	limit := flags.Int("limit", 0, "page size")
	cursor := flags.String("cursor", "", "pagination cursor")
	status := flags.String("status", "", "reconciliation status filter")
	if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
		return err
	}
	scope, err := scopeFlags.scope()
	if err != nil {
		return err
	}
	if args[0] == "list" {
		query := platformQuery(page, limit, cursor)
		if *status != "" {
			query.Set("status", *status)
		}
		result, _, err := client.PlatformListReconciliations(ctx, scope, query)
		if err != nil {
			return err
		}
		return printJSON(result)
	}
	if args[0] == "create" {
		body, err := parsePlatformBody(bodyFlags, true)
		if err != nil {
			return err
		}
		result, _, err := client.PlatformStartReconciliation(ctx, scope, body)
		if err != nil {
			return err
		}
		return printJSON(result)
	}
	if flags.NArg() < 1 {
		return errors.New("reconciliation-id is required")
	}
	id := flags.Arg(0)
	switch args[0] {
	case "get":
		result, _, err := client.PlatformGetReconciliation(ctx, scope, id)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "finish":
		body, err := parsePlatformBody(bodyFlags, true)
		if err != nil {
			return err
		}
		result, _, err := client.PlatformFinishReconciliation(ctx, scope, id, body)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "items":
		if flags.NArg() == 1 {
			result, _, err := client.PlatformListReconciliationItems(ctx, scope, id, platformQuery(nil, limit, cursor))
			if err != nil {
				return err
			}
			return printJSON(result)
		}
		if flags.NArg() != 2 || flags.Arg(1) != "add" {
			return errors.New("usage: clover-cli usage reconciliations items <reconciliation-id> [add -json '{...}']")
		}
		body, err := parsePlatformBody(bodyFlags, true)
		if err != nil {
			return err
		}
		result, _, err := client.PlatformAddReconciliationItem(ctx, scope, id, body)
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown reconciliations command %q", args[0])
	}
}
