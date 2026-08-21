package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type jsonInputFlags struct {
	raw  *string
	file *string
}

func addJSONInputFlags(flags *flag.FlagSet) jsonInputFlags {
	return jsonInputFlags{
		raw:  flags.String("json", "", "JSON request body"),
		file: flags.String("file", "", "read the JSON request body from a file (or - for stdin)"),
	}
}

func (input jsonInputFlags) read(required bool, max int) (json.RawMessage, error) {
	if strings.TrimSpace(*input.raw) != "" && strings.TrimSpace(*input.file) != "" {
		return nil, errors.New("use only one of -json or -file")
	}
	if strings.TrimSpace(*input.raw) == "" && strings.TrimSpace(*input.file) == "" {
		if required {
			return nil, errors.New("one of -json or -file is required")
		}
		return json.RawMessage("{}"), nil
	}
	var data []byte
	if strings.TrimSpace(*input.file) != "" {
		var reader io.Reader
		if strings.TrimSpace(*input.file) == "-" {
			reader = os.Stdin
		} else {
			file, err := os.Open(*input.file)
			if err != nil {
				return nil, fmt.Errorf("open JSON input: %w", err)
			}
			defer file.Close()
			reader = file
		}
		var err error
		data, err = io.ReadAll(io.LimitReader(reader, int64(max)+1))
		if err != nil {
			return nil, fmt.Errorf("read JSON input: %w", err)
		}
	} else {
		data = []byte(*input.raw)
	}
	if len(data) > max {
		return nil, errors.New("JSON request body exceeds the configured limit")
	}
	if !json.Valid(data) {
		return nil, errors.New("JSON request body is invalid")
	}
	return json.RawMessage(data), nil
}

type scopeFlags struct {
	organizationID *string
	projectID      *string
	environment    *string
	tenantID       *string
}

func addScopeFlags(flags *flag.FlagSet) scopeFlags {
	return scopeFlags{
		organizationID: flags.String("organization-id", os.Getenv("CLOVER_ORGANIZATION_ID"), "organization scope UUID (required for request-body scopes)"),
		projectID:      flags.String("project-id", os.Getenv("CLOVER_PROJECT_ID"), "project scope UUID"),
		environment:    flags.String("environment", envOrDefault(), "scope environment: development, preview, staging, or production"),
		tenantID:       flags.String("tenant-id", os.Getenv("CLOVER_TENANT_ID"), "tenant scope UUID"),
	}
}

func envOrDefault() string {
	if value := strings.TrimSpace(os.Getenv("CLOVER_ENVIRONMENT")); value != "" {
		return value
	}
	return ""
}

func (input scopeFlags) scope(requireTenant bool) (Scope, error) {
	scope := Scope{OrganizationID: strings.TrimSpace(*input.organizationID), ProjectID: strings.TrimSpace(*input.projectID), Environment: strings.TrimSpace(*input.environment), TenantID: strings.TrimSpace(*input.tenantID)}
	if err := scope.validate(requireTenant); err != nil {
		return Scope{}, err
	}
	return scope, nil
}

func (input scopeFlags) anySet() bool {
	return strings.TrimSpace(*input.organizationID) != "" || strings.TrimSpace(*input.projectID) != "" || strings.TrimSpace(*input.environment) != "" || strings.TrimSpace(*input.tenantID) != ""
}

func mergeBodyScope(body json.RawMessage, scope Scope) (json.RawMessage, error) {
	var object map[string]any
	if err := json.Unmarshal(body, &object); err != nil {
		return nil, errors.New("JSON request body must be an object")
	}
	if object == nil {
		return nil, errors.New("JSON request body must be an object")
	}
	scopeObject := map[string]any{"project_id": scope.ProjectID, "environment": scope.Environment}
	if strings.TrimSpace(scope.OrganizationID) != "" {
		scopeObject["organization_id"] = scope.OrganizationID
	} else if existing, ok := object["scope"].(map[string]any); ok {
		if organizationID, ok := existing["organization_id"].(string); ok && strings.TrimSpace(organizationID) != "" {
			scopeObject["organization_id"] = strings.TrimSpace(organizationID)
		}
	}
	if _, ok := scopeObject["organization_id"]; !ok {
		return nil, errors.New("organization-id is required for request-body scopes")
	}
	if strings.TrimSpace(scope.TenantID) != "" {
		scopeObject["tenant_id"] = scope.TenantID
	}
	object["scope"] = scopeObject
	merged, err := json.Marshal(object)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(merged), nil
}

func requireArgs(flags *flag.FlagSet, count int, usage string) error {
	if flags.NArg() != count {
		return errors.New(usage)
	}
	return nil
}

