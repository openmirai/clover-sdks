package clover

import (
	"context"
	"net/http"
	"net/url"
)

type AuditService struct{ client *Client }

type AuditListOptions struct {
	Page        int
	Limit       int
	Cursor      string
	TenantID    string
	ProjectID   string
	Environment string
	Query       url.Values
}

func (o AuditListOptions) values() url.Values {
	return (ScopedPageOptions{Page: o.Page, Limit: o.Limit, Cursor: o.Cursor, TenantID: o.TenantID, ProjectID: o.ProjectID, Environment: o.Environment, Query: o.Query}).values()
}

func (o AuditListOptions) scope() ScopedPageOptions {
	return ScopedPageOptions{Page: o.Page, Limit: o.Limit, Cursor: o.Cursor, TenantID: o.TenantID, ProjectID: o.ProjectID, Environment: o.Environment, Query: o.Query}
}

func (s *AuditService) List(ctx context.Context, options AuditListOptions) (AuditEventPage, ResponseMeta, error) {
	var result AuditEventPage
	path, err := scopedPathChecked("/api/v1/audit-events", options.scope(), false)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

func (s *AuditService) Append(ctx context.Context, request AppendAuditEventRequest, idempotencyKey string, options ...ScopedPageOptions) (AuditEvent, ResponseMeta, error) {
	var result AuditEvent
	scope, err := scopeOption(options, false)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	path, err := scopedPathChecked("/api/v1/audit-events", scope, false)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, request, idempotencyKey, &result)
	return result, meta, err
}

func (s *AuditService) Get(ctx context.Context, id string, options ScopedPageOptions) (AuditEvent, ResponseMeta, error) {
	var result AuditEvent
	path, err := scopedPathChecked("/api/v1/audit-events/"+segment(id), options, false)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

type AuditHoldListOptions struct {
	Page        int
	Limit       int
	Cursor      string
	TenantID    string
	ProjectID   string
	Environment string
	Active      *bool
	Query       url.Values
}

func (o AuditHoldListOptions) values() url.Values {
	v := (ScopedPageOptions{Page: o.Page, Limit: o.Limit, Cursor: o.Cursor, TenantID: o.TenantID, ProjectID: o.ProjectID, Environment: o.Environment, Query: o.Query}).values()
	if o.Active != nil {
		v.Set("active", boolString(*o.Active))
	}
	return v
}

func (o AuditHoldListOptions) scope() ScopedPageOptions {
	return ScopedPageOptions{Page: o.Page, Limit: o.Limit, Cursor: o.Cursor, TenantID: o.TenantID, ProjectID: o.ProjectID, Environment: o.Environment, Query: o.Query}
}

func (s *AuditService) Holds(ctx context.Context, options AuditHoldListOptions) (AuditHoldList, ResponseMeta, error) {
	var result AuditHoldList
	path, err := scopedPathChecked("/api/v1/audit-events/holds", options.scope(), false)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	if options.Active != nil {
		path = withValues(path, url.Values{"active": []string{boolString(*options.Active)}})
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

func (s *AuditService) CreateHold(ctx context.Context, request CreateAuditHoldRequest, idempotencyKey string, options ...ScopedPageOptions) (AuditHold, ResponseMeta, error) {
	var result AuditHold
	scope, err := scopeOption(options, false)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	path, err := scopedPathChecked("/api/v1/audit-events/holds", scope, false)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, request, idempotencyKey, &result)
	return result, meta, err
}

func (s *AuditService) ReleaseHold(ctx context.Context, id string, options ScopedPageOptions, idempotencyKey string) (JSON, ResponseMeta, error) {
	var result JSON
	path, err := scopedPathChecked("/api/v1/audit-events/holds/"+segment(id), options, false)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodDelete, path, nil, idempotencyKey, &result)
	return result, meta, err
}
