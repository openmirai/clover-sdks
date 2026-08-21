package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"strings"
)

func runRouting(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli routing <policy|capabilities|pools|audit>")
	}
	switch args[0] {
	case "policy":
		return runRoutingPolicy(ctx, client, args[1:])
	case "capabilities":
		flags := flag.NewFlagSet("routing capabilities", flag.ContinueOnError)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return errors.New("usage: clover-cli routing capabilities")
		}
		result, _, err := client.ListRoutingCapabilities(ctx)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "pools":
		return runRoutingPools(ctx, client, args[1:])
	case "audit":
		return runRoutingAudit(ctx, client, args[1:])
	default:
		return fmt.Errorf("unknown routing command %q", args[0])
	}
}

func runRoutingPolicy(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli routing policy <get|put>")
	}
	switch args[0] {
	case "get":
		flags := flag.NewFlagSet("routing policy get", flag.ContinueOnError)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return errors.New("usage: clover-cli routing policy get")
		}
		result, _, err := client.GetRoutingPolicy(ctx)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "put", "update":
		flags := flag.NewFlagSet("routing policy put", flag.ContinueOnError)
		bodyFlags := addJSONInputFlags(flags)
		key := flags.String("idempotency-key", "", "required idempotency key")
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return errors.New("usage: clover-cli routing policy put -json '{...}' -idempotency-key <key>")
		}
		if err := requiredKey(*key); err != nil {
			return err
		}
		body, err := bodyFlags.read(true, DefaultMaxRequestBodyBytes)
		if err != nil {
			return err
		}
		result, _, err := client.PutRoutingPolicy(ctx, body, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown routing policy command %q", args[0])
	}
}

func runRoutingPools(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli routing pools <list|create|get|command|warmup|ip-command>")
	}
	switch args[0] {
	case "list":
		flags := flag.NewFlagSet("routing pools list", flag.ContinueOnError)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return errors.New("usage: clover-cli routing pools list")
		}
		result, _, err := client.ListRoutingPools(ctx)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "create":
		flags := flag.NewFlagSet("routing pools create", flag.ContinueOnError)
		bodyFlags := addJSONInputFlags(flags)
		key := flags.String("idempotency-key", "", "required idempotency key")
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return errors.New("usage: clover-cli routing pools create -json '{...}' -idempotency-key <key>")
		}
		if err := requiredKey(*key); err != nil {
			return err
		}
		body, err := bodyFlags.read(true, DefaultMaxRequestBodyBytes)
		if err != nil {
			return err
		}
		result, _, err := client.CreateRoutingPool(ctx, body, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "get":
		flags := flag.NewFlagSet("routing pools get", flag.ContinueOnError)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli routing pools get <pool-id>"); err != nil {
			return err
		}
		result, _, err := client.GetRoutingPool(ctx, flags.Arg(0))
		if err != nil {
			return err
		}
		return printJSON(result)
	case "command", "warmup":
		return runRoutingPoolCommand(ctx, client, args[0], args[1:])
	case "ip-command":
		flags := flag.NewFlagSet("routing pools ip-command", flag.ContinueOnError)
		bodyFlags := addJSONInputFlags(flags)
		key := flags.String("idempotency-key", "", "required idempotency key")
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 2, "usage: clover-cli routing pools ip-command <pool-id> <ip-id> -json '{...}' -idempotency-key <key>"); err != nil {
			return err
		}
		if err := requiredKey(*key); err != nil {
			return err
		}
		body, err := bodyFlags.read(true, DefaultMaxRequestBodyBytes)
		if err != nil {
			return err
		}
		result, _, err := client.ApplyRoutingIPCommand(ctx, flags.Arg(0), flags.Arg(1), body, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown routing pools command %q", args[0])
	}
}

func runRoutingPoolCommand(ctx context.Context, client *Client, command string, args []string) error {
	flags := flag.NewFlagSet("routing pools "+command, flag.ContinueOnError)
	bodyFlags := addJSONInputFlags(flags)
	key := flags.String("idempotency-key", "", "required idempotency key")
	if err := flags.Parse(flagArgsFirst(args)); err != nil {
		return err
	}
	if err := requireArgs(flags, 1, "usage: clover-cli routing pools command <pool-id> -json '{\"action\":\"...\"}' -idempotency-key <key>"); err != nil {
		return err
	}
	if err := requiredKey(*key); err != nil {
		return err
	}
	body, err := bodyFlags.read(command != "warmup", DefaultMaxRequestBodyBytes)
	if err != nil {
		return err
	}
	if command == "warmup" && strings.TrimSpace(string(body)) == "{}" {
		body = json.RawMessage(`{"action":"start_warmup"}`)
	}
	result, _, err := client.ApplyRoutingPoolCommand(ctx, flags.Arg(0), body, *key)
	if err != nil {
		return err
	}
	return printJSON(result)
}