// The standard flag package stops parsing at the first positional argument.
// Keep the CLI ergonomic by accepting both `command ID -flag value` and
// `command -flag value ID` forms while preserving each flag's value.
func flagArgsFirst(args []string) []string {
	flags := make([]string, 0, len(args))
	positional := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--" {
			positional = append(positional, args[index+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positional = append(positional, arg)
			continue
		}
		flags = append(flags, arg)
		if strings.Contains(arg, "=") || index+1 >= len(args) || (strings.HasPrefix(args[index+1], "-") && args[index+1] != "-") {
			continue
		}
		flags = append(flags, args[index+1])
		index++
	}
	return append(flags, positional...)
}

func requiredKey(value string) error {
	if !idempotencyKeyPattern.MatchString(strings.TrimSpace(value)) {
		return errors.New("idempotency-key must match ^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$")
	}
	return nil
}

func optionalKey(value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return requiredKey(value)
}

func runDomains(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli domains <list|get|create|configure|verify|delete|dns-records|provision>")
	}
	switch args[0] {
	case "list":
		flags := flag.NewFlagSet("domains list", flag.ContinueOnError)
		page := flags.Int("page", 0, "page number")
		limit := flags.Int("limit", 0, "page size")
		status := flags.String("status", "", "domain status filter")
		provider := flags.String("provider", "", "provider filter")
		sending := flags.String("sending-enabled", "", "sending enabled filter (true or false)")
		receiving := flags.String("receiving-enabled", "", "receiving enabled filter (true or false)")
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return errors.New("usage: clover-cli domains list [-page N] [-limit N]")
		}
		query := url.Values{}
		if *page > 0 {
			query.Set("page", strconv.Itoa(*page))
		}
		if *limit > 0 {
			query.Set("limit", strconv.Itoa(*limit))
		}
		if *status != "" {
			query.Set("status", *status)
		}
		if *provider != "" {
			query.Set("provider", *provider)
		}
		if err := setBooleanQuery(query, "sendingEnabled", *sending); err != nil {
			return err
		}
		if err := setBooleanQuery(query, "receivingEnabled", *receiving); err != nil {
			return err
		}
		result, _, err := client.ListDomains(ctx, query)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "get":
		flags := flag.NewFlagSet("domains get", flag.ContinueOnError)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli domains get <domain-id>"); err != nil {
			return err
		}
		result, _, err := client.GetDomain(ctx, flags.Arg(0))
		if err != nil {
			return err
		}
		return printJSON(result)
	case "create":
		flags := flag.NewFlagSet("domains create", flag.ContinueOnError)
		bodyFlags := addJSONInputFlags(flags)
		key := flags.String("idempotency-key", "", "optional idempotency key")
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := optionalKey(*key); err != nil {
			return err
		}
		body, err := bodyFlags.read(true, DefaultMaxRequestBodyBytes)
		if err != nil {
			return err
		}
		result, _, err := client.CreateDomain(ctx, body, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "configure":
		flags := flag.NewFlagSet("domains configure", flag.ContinueOnError)
		bodyFlags := addJSONInputFlags(flags)
		key := flags.String("idempotency-key", "", "optional idempotency key")
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli domains configure <domain-id> -json '<body>' [-idempotency-key <key>]"); err != nil {
			return err
		}
		if err := optionalKey(*key); err != nil {
			return err
		}
		body, err := bodyFlags.read(true, DefaultMaxRequestBodyBytes)
		if err != nil {
			return err
		}
		result, _, err := client.ConfigureDomain(ctx, flags.Arg(0), body, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "verify":
		flags := flag.NewFlagSet("domains verify", flag.ContinueOnError)
		bodyFlags := addJSONInputFlags(flags)
		key := flags.String("idempotency-key", "", "optional idempotency key")
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli domains verify <domain-id> [-json '{}'] [-idempotency-key <key>]"); err != nil {
			return err
		}
		if err := optionalKey(*key); err != nil {
			return err
		}
		body, err := bodyFlags.read(false, DefaultMaxRequestBodyBytes)
		if err != nil {
			return err
		}
		result, _, err := client.VerifyDomain(ctx, flags.Arg(0), body, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "delete":
		flags := flag.NewFlagSet("domains delete", flag.ContinueOnError)
		key := flags.String("idempotency-key", "", "optional idempotency key")
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli domains delete <domain-id>"); err != nil {
			return err
		}
		if err := optionalKey(*key); err != nil {
			return err
		}
		result, _, err := client.DeleteDomain(ctx, flags.Arg(0), *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "provision":
		flags := flag.NewFlagSet("domains provision", flag.ContinueOnError)
		bodyFlags := addJSONInputFlags(flags)
		key := flags.String("idempotency-key", "", "required idempotency key")
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli domains provision <domain-id> [-json '{\"force\":true}'] -idempotency-key <key>"); err != nil {
			return err
		}
		if err := requiredKey(*key); err != nil {
			return err
		}
		body, err := bodyFlags.read(false, DefaultMaxRequestBodyBytes)
		if err != nil {
			return err
		}
		result, _, err := client.ProvisionDomainDNS(ctx, flags.Arg(0), body, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "dns-records":
		flags := flag.NewFlagSet("domains dns-records", flag.ContinueOnError)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli domains dns-records <domain-id>"); err != nil {
			return err
		}
		result, _, err := client.ListDomainDNSRecords(ctx, flags.Arg(0))
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown domains command %q", args[0])
	}
}

