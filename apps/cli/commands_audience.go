package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/url"
)

func runContacts(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli contacts <list|create|get|transition>")
	}
	switch args[0] {
	case "list":
		flags := flag.NewFlagSet("contacts list", flag.ContinueOnError)
		page := addPageFlags(flags, true)
		status := flags.String("status", "", "contact status filter")
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		query := page.query()
		if *status != "" {
			query.Set("status", *status)
		}
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		result, _, err := client.ListContacts(ctx, query, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "create":
		flags := flag.NewFlagSet("contacts create", flag.ContinueOnError)
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
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		body, err = mergeBodyScope(body, scope)
		if err != nil {
			return err
		}
		result, _, err := client.CreateContact(ctx, body, *key, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "get":
		flags := flag.NewFlagSet("contacts get", flag.ContinueOnError)
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli contacts get <contact-id>"); err != nil {
			return err
		}
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		result, _, err := client.GetContact(ctx, flags.Arg(0), scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "transition", "update":
		flags := flag.NewFlagSet("contacts transition", flag.ContinueOnError)
		bodyFlags := addJSONInputFlags(flags)
		key := flags.String("idempotency-key", "", "required idempotency key")
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli contacts transition <contact-id> -json '{\"event\":\"...\"}'"); err != nil {
			return err
		}
		if err := requiredKey(*key); err != nil {
			return err
		}
		body, err := bodyFlags.read(true, DefaultMaxRequestBodyBytes)
		if err != nil {
			return err
		}
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		result, _, err := client.TransitionContact(ctx, flags.Arg(0), body, *key, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown contacts command %q", args[0])
	}
}

func runSegments(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli segments <list|create|get|archive|evaluate>")
	}
	switch args[0] {
	case "list":
		flags := flag.NewFlagSet("segments list", flag.ContinueOnError)
		page := addPageFlags(flags, true)
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		result, _, err := client.ListSegments(ctx, page.query(), scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "create":
		flags := flag.NewFlagSet("segments create", flag.ContinueOnError)
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
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		body, err = mergeBodyScope(body, scope)
		if err != nil {
			return err
		}
		result, _, err := client.CreateSegment(ctx, body, *key, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "get":
		flags := flag.NewFlagSet("segments get", flag.ContinueOnError)
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli segments get <segment-id>"); err != nil {
			return err
		}
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		result, _, err := client.GetSegment(ctx, flags.Arg(0), scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "archive":
		flags := flag.NewFlagSet("segments archive", flag.ContinueOnError)
		key := flags.String("idempotency-key", "", "required idempotency key")
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli segments archive <segment-id>"); err != nil {
			return err
		}
		if err := requiredKey(*key); err != nil {
			return err
		}
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		result, _, err := client.ArchiveSegment(ctx, flags.Arg(0), *key, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "evaluate":
		flags := flag.NewFlagSet("segments evaluate", flag.ContinueOnError)
		key := flags.String("idempotency-key", "", "optional idempotency key")
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli segments evaluate <segment-id>"); err != nil {
			return err
		}
		if err := optionalKey(*key); err != nil {
			return err
		}
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		result, _, err := client.EvaluateSegment(ctx, flags.Arg(0), *key, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown segments command %q", args[0])
	}
}

func runBroadcasts(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli broadcasts <list|create|get|schedule|cancel>")
	}
	switch args[0] {
	case "list":
		flags := flag.NewFlagSet("broadcasts list", flag.ContinueOnError)
		page := addPageFlags(flags, true)
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		result, _, err := client.ListBroadcasts(ctx, page.query(), scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "create":
		flags := flag.NewFlagSet("broadcasts create", flag.ContinueOnError)
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
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		body, err = mergeBodyScope(body, scope)
		if err != nil {
			return err
		}
		result, _, err := client.CreateBroadcast(ctx, body, *key, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "get":
		flags := flag.NewFlagSet("broadcasts get", flag.ContinueOnError)
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli broadcasts get <broadcast-id>"); err != nil {
			return err
		}
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		result, _, err := client.GetBroadcast(ctx, flags.Arg(0), scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "schedule":
		flags := flag.NewFlagSet("broadcasts schedule", flag.ContinueOnError)
		bodyFlags := addJSONInputFlags(flags)
		key := flags.String("idempotency-key", "", "required idempotency key")
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli broadcasts schedule <broadcast-id> -json '{\"scheduled_at\":\"...\"}'"); err != nil {
			return err
		}
		if err := requiredKey(*key); err != nil {
			return err
		}
		body, err := bodyFlags.read(true, DefaultMaxRequestBodyBytes)
		if err != nil {
			return err
		}
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		result, _, err := client.ScheduleBroadcast(ctx, flags.Arg(0), body, *key, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "cancel":
		flags := flag.NewFlagSet("broadcasts cancel", flag.ContinueOnError)
		key := flags.String("idempotency-key", "", "required idempotency key")
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli broadcasts cancel <broadcast-id>"); err != nil {
			return err
		}
		if err := requiredKey(*key); err != nil {
			return err
		}
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		result, _, err := client.CancelBroadcast(ctx, flags.Arg(0), *key, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown broadcasts command %q", args[0])
	}
}

func runAutomations(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli automations <list|create|update|get|activate|pause|run|run-get|event>")
	}
	switch args[0] {
	case "list":
		flags := flag.NewFlagSet("automations list", flag.ContinueOnError)
		page := addPageFlags(flags, true)
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		result, _, err := client.ListAutomations(ctx, page.query(), scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "create":
		flags := flag.NewFlagSet("automations create", flag.ContinueOnError)
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
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		body, err = mergeBodyScope(body, scope)
		if err != nil {
			return err
		}
		result, _, err := client.CreateAutomation(ctx, body, *key, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "update":
		flags := flag.NewFlagSet("automations update", flag.ContinueOnError)
		bodyFlags := addJSONInputFlags(flags)
		key := flags.String("idempotency-key", "", "required idempotency key")
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli automations update <automation-id> -json '{...}'"); err != nil {
			return err
		}
		if err := requiredKey(*key); err != nil {
			return err
		}
		body, err := bodyFlags.read(true, DefaultMaxRequestBodyBytes)
		if err != nil {
			return err
		}
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		body, err = mergeBodyScope(body, scope)
		if err != nil {
			return err
		}
		result, _, err := client.UpdateAutomation(ctx, flags.Arg(0), body, *key, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "get":
		flags := flag.NewFlagSet("automations get", flag.ContinueOnError)
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli automations get <automation-id>"); err != nil {
			return err
		}
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		result, _, err := client.GetAutomation(ctx, flags.Arg(0), scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "activate", "pause":
		flags := flag.NewFlagSet("automations transition", flag.ContinueOnError)
		key := flags.String("idempotency-key", "", "required idempotency key")
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli automations activate|pause <automation-id>"); err != nil {
			return err
		}
		if err := requiredKey(*key); err != nil {
			return err
		}
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		result, _, err := client.TransitionAutomation(ctx, flags.Arg(0), args[0], *key, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "run":
		flags := flag.NewFlagSet("automations run", flag.ContinueOnError)
		bodyFlags := addJSONInputFlags(flags)
		key := flags.String("idempotency-key", "", "required idempotency key")
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli automations run <automation-id> -json '{\"contact_id\":\"...\"}'"); err != nil {
			return err
		}
		if err := requiredKey(*key); err != nil {
			return err
		}
		body, err := bodyFlags.read(true, DefaultMaxRequestBodyBytes)
		if err != nil {
			return err
		}
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		result, _, err := client.StartAutomationRun(ctx, flags.Arg(0), body, *key, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "run-get":
		flags := flag.NewFlagSet("automations run-get", flag.ContinueOnError)
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 2, "usage: clover-cli automations run-get <automation-id> <run-id>"); err != nil {
			return err
		}
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		result, _, err := client.GetAutomationRun(ctx, flags.Arg(0), flags.Arg(1), scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "event":
		flags := flag.NewFlagSet("automations event", flag.ContinueOnError)
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
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		result, _, err := client.IngestAutomationEvent(ctx, body, *key, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown automations command %q", args[0])
	}
}

func runEmailReliability(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli emails <trace|replay>")
	}
	switch args[0] {
	case "trace":
		flags := flag.NewFlagSet("emails trace", flag.ContinueOnError)
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli emails trace <email-id>"); err != nil {
			return err
		}
		scope, err := scopeFlags.scope(false)
		if err != nil {
			return err
		}
		result, _, err := client.GetEmailTrace(ctx, flags.Arg(0), scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "replay":
		flags := flag.NewFlagSet("emails replay", flag.ContinueOnError)
		bodyFlags := addJSONInputFlags(flags)
		key := flags.String("idempotency-key", "", "required idempotency key")
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli emails replay <email-id>"); err != nil {
			return err
		}
		if err := requiredKey(*key); err != nil {
			return err
		}
		body, err := bodyFlags.read(false, DefaultMaxRequestBodyBytes)
		if err != nil {
			return err
		}
		scope, err := scopeFlags.scope(false)
		if err != nil {
			return err
		}
		result, _, err := client.ReplayEmail(ctx, flags.Arg(0), body, *key, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown emails command %q", args[0])
	}
}

func runAudit(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli audit <list|get|append|holds>")
	}
	if args[0] == "holds" {
		return runAuditHolds(ctx, client, args[1:])
	}
	switch args[0] {
	case "list":
		flags := flag.NewFlagSet("audit list", flag.ContinueOnError)
		page := addPageFlags(flags, true)
		resourceType := flags.String("resource-type", "", "resource type filter")
		resourceID := flags.String("resource-id", "", "resource UUID filter")
		outcome := flags.String("outcome", "", "outcome filter")
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		query := page.query()
		if *resourceType != "" {
			query.Set("resource_type", *resourceType)
		}
		if *resourceID != "" {
			query.Set("resource_id", *resourceID)
		}
		if *outcome != "" {
			query.Set("outcome", *outcome)
		}
		scope, err := scopeFlags.scope(false)
		if err != nil {
			return err
		}
		result, _, err := client.ListAuditEvents(ctx, query, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "append", "create":
		flags := flag.NewFlagSet("audit append", flag.ContinueOnError)
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
		result, _, err := client.AppendAuditEvent(ctx, body, *key, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "get":
		flags := flag.NewFlagSet("audit get", flag.ContinueOnError)
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli audit get <event-id>"); err != nil {
			return err
		}
		scope, err := scopeFlags.scope(false)
		if err != nil {
			return err
		}
		result, _, err := client.GetAuditEvent(ctx, flags.Arg(0), scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown audit command %q", args[0])
	}
}

func runAuditHolds(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli audit holds <list|create|release>")
	}
	switch args[0] {
	case "list":
		flags := flag.NewFlagSet("audit holds list", flag.ContinueOnError)
		active := flags.String("active", "", "active filter (true or false)")
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		query := url.Values{}
		if err := setBooleanQuery(query, "active", *active); err != nil {
			return err
		}
		scope, err := scopeFlags.scope(false)
		if err != nil {
			return err
		}
		result, _, err := client.ListAuditHolds(ctx, query, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "create":
		flags := flag.NewFlagSet("audit holds create", flag.ContinueOnError)
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
		result, _, err := client.CreateAuditHold(ctx, body, *key, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "release":
		flags := flag.NewFlagSet("audit holds release", flag.ContinueOnError)
		key := flags.String("idempotency-key", "", "required idempotency key")
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli audit holds release <hold-id>"); err != nil {
			return err
		}
		if err := requiredKey(*key); err != nil {
			return err
		}
		scope, err := scopeFlags.scope(false)
		if err != nil {
			return err
		}
		result, _, err := client.ReleaseAuditHold(ctx, flags.Arg(0), *key, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown audit holds command %q", args[0])
	}
}