func runRoutingAudit(ctx context.Context, client *Client, args []string) error {
	flags := flag.NewFlagSet("routing audit", flag.ContinueOnError)
	entityType := flags.String("entity-type", "", "entity type filter")
	entityID := flags.String("entity-id", "", "entity UUID filter")
	if err := flags.Parse(flagArgsFirst(args)); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("usage: clover-cli routing audit [-entity-type type] [-entity-id uuid]")
	}
	query := url.Values{}
	if *entityType != "" {
		query.Set("entityType", *entityType)
	}
	if *entityID != "" {
		query.Set("entityId", *entityID)
	}
	result, _, err := client.ListRoutingAudit(ctx, query)
	if err != nil {
		return err
	}
	return printJSON(result)
}

func runDeliveryPolicies(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli delivery-policies <list|create|update>")
	}
	switch args[0] {
	case "list":
		flags := flag.NewFlagSet("delivery-policies list", flag.ContinueOnError)
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		scope, err := scopeFlags.scope(false)
		if err != nil {
			return err
		}
		result, _, err := client.ListDeliveryPolicies(ctx, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "create", "update":
		flags := flag.NewFlagSet("delivery-policies mutation", flag.ContinueOnError)
		bodyFlags := addJSONInputFlags(flags)
		key := flags.String("idempotency-key", "", "required idempotency key")
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requiredKey(*key); err != nil {
			return err
		}
		if args[0] == "update" {
			if err := requireArgs(flags, 1, "usage: clover-cli delivery-policies update <policy-id> -json '{...}'"); err != nil {
				return err
			}
		}
		body, err := bodyFlags.read(true, DefaultMaxRequestBodyBytes)
		if err != nil {
			return err
		}
		scope, err := scopeFlags.scope(false)
		if err != nil {
			return err
		}
		body, err = mergeBodyScope(body, scope)
		if err != nil {
			return err
		}
		var result json.RawMessage
		if args[0] == "create" {
			result, _, err = client.CreateDeliveryPolicy(ctx, body, *key, scope)
		} else {
			result, _, err = client.UpdateDeliveryPolicy(ctx, flags.Arg(0), body, *key, scope)
		}
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown delivery-policies command %q", args[0])
	}
}

func runDeliveryRoutes(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli delivery-routes <list|create>")
	}
	switch args[0] {
	case "list":
		flags := flag.NewFlagSet("delivery-routes list", flag.ContinueOnError)
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		scope, err := scopeFlags.scope(false)
		if err != nil {
			return err
		}
		result, _, err := client.ListDeliveryRoutes(ctx, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "create":
		flags := flag.NewFlagSet("delivery-routes create", flag.ContinueOnError)
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
		body, err = mergeBodyScope(body, scope)
		if err != nil {
			return err
		}
		result, _, err := client.CreateDeliveryRoute(ctx, body, *key, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown delivery-routes command %q", args[0])
	}
}

func runSMTPCredentials(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli smtp-credentials <list|create|get|revoke>")
	}
	switch args[0] {
	case "list":
		flags := flag.NewFlagSet("smtp-credentials list", flag.ContinueOnError)
		page := addPageFlags(flags, false)
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		scope, err := scopeFlags.scope(false)
		if err != nil {
			return err
		}
		result, _, err := client.ListSMTPCredentials(ctx, page.query(), scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "create":
		flags := flag.NewFlagSet("smtp-credentials create", flag.ContinueOnError)
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
		result, _, err := client.CreateSMTPCredential(ctx, body, *key, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "get", "revoke":
		flags := flag.NewFlagSet("smtp-credentials resource", flag.ContinueOnError)
		key := flags.String("idempotency-key", "", "required idempotency key for revoke")
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli smtp-credentials get|revoke <id>"); err != nil {
			return err
		}
		scope, err := scopeFlags.scope(false)
		if err != nil {
			return err
		}
		var result json.RawMessage
		if args[0] == "get" {
			result, _, err = client.GetSMTPCredential(ctx, flags.Arg(0), scope)
		} else {
			if err = requiredKey(*key); err == nil {
				result, _, err = client.RevokeSMTPCredential(ctx, flags.Arg(0), *key, scope)
			}
		}
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown smtp-credentials command %q", args[0])
	}
}