func setBooleanQuery(query url.Values, name, raw string) error {
	if raw == "" {
		return nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fmt.Errorf("%s must be true or false", name)
	}
	query.Set(name, strconv.FormatBool(value))
	return nil
}

func runTemplates(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli templates <list|get|create|update|versions|publish>")
	}
	switch args[0] {
	case "list":
		flags := flag.NewFlagSet("templates list", flag.ContinueOnError)
		page := flags.Int("page", 0, "page number")
		limit := flags.Int("limit", 0, "page size")
		cursor := flags.String("cursor", "", "pagination cursor")
		status := flags.String("status", "", "template status filter")
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		query := url.Values{}
		if *page > 0 {
			query.Set("page", strconv.Itoa(*page))
		}
		if *limit > 0 {
			query.Set("limit", strconv.Itoa(*limit))
		}
		if *cursor != "" {
			query.Set("cursor", *cursor)
		}
		if *status != "" {
			query.Set("status", *status)
		}
		result, _, err := client.ListTemplates(ctx, query, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "get":
		flags := flag.NewFlagSet("templates get", flag.ContinueOnError)
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli templates get <template-id> -project-id <id> -environment <env> -tenant-id <id>"); err != nil {
			return err
		}
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		result, _, err := client.GetTemplate(ctx, flags.Arg(0), scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "create":
		flags := flag.NewFlagSet("templates create", flag.ContinueOnError)
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
		result, _, err := client.CreateTemplateScoped(ctx, body, *key, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "update", "transition":
		flags := flag.NewFlagSet("templates update", flag.ContinueOnError)
		bodyFlags := addJSONInputFlags(flags)
		key := flags.String("idempotency-key", "", "required idempotency key")
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli templates update <template-id> -json '{\"event\":\"...\"}' -idempotency-key <key>"); err != nil {
			return err
		}
		if err := requiredKey(*key); err != nil {
			return err
		}
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		body, err := bodyFlags.read(true, DefaultMaxRequestBodyBytes)
		if err != nil {
			return err
		}
		result, _, err := client.TransitionTemplate(ctx, flags.Arg(0), body, *key, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "versions":
		return runTemplateVersions(ctx, client, args[1:])
	case "publish":
		return runTemplatePublish(ctx, client, args[1:])
	default:
		return fmt.Errorf("unknown templates command %q", args[0])
	}
}

func runTemplateVersions(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli templates versions <list|get|compare|create|publish|rollback>")
	}
	switch args[0] {
	case "list":
		flags := flag.NewFlagSet("templates versions list", flag.ContinueOnError)
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli templates versions list <template-id>"); err != nil {
			return err
		}
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		result, _, err := client.ListTemplateVersions(ctx, flags.Arg(0), scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "get":
		flags := flag.NewFlagSet("templates versions get", flag.ContinueOnError)
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 2, "usage: clover-cli templates versions get <template-id> <version-ref>"); err != nil {
			return err
		}
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		result, _, err := client.GetTemplateVersion(ctx, flags.Arg(0), flags.Arg(1), scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "compare":
		flags := flag.NewFlagSet("templates versions compare", flag.ContinueOnError)
		from := flags.String("from", "", "source version number, UUID, or latest")
		to := flags.String("to", "", "target version number, UUID, or latest")
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli templates versions compare <template-id> -from <version> -to <version>"); err != nil {
			return err
		}
		if strings.TrimSpace(*from) == "" || strings.TrimSpace(*to) == "" {
			return errors.New("-from and -to are required")
		}
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		result, _, err := client.CompareTemplateVersions(ctx, flags.Arg(0), *from, *to, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "create":
		flags := flag.NewFlagSet("templates versions create", flag.ContinueOnError)
		bodyFlags := addJSONInputFlags(flags)
		key := flags.String("idempotency-key", "", "required idempotency key")
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli templates versions create <template-id>"); err != nil {
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
		result, _, err := client.CreateTemplateVersionScoped(ctx, flags.Arg(0), body, *key, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "publish":
		return runTemplatePublish(ctx, client, args[1:])
	case "rollback":
		flags := flag.NewFlagSet("templates versions rollback", flag.ContinueOnError)
		key := flags.String("idempotency-key", "", "required idempotency key")
		scopeFlags := addScopeFlags(flags)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 2, "usage: clover-cli templates versions rollback <template-id> <version-id>"); err != nil {
			return err
		}
		if err := requiredKey(*key); err != nil {
			return err
		}
		scope, err := scopeFlags.scope(true)
		if err != nil {
			return err
		}
		result, _, err := client.RollbackTemplateVersion(ctx, flags.Arg(0), flags.Arg(1), *key, scope)
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown template versions command %q", args[0])
	}
}

