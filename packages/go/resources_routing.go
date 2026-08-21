package clover

import (
	"context"
	"net/http"
	"net/url"
)

// RoutingService exposes the provider-neutral routing control plane. Routing
// mutations use the supplied idempotency key as their durable command ID;
// reads remain keyless.
type RoutingService struct{ client *Client }

// ProviderRoutingService is an explicit alias for integrations that prefer
// the longer resource name.
type ProviderRoutingService = RoutingService

func (s *RoutingService) GetPolicy(ctx context.Context) (RoutingPolicy, ResponseMeta, error) {
	var result RoutingPolicy
	meta, err := requestTyped(s.client, ctx, http.MethodGet, "/api/v1/routing/policy", nil, "", &result)
	return result, meta, err
}

func (s *RoutingService) PutPolicy(ctx context.Context, request RoutingPolicyRequest, idempotencyKey string) (RoutingPolicy, ResponseMeta, error) {
	var result RoutingPolicy
	meta, err := requestTyped(s.client, ctx, http.MethodPut, "/api/v1/routing/policy", request, idempotencyKey, &result)
	return result, meta, err
}

func (s *RoutingService) ListCapabilities(ctx context.Context) (RoutingCapabilities, ResponseMeta, error) {
	var result RoutingCapabilities
	meta, err := requestTyped(s.client, ctx, http.MethodGet, "/api/v1/routing/capabilities", nil, "", &result)
	return result, meta, err
}

func (s *RoutingService) ListPools(ctx context.Context) (RoutingPools, ResponseMeta, error) {
	var result RoutingPools
	meta, err := requestTyped(s.client, ctx, http.MethodGet, "/api/v1/routing/pools", nil, "", &result)
	return result, meta, err
}

func (s *RoutingService) CreatePool(ctx context.Context, request CreatePoolRequest, idempotencyKey string) (DedicatedPool, ResponseMeta, error) {
	var result DedicatedPool
	meta, err := requestTyped(s.client, ctx, http.MethodPost, "/api/v1/routing/pools", request, idempotencyKey, &result)
	return result, meta, err
}

func (s *RoutingService) GetPool(ctx context.Context, id string) (DedicatedPool, ResponseMeta, error) {
	var result DedicatedPool
	meta, err := requestTyped(s.client, ctx, http.MethodGet, "/api/v1/routing/pools/"+segment(id), nil, "", &result)
	return result, meta, err
}

func (s *RoutingService) ApplyPoolCommand(ctx context.Context, id string, request PoolCommandRequest, idempotencyKey string) (RoutingTransition, ResponseMeta, error) {
	var result RoutingTransition
	path := "/api/v1/routing/pools/" + segment(id) + "/command"
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, request, idempotencyKey, &result)
	return result, meta, err
}

func (s *RoutingService) ApplyIPCommand(ctx context.Context, poolID, ipID string, request IPCommandRequest, idempotencyKey string) (RoutingTransition, ResponseMeta, error) {
	var result RoutingTransition
	path := "/api/v1/routing/pools/" + segment(poolID) + "/ips/" + segment(ipID) + "/command"
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, request, idempotencyKey, &result)
	return result, meta, err
}

func (s *RoutingService) ListAudit(ctx context.Context, options RoutingAuditOptions) (RoutingAudit, ResponseMeta, error) {
	var result RoutingAudit
	values := make(url.Values)
	if options.EntityType != "" {
		values.Set("entityType", options.EntityType)
	}
	if options.EntityID != "" {
		values.Set("entityId", options.EntityID)
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, withValues("/api/v1/routing/audit", values), nil, "", &result)
	return result, meta, err
}
