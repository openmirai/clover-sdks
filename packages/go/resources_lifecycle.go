package clover

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

func scopedPath(path string, options ScopedPageOptions) string {
	return withValues(path, options.values())
}

type TemplatesService struct{ client *Client }

type TemplateListOptions struct {
	Page        int
	Limit       int
	Cursor      string
	TenantID    string
	ProjectID   string
	Environment string
	Status      string
	Query       url.Values
}

func (o TemplateListOptions) values() url.Values {
	v := (ScopedPageOptions{Page: o.Page, Limit: o.Limit, Cursor: o.Cursor, TenantID: o.TenantID, ProjectID: o.ProjectID, Environment: o.Environment, Query: o.Query}).values()
	if o.Status != "" {
		v.Set("status", o.Status)
	}
	return v
}

func (o TemplateListOptions) scope() ScopedPageOptions {
	return ScopedPageOptions{Page: o.Page, Limit: o.Limit, Cursor: o.Cursor, TenantID: o.TenantID, ProjectID: o.ProjectID, Environment: o.Environment, Query: o.Query}
}

func (s *TemplatesService) List(ctx context.Context, options TemplateListOptions) (Page[Template], ResponseMeta, error) {
	var result Page[Template]
	path, err := scopedPathChecked("/api/v1/templates", options.scope(), true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	if options.Status != "" {
		path = withValues(path, url.Values{"status": []string{options.Status}})
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

func (s *TemplatesService) Create(ctx context.Context, request CreateTemplateRequest, idempotencyKey string) (Template, ResponseMeta, error) {
	var result Template
	if err := request.Scope.Validate(true); err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, "/api/v1/templates", request, idempotencyKey, &result)
	return result, meta, err
}

func (s *TemplatesService) Get(ctx context.Context, id string, options ScopedPageOptions) (Template, ResponseMeta, error) {
	var result Template
	path, err := scopedPathChecked("/api/v1/templates/"+segment(id), options, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

func (s *TemplatesService) Update(ctx context.Context, id string, request TemplateTransitionRequest, idempotencyKey string, options ...ScopedPageOptions) (Template, ResponseMeta, error) {
	var result Template
	scope, err := scopeOption(options, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	path, err := scopedPathChecked("/api/v1/templates/"+segment(id), scope, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPatch, path, request, idempotencyKey, &result)
	return result, meta, err
}

func (s *TemplatesService) Versions(ctx context.Context, templateID string, options ScopedPageOptions) ([]TemplateVersion, ResponseMeta, error) {
	var result []TemplateVersion
	path := "/api/v1/templates/" + segment(templateID) + "/versions"
	path, err := scopedPathChecked(path, options, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

func (s *TemplatesService) CreateVersion(ctx context.Context, templateID string, request CreateTemplateVersionRequest, idempotencyKey string) (TemplateVersion, ResponseMeta, error) {
	var result TemplateVersion
	if err := request.Scope.Validate(true); err != nil {
		return result, ResponseMeta{}, err
	}
	path := "/api/v1/templates/" + segment(templateID) + "/versions"
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, request, idempotencyKey, &result)
	return result, meta, err
}

func (s *TemplatesService) GetVersion(ctx context.Context, templateID, versionRef string, options ScopedPageOptions) (TemplateVersion, ResponseMeta, error) {
	var result TemplateVersion
	path, err := scopedPathChecked("/api/v1/templates/"+segment(templateID)+"/versions/"+segment(versionRef), options, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

func (s *TemplatesService) Compare(ctx context.Context, templateID, from, to string, options ScopedPageOptions) (TemplateVersionComparison, ResponseMeta, error) {
	var result TemplateVersionComparison
	if from == "" || to == "" {
		return result, ResponseMeta{}, fmt.Errorf("%w: from and to version references are required", ErrInvalidScope)
	}
	path, err := scopedPathChecked("/api/v1/templates/"+segment(templateID)+"/versions/compare", options, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	path = withValues(path, url.Values{"from": []string{from}, "to": []string{to}})
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

func (s *TemplatesService) Publish(ctx context.Context, templateID, versionID, idempotencyKey string, options ...ScopedPageOptions) (TemplateVersion, ResponseMeta, error) {
	var result TemplateVersion
	scope, err := scopeOption(options, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	path, err := scopedPathChecked("/api/v1/templates/"+segment(templateID)+"/versions/"+segment(versionID)+"/publish", scope, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, nil, idempotencyKey, &result)
	return result, meta, err
}

// Rollback uses the backend's explicit rollback route. It is intentionally
// distinct from Publish so callers can audit the requested operation.
func (s *TemplatesService) Rollback(ctx context.Context, templateID, versionID, idempotencyKey string, options ...ScopedPageOptions) (TemplateVersion, ResponseMeta, error) {
	var result TemplateVersion
	scope, err := scopeOption(options, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	path, err := scopedPathChecked("/api/v1/templates/"+segment(templateID)+"/versions/"+segment(versionID)+"/rollback", scope, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, nil, idempotencyKey, &result)
	return result, meta, err
}

type ContactsService struct{ client *Client }

type ContactListOptions struct {
	Page        int
	Limit       int
	Cursor      string
	TenantID    string
	ProjectID   string
	Environment string
	Status      string
	Query       url.Values
}

func (o ContactListOptions) values() url.Values {
	v := (ScopedPageOptions{Page: o.Page, Limit: o.Limit, Cursor: o.Cursor, TenantID: o.TenantID, ProjectID: o.ProjectID, Environment: o.Environment, Query: o.Query}).values()
	if o.Status != "" {
		v.Set("status", o.Status)
	}
	return v
}

func (o ContactListOptions) scope() ScopedPageOptions {
	return ScopedPageOptions{Page: o.Page, Limit: o.Limit, Cursor: o.Cursor, TenantID: o.TenantID, ProjectID: o.ProjectID, Environment: o.Environment, Query: o.Query}
}

func (s *ContactsService) List(ctx context.Context, options ContactListOptions) (Page[Contact], ResponseMeta, error) {
	var result Page[Contact]
	path, err := scopedPathChecked("/api/v1/contacts", options.scope(), true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	if options.Status != "" {
		path = withValues(path, url.Values{"status": []string{options.Status}})
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

func (s *ContactsService) Create(ctx context.Context, request CreateContactRequest, idempotencyKey string) (Contact, ResponseMeta, error) {
	var result Contact
	if err := request.Scope.Validate(true); err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, "/api/v1/contacts", request, idempotencyKey, &result)
	return result, meta, err
}

func (s *ContactsService) Get(ctx context.Context, id string, options ScopedPageOptions) (Contact, ResponseMeta, error) {
	var result Contact
	path, err := scopedPathChecked("/api/v1/contacts/"+segment(id), options, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

func (s *ContactsService) Transition(ctx context.Context, id string, request ContactTransitionRequest, idempotencyKey string, options ...ScopedPageOptions) (Contact, ResponseMeta, error) {
	var result Contact
	scope, err := scopeOption(options, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	path, err := scopedPathChecked("/api/v1/contacts/"+segment(id), scope, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPatch, path, request, idempotencyKey, &result)
	return result, meta, err
}

// Resubscribe is a convenience for the backend's contact transition event.
func (s *ContactsService) Resubscribe(ctx context.Context, id, idempotencyKey string, options ...ScopedPageOptions) (Contact, ResponseMeta, error) {
	return s.Transition(ctx, id, ContactTransitionRequest{Event: "resubscribe"}, idempotencyKey, options...)
}

type SegmentsService struct{ client *Client }

type SegmentListOptions struct {
	Page        int
	Limit       int
	Cursor      string
	TenantID    string
	ProjectID   string
	Environment string
	Query       url.Values
}

func (o SegmentListOptions) values() url.Values {
	return (ScopedPageOptions{Page: o.Page, Limit: o.Limit, Cursor: o.Cursor, TenantID: o.TenantID, ProjectID: o.ProjectID, Environment: o.Environment, Query: o.Query}).values()
}

func (o SegmentListOptions) scope() ScopedPageOptions {
	return ScopedPageOptions{Page: o.Page, Limit: o.Limit, Cursor: o.Cursor, TenantID: o.TenantID, ProjectID: o.ProjectID, Environment: o.Environment, Query: o.Query}
}

func (s *SegmentsService) List(ctx context.Context, options SegmentListOptions) (Page[Segment], ResponseMeta, error) {
	var result Page[Segment]
	path, err := scopedPathChecked("/api/v1/segments", options.scope(), true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

func (s *SegmentsService) Create(ctx context.Context, request CreateSegmentRequest, idempotencyKey string) (Segment, ResponseMeta, error) {
	var result Segment
	if err := request.Scope.Validate(true); err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, "/api/v1/segments", request, idempotencyKey, &result)
	return result, meta, err
}

func (s *SegmentsService) Get(ctx context.Context, id string, options ScopedPageOptions) (Segment, ResponseMeta, error) {
	var result Segment
	path, err := scopedPathChecked("/api/v1/segments/"+segment(id), options, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

func (s *SegmentsService) Archive(ctx context.Context, id, idempotencyKey string, options ...ScopedPageOptions) (Segment, ResponseMeta, error) {
	var result Segment
	scope, err := scopeOption(options, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	path, err := scopedPathChecked("/api/v1/segments/"+segment(id), scope, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPatch, path, nil, idempotencyKey, &result)
	return result, meta, err
}

// Evaluate is a read-only calculation exposed as POST by the backend and
// therefore intentionally does not require an Idempotency-Key.
func (s *SegmentsService) Evaluate(ctx context.Context, id string, options ScopedPageOptions) (Page[Contact], ResponseMeta, error) {
	var result Page[Contact]
	path, err := scopedPathChecked("/api/v1/segments/"+segment(id)+"/evaluate", options, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithoutIdempotency(s.client, ctx, http.MethodPost, path, nil, &result)
	return result, meta, err
}

type BroadcastsService struct{ client *Client }

type BroadcastListOptions struct {
	Page        int
	Limit       int
	Cursor      string
	TenantID    string
	ProjectID   string
	Environment string
	Query       url.Values
}

func (o BroadcastListOptions) values() url.Values {
	return (ScopedPageOptions{Page: o.Page, Limit: o.Limit, Cursor: o.Cursor, TenantID: o.TenantID, ProjectID: o.ProjectID, Environment: o.Environment, Query: o.Query}).values()
}

func (o BroadcastListOptions) scope() ScopedPageOptions {
	return ScopedPageOptions{Page: o.Page, Limit: o.Limit, Cursor: o.Cursor, TenantID: o.TenantID, ProjectID: o.ProjectID, Environment: o.Environment, Query: o.Query}
}

func (s *BroadcastsService) List(ctx context.Context, options BroadcastListOptions) (Page[Broadcast], ResponseMeta, error) {
	var result Page[Broadcast]
	path, err := scopedPathChecked("/api/v1/broadcasts", options.scope(), true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

func (s *BroadcastsService) Create(ctx context.Context, request CreateBroadcastRequest, idempotencyKey string) (Broadcast, ResponseMeta, error) {
	var result Broadcast
	if err := request.Scope.Validate(true); err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, "/api/v1/broadcasts", request, idempotencyKey, &result)
	return result, meta, err
}

func (s *BroadcastsService) Get(ctx context.Context, id string, options ScopedPageOptions) (Broadcast, ResponseMeta, error) {
	var result Broadcast
	path, err := scopedPathChecked("/api/v1/broadcasts/"+segment(id), options, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

func (s *BroadcastsService) Schedule(ctx context.Context, id string, request ScheduleBroadcastRequest, idempotencyKey string, options ...ScopedPageOptions) (Broadcast, ResponseMeta, error) {
	var result Broadcast
	scope, err := scopeOption(options, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	path, err := scopedPathChecked("/api/v1/broadcasts/"+segment(id)+"/schedule", scope, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, request, idempotencyKey, &result)
	return result, meta, err
}

func (s *BroadcastsService) Cancel(ctx context.Context, id, idempotencyKey string, options ...ScopedPageOptions) (Broadcast, ResponseMeta, error) {
	var result Broadcast
	scope, err := scopeOption(options, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	path, err := scopedPathChecked("/api/v1/broadcasts/"+segment(id)+"/cancel", scope, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, nil, idempotencyKey, &result)
	return result, meta, err
}

type AutomationsService struct{ client *Client }

type AutomationListOptions struct {
	Page        int
	Limit       int
	Cursor      string
	TenantID    string
	ProjectID   string
	Environment string
	Query       url.Values
}

func (o AutomationListOptions) values() url.Values {
	return (ScopedPageOptions{Page: o.Page, Limit: o.Limit, Cursor: o.Cursor, TenantID: o.TenantID, ProjectID: o.ProjectID, Environment: o.Environment, Query: o.Query}).values()
}

func (o AutomationListOptions) scope() ScopedPageOptions {
	return ScopedPageOptions{Page: o.Page, Limit: o.Limit, Cursor: o.Cursor, TenantID: o.TenantID, ProjectID: o.ProjectID, Environment: o.Environment, Query: o.Query}
}

func (s *AutomationsService) List(ctx context.Context, options AutomationListOptions) (Page[Automation], ResponseMeta, error) {
	var result Page[Automation]
	path, err := scopedPathChecked("/api/v1/automations", options.scope(), true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

func (s *AutomationsService) Create(ctx context.Context, request CreateAutomationRequest, idempotencyKey string) (Automation, ResponseMeta, error) {
	var result Automation
	if err := request.Scope.Validate(true); err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, "/api/v1/automations", request, idempotencyKey, &result)
	return result, meta, err
}

// Update replaces an automation's complete definition. The request scope is
// sent both as the JSON body scope and as the required lifecycle query scope;
// this prevents a valid body from being applied to a different tenant when a
// caller accidentally mixes scope values.
func (s *AutomationsService) Update(ctx context.Context, id string, request UpdateAutomationRequest, idempotencyKey string) (Automation, ResponseMeta, error) {
	var result Automation
	if err := request.Scope.Validate(true); err != nil {
		return result, ResponseMeta{}, err
	}
	options := ScopedPageOptions{
		TenantID:    request.Scope.TenantID,
		ProjectID:   request.Scope.ProjectID,
		Environment: request.Scope.Environment,
	}
	path, err := scopedPathChecked("/api/v1/automations/"+segment(id), options, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPatch, path, request, idempotencyKey, &result)
	return result, meta, err
}

func (s *AutomationsService) Get(ctx context.Context, id string, options ScopedPageOptions) (Automation, ResponseMeta, error) {
	var result Automation
	path, err := scopedPathChecked("/api/v1/automations/"+segment(id), options, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

func (s *AutomationsService) Activate(ctx context.Context, id, idempotencyKey string, options ...ScopedPageOptions) (Automation, ResponseMeta, error) {
	return s.transition(ctx, id, "activate", idempotencyKey, options...)
}

func (s *AutomationsService) Pause(ctx context.Context, id, idempotencyKey string, options ...ScopedPageOptions) (Automation, ResponseMeta, error) {
	return s.transition(ctx, id, "pause", idempotencyKey, options...)
}

func (s *AutomationsService) transition(ctx context.Context, id, action, idempotencyKey string, options ...ScopedPageOptions) (Automation, ResponseMeta, error) {
	var result Automation
	scope, err := scopeOption(options, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	path, err := scopedPathChecked("/api/v1/automations/"+segment(id)+"/"+action, scope, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, nil, idempotencyKey, &result)
	return result, meta, err
}

func (s *AutomationsService) StartRun(ctx context.Context, id string, request StartAutomationRunRequest, idempotencyKey string, options ...ScopedPageOptions) (AutomationRun, ResponseMeta, error) {
	var result AutomationRun
	scope, err := scopeOption(options, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	path, err := scopedPathChecked("/api/v1/automations/"+segment(id)+"/runs", scope, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, request, idempotencyKey, &result)
	return result, meta, err
}

func (s *AutomationsService) GetRun(ctx context.Context, automationID, runID string, options ScopedPageOptions) (AutomationRun, ResponseMeta, error) {
	var result AutomationRun
	path := "/api/v1/automations/" + segment(automationID) + "/runs/" + segment(runID)
	path, err := scopedPathChecked(path, options, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

func (s *AutomationsService) IngestEvent(ctx context.Context, request IngestAutomationEventRequest, options ScopedPageOptions, idempotencyKey string) (AutomationEventResult, ResponseMeta, error) {
	var result AutomationEventResult
	path, err := scopedPathChecked("/api/v1/automation-events", options, true)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, request, idempotencyKey, &result)
	return result, meta, err
}