func runTemplatePublish(ctx context.Context, client *Client, args []string) error {
	flags := flag.NewFlagSet("templates publish", flag.ContinueOnError)
	key := flags.String("idempotency-key", "", "required idempotency key")
	scopeFlags := addScopeFlags(flags)
	if err := flags.Parse(flagArgsFirst(args)); err != nil {
		return err
	}
	if err := requireArgs(flags, 2, "usage: clover-cli templates publish <template-id> <version-id> -project-id <id> -environment <env> -tenant-id <id> -idempotency-key <key>"); err != nil {
		return err
	}
	if err := requiredKey(*key); err != nil {
		return err
	}
	scope, err := scopeFlags.scope(true)
	if err != nil {
		return err
	}
	result, _, err := client.PublishTemplateVersion(ctx, flags.Arg(0), flags.Arg(1), *key, scope)
	if err != nil {
		return err
	}
	return printJSON(result)
}

func runWebhooks(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli webhooks <list|get|create|update|delete|rotate|deliveries>")
	}
	if args[0] == "deliveries" {
		return runWebhookDeliveries(ctx, client, args[1:])
	}
	switch args[0] {
	case "list":
		flags := flag.NewFlagSet("webhooks list", flag.ContinueOnError)
		cursor := flags.String("cursor", "", "pagination cursor")
		limit := flags.Int("limit", 0, "page size")
		enabled := flags.String("enabled", "", "enabled filter (true or false)")
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return errors.New("usage: clover-cli webhooks list [-cursor cursor] [-limit N]")
		}
		query := url.Values{}
		if *cursor != "" {
			query.Set("cursor", *cursor)
		}
		if *limit > 0 {
			query.Set("limit", strconv.Itoa(*limit))
		}
		if err := setBooleanQuery(query, "enabled", *enabled); err != nil {
			return err
		}
		result, _, err := client.ListWebhooks(ctx, query)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "get":
		flags := flag.NewFlagSet("webhooks get", flag.ContinueOnError)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli webhooks get <webhook-id>"); err != nil {
			return err
		}
		result, _, err := client.GetWebhook(ctx, flags.Arg(0))
		if err != nil {
			return err
		}
		return printJSON(result)
	case "create":
		flags := flag.NewFlagSet("webhooks create", flag.ContinueOnError)
		bodyFlags := addJSONInputFlags(flags)
		key := flags.String("idempotency-key", "", "optional idempotency key")
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := optionalKey(*key); err != nil {
			return err
		}
		body, err := bodyFlags.read(true, DefaultMaxRequestBodyBytes)
		if err != nil {
			return err
		}
		result, _, err := client.CreateWebhook(ctx, body, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "update":
		flags := flag.NewFlagSet("webhooks update", flag.ContinueOnError)
		bodyFlags := addJSONInputFlags(flags)
		key := flags.String("idempotency-key", "", "optional idempotency key")
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli webhooks update <webhook-id> -json '<body>' [-idempotency-key <key>]"); err != nil {
			return err
		}
		if err := optionalKey(*key); err != nil {
			return err
		}
		body, err := bodyFlags.read(true, DefaultMaxRequestBodyBytes)
		if err != nil {
			return err
		}
		result, _, err := client.UpdateWebhook(ctx, flags.Arg(0), body, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "delete":
		flags := flag.NewFlagSet("webhooks delete", flag.ContinueOnError)
		key := flags.String("idempotency-key", "", "optional idempotency key")
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli webhooks delete <webhook-id> [-idempotency-key <key>]"); err != nil {
			return err
		}
		if err := optionalKey(*key); err != nil {
			return err
		}
		result, _, err := client.DeleteWebhook(ctx, flags.Arg(0), *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "rotate":
		flags := flag.NewFlagSet("webhooks rotate", flag.ContinueOnError)
		bodyFlags := addJSONInputFlags(flags)
		key := flags.String("idempotency-key", "", "optional idempotency key")
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli webhooks rotate <webhook-id> [-json '{\"overlap_seconds\":300}'] [-idempotency-key <key>]"); err != nil {
			return err
		}
		if err := optionalKey(*key); err != nil {
			return err
		}
		body, err := bodyFlags.read(false, DefaultMaxRequestBodyBytes)
		if err != nil {
			return err
		}
		result, _, err := client.RotateWebhookSecret(ctx, flags.Arg(0), body, *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown webhooks command %q", args[0])
	}
}

func runWebhookDeliveries(ctx context.Context, client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clover-cli webhooks deliveries <list|get|replay>")
	}
	switch args[0] {
	case "list":
		flags := flag.NewFlagSet("webhooks deliveries list", flag.ContinueOnError)
		cursor := flags.String("cursor", "", "pagination cursor")
		limit := flags.Int("limit", 0, "page size")
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return errors.New("usage: clover-cli webhooks deliveries list [-cursor cursor] [-limit N]")
		}
		query := url.Values{}
		if *cursor != "" {
			query.Set("cursor", *cursor)
		}
		if *limit > 0 {
			query.Set("limit", strconv.Itoa(*limit))
		}
		result, _, err := client.ListWebhookDeliveries(ctx, query)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "get":
		flags := flag.NewFlagSet("webhooks deliveries get", flag.ContinueOnError)
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli webhooks deliveries get <delivery-id>"); err != nil {
			return err
		}
		result, _, err := client.GetWebhookDelivery(ctx, flags.Arg(0))
		if err != nil {
			return err
		}
		return printJSON(result)
	case "replay":
		flags := flag.NewFlagSet("webhooks deliveries replay", flag.ContinueOnError)
		key := flags.String("idempotency-key", "", "optional idempotency key")
		if err := flags.Parse(flagArgsFirst(args[1:])); err != nil {
			return err
		}
		if err := requireArgs(flags, 1, "usage: clover-cli webhooks deliveries replay <delivery-id> [-idempotency-key <key>]"); err != nil {
			return err
		}
		if err := optionalKey(*key); err != nil {
			return err
		}
		result, _, err := client.ReplayWebhookDelivery(ctx, flags.Arg(0), *key)
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown webhook deliveries command %q", args[0])
	}
}

func runLogs(ctx context.Context, client *Client, args []string, forceFollow bool) error {
	flags := flag.NewFlagSet("logs", flag.ContinueOnError)
	follow := flags.Bool("follow", forceFollow, "poll /api/v1/logs until cancelled")
	cursor := flags.String("cursor", "", "pagination cursor")
	limit := flags.Int("limit", 0, "page size")
	page := flags.Int("page", 0, "page number")
	requestID := flags.String("request-id", "", "request ID filter")
	operation := flags.String("operation", "", "operation filter")
	statusMin := flags.Int("status-min", 0, "minimum HTTP status")
	statusMax := flags.Int("status-max", 0, "maximum HTTP status")
	source := flags.String("source", "", "request source filter")
	interval := flags.Duration("interval", 2*time.Second, "follow polling interval (minimum 100ms)")
	if err := flags.Parse(flagArgsFirst(args)); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("usage: clover-cli logs [-follow] [-cursor cursor] [-interval 2s]")
	}
	query := url.Values{}
	if *cursor != "" {
		query.Set("cursor", *cursor)
	}
	if *limit > 0 {
		query.Set("limit", strconv.Itoa(*limit))
	}
	if *page > 0 {
		query.Set("page", strconv.Itoa(*page))
	}
	if *requestID != "" {
		query.Set("request_id", *requestID)
	}
	if *operation != "" {
		query.Set("operation", *operation)
	}
	if *statusMin > 0 {
		query.Set("status_min", strconv.Itoa(*statusMin))
	}
	if *statusMax > 0 {
		query.Set("status_max", strconv.Itoa(*statusMax))
	}
	if *source != "" {
		query.Set("source", *source)
	}
	if *follow {
		err := client.FollowLogs(ctx, query, *interval, printCompactJSON)
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}
	result, _, err := client.ListLogs(ctx, query)
	if err != nil {
		return err
	}
	return printJSON(result)
}

func printCompactJSON(value json.RawMessage) error {
	var compact bytes.Buffer
	if err := json.Compact(&compact, value); err != nil {
		return err
	}
	_, err := fmt.Fprintln(os.Stdout, compact.String())
	return err
}
