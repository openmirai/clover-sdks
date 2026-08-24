package clover

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// CursorPage is the response shape used by cursor-based platform resources.
type CursorPage[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor"`
}

var (
	ErrMissingPlatformScope = errors.New("account_id and environment_id are required")
	ErrInvalidPlatformScope = errors.New("invalid platform scope")
)

func (s PlatformScope) Validate() error {
	if strings.TrimSpace(s.AccountID) == "" || strings.TrimSpace(s.EnvironmentID) == "" {
		return ErrMissingPlatformScope
	}
	if len(s.AccountID) > 256 || len(s.EnvironmentID) > 256 {
		return fmt.Errorf("%w: account_id and environment_id must be at most 256 bytes", ErrInvalidPlatformScope)
	}
	return nil
}

func (s AccountScope) Validate() error {
	if strings.TrimSpace(s.AccountID) == "" {
		return fmt.Errorf("%w: account_id is required", ErrInvalidPlatformScope)
	}
	if len(s.AccountID) > 256 {
		return fmt.Errorf("%w: account_id must be at most 256 bytes", ErrInvalidPlatformScope)
	}
	return nil
}

func platformAccountPath(scope AccountScope, suffix string) (string, error) {
	if err := scope.Validate(); err != nil {
		return "", err
	}
	return "/api/v1/platform/accounts/" + segment(scope.AccountID) + suffix, nil
}

func platformEnvironmentPath(scope PlatformScope, suffix string) (string, error) {
	if err := scope.Validate(); err != nil {
		return "", err
	}
	return "/api/v1/platform/accounts/" + segment(scope.AccountID) + "/environments/" + segment(scope.EnvironmentID) + suffix, nil
}

func platformPageValues(options PlatformPageOptions) url.Values {
	values := cloneValues(options.Query)
	if options.Page != 0 {
		values.Set("page", strconv.Itoa(options.Page))
	}
	if options.Limit != 0 {
		values.Set("limit", strconv.Itoa(options.Limit))
	}
	if options.Cursor != "" {
		values.Set("cursor", options.Cursor)
	}
	if options.Status != "" {
		values.Set("status", options.Status)
	}
	if options.Enabled != nil {
		values.Set("enabled", strconv.FormatBool(*options.Enabled))
	}
	if options.From != "" {
		values.Set("from", options.From)
	}
	if options.To != "" {
		values.Set("to", options.To)
	}
	for key, entries := range values {
		if len(entries) == 0 || (len(entries) == 1 && entries[0] == "") {
			delete(values, key)
		}
	}
	return values
}

func platformPathWithQuery(path string, options PlatformPageOptions) string {
	return withValues(path, platformPageValues(options))
}

func platformOptionalKey(keys []string) (string, error) {
	return optionalIdempotencyKey(keys)
}

// platformRequiredKey preserves the variadic source-compatibility shape used
// by a few early platform helpers while enforcing the generated contract's
// required Idempotency-Key header for new mutation routes.
func platformRequiredKey(keys []string) (string, error) {
	if len(keys) != 1 {
		return "", fmt.Errorf("%w: exactly one idempotency key is required", ErrInvalidIdempotencyKey)
	}
	if err := ValidateIdempotencyKey(keys[0]); err != nil {
		return "", err
	}
	return keys[0], nil
}

// PlatformService is the account/environment-scoped Phase 1 API. All child
// resources emit account_id and environment_id as escaped path segments.
type PlatformService struct {
	Accounts     *PlatformAccountsService
	Environments *PlatformEnvironmentsService
	Messages     *PlatformMessagesService
	Templates    *PlatformTemplatesService
	Webhooks     *PlatformWebhooksService
	Timeline     *MessageTimelineService
	// Logs aliases the redacted message timeline endpoint. The old request-log
	// route is not part of the account/environment platform contract.
	Logs             *MessageTimelineService
	Inbound          *PlatformInboundService
	Preferences      *PlatformPreferencesService
	Suppressions     *PlatformSuppressionsService
	SMTP             *SMTPService
	Contacts         *PlatformContactsService
	Segments         *PlatformSegmentsService
	Automations      *PlatformAutomationsService
	Routing          *PlatformRoutingService
	Domains          *PlatformDomainsService
	DomainHealth     *PlatformDomainHealthService
	Usage            *UsageContractService
	Audiences        *PlatformAudiencesService
	Broadcasts       *PlatformBroadcastsService
	ProviderBindings *PlatformProviderBindingsService
}

func newPlatformService(client *Client) *PlatformService {
	return &PlatformService{
		Accounts:         &PlatformAccountsService{client: client},
		Environments:     &PlatformEnvironmentsService{client: client},
		Messages:         &PlatformMessagesService{client: client},
		Templates:        &PlatformTemplatesService{client: client},
		Webhooks:         &PlatformWebhooksService{client: client},
		Timeline:         &MessageTimelineService{client: client},
		Logs:             &MessageTimelineService{client: client},
		Inbound:          &PlatformInboundService{client: client},
		Preferences:      &PlatformPreferencesService{client: client},
		Suppressions:     &PlatformSuppressionsService{client: client},
		SMTP:             &SMTPService{client: client},
		Contacts:         &PlatformContactsService{client: client},
		Segments:         &PlatformSegmentsService{client: client},
		Automations:      &PlatformAutomationsService{client: client},
		Routing:          &PlatformRoutingService{client: client},
		Domains:          &PlatformDomainsService{client: client},
		DomainHealth:     &PlatformDomainHealthService{client: client},
		Usage:            &UsageContractService{client: client},
		Audiences:        &PlatformAudiencesService{client: client},
		Broadcasts:       &PlatformBroadcastsService{client: client},
		ProviderBindings: &PlatformProviderBindingsService{client: client},
	}
}

type PlatformAccountsService struct{ client *Client }

func (s *PlatformAccountsService) List(ctx context.Context, options PlatformPageOptions) (Page[ClientAccount], ResponseMeta, error) {
	var result Page[ClientAccount]
	meta, err := requestTyped(s.client, ctx, http.MethodGet, platformPathWithQuery("/api/v1/platform/accounts", options), nil, "", &result)
	return result, meta, err
}

func (s *PlatformAccountsService) Create(ctx context.Context, request CreateClientAccountRequest, idempotencyKey string) (ClientAccount, ResponseMeta, error) {
	var result ClientAccount
	meta, err := requestTyped(s.client, ctx, http.MethodPost, "/api/v1/platform/accounts", request, idempotencyKey, &result)
	return result, meta, err
}

func (s *PlatformAccountsService) Get(ctx context.Context, accountID string) (ClientAccount, ResponseMeta, error) {
	var result ClientAccount
	path, err := platformAccountPath(AccountScope{AccountID: accountID}, "")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

func (s *PlatformAccountsService) Update(ctx context.Context, accountID string, request UpdateClientAccountRequest, idempotencyKey string) (ClientAccount, ResponseMeta, error) {
	var result ClientAccount
	path, err := platformAccountPath(AccountScope{AccountID: accountID}, "")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPatch, path, request, idempotencyKey, &result)
	return result, meta, err
}

type PlatformEnvironmentsService struct{ client *Client }

func (s *PlatformEnvironmentsService) List(ctx context.Context, accountID string, options PlatformPageOptions) (Page[ClientEnvironment], ResponseMeta, error) {
	var result Page[ClientEnvironment]
	path, err := platformAccountPath(AccountScope{AccountID: accountID}, "/environments")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, platformPathWithQuery(path, options), nil, "", &result)
	return result, meta, err
}

func (s *PlatformEnvironmentsService) Create(ctx context.Context, accountID string, request CreateEnvironmentRequest, idempotencyKey string) (ClientEnvironment, ResponseMeta, error) {
	var result ClientEnvironment
	path, err := platformAccountPath(AccountScope{AccountID: accountID}, "/environments")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, request, idempotencyKey, &result)
	return result, meta, err
}

type PlatformMessagesService struct{ client *Client }

func (s *PlatformMessagesService) List(ctx context.Context, scope PlatformScope, options PlatformPageOptions) (Page[MessageSummary], ResponseMeta, error) {
	var result Page[MessageSummary]
	path, err := platformEnvironmentPath(scope, "/messages")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, platformPathWithQuery(path, options), nil, "", &result)
	return result, meta, err
}

func (s *PlatformMessagesService) Send(ctx context.Context, scope PlatformScope, request SendMessageRequest, idempotencyKey string) (MessageAccepted, ResponseMeta, error) {
	var result MessageAccepted
	path, err := platformEnvironmentPath(scope, "/messages")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, request, idempotencyKey, &result)
	return result, meta, err
}

func (s *PlatformMessagesService) SendBatch(ctx context.Context, scope PlatformScope, request SendMessageBatchRequest, idempotencyKey string) (BatchAccepted, ResponseMeta, error) {
	var result BatchAccepted
	path, err := platformEnvironmentPath(scope, "/messages/batch")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, request, idempotencyKey, &result)
	return result, meta, err
}

func (s *PlatformMessagesService) Get(ctx context.Context, scope PlatformScope, messageID string) (MessageDetail, ResponseMeta, error) {
	var result MessageDetail
	path, err := platformEnvironmentPath(scope, "/messages/"+segment(messageID))
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

func (s *PlatformMessagesService) Schedule(ctx context.Context, scope PlatformScope, messageID string, request ScheduleMessageRequest, idempotencyKey string) (MessageAccepted, ResponseMeta, error) {
	var result MessageAccepted
	path, err := platformEnvironmentPath(scope, "/messages/"+segment(messageID)+"/schedule")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, request, idempotencyKey, &result)
	return result, meta, err
}

func (s *PlatformMessagesService) Cancel(ctx context.Context, scope PlatformScope, messageID, idempotencyKey string) (MessageDetail, ResponseMeta, error) {
	var result MessageDetail
	path, err := platformEnvironmentPath(scope, "/messages/"+segment(messageID)+"/cancel")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, nil, idempotencyKey, &result)
	return result, meta, err
}

type PlatformTemplatesService struct{ client *Client }

func (s *PlatformTemplatesService) List(ctx context.Context, scope PlatformScope, options PlatformPageOptions) (Page[PlatformTemplate], ResponseMeta, error) {
	var result Page[PlatformTemplate]
	path, err := platformEnvironmentPath(scope, "/templates")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, platformPathWithQuery(path, options), nil, "", &result)
	return result, meta, err
}

func (s *PlatformTemplatesService) Create(ctx context.Context, scope PlatformScope, request CreatePlatformTemplateRequest, idempotencyKey string) (PlatformTemplate, ResponseMeta, error) {
	var result PlatformTemplate
	path, err := platformEnvironmentPath(scope, "/templates")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, request, idempotencyKey, &result)
	return result, meta, err
}

func (s *PlatformTemplatesService) Get(ctx context.Context, scope PlatformScope, templateID string) (PlatformTemplate, ResponseMeta, error) {
	var result PlatformTemplate
	path, err := platformEnvironmentPath(scope, "/templates/"+segment(templateID))
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

func (s *PlatformTemplatesService) Update(ctx context.Context, scope PlatformScope, templateID string, request UpdatePlatformTemplateRequest, idempotencyKey string) (PlatformTemplate, ResponseMeta, error) {
	var result PlatformTemplate
	path, err := platformEnvironmentPath(scope, "/templates/"+segment(templateID))
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPatch, path, request, idempotencyKey, &result)
	return result, meta, err
}

func (s *PlatformTemplatesService) Archive(ctx context.Context, scope PlatformScope, templateID, idempotencyKey string) (PlatformTemplate, ResponseMeta, error) {
	return s.transition(ctx, scope, templateID, "archive", idempotencyKey)
}
func (s *PlatformTemplatesService) Unarchive(ctx context.Context, scope PlatformScope, templateID, idempotencyKey string) (PlatformTemplate, ResponseMeta, error) {
	return s.transition(ctx, scope, templateID, "unarchive", idempotencyKey)
}
func (s *PlatformTemplatesService) transition(ctx context.Context, scope PlatformScope, templateID, action, idempotencyKey string) (PlatformTemplate, ResponseMeta, error) {
	var result PlatformTemplate
	path, err := platformEnvironmentPath(scope, "/templates/"+segment(templateID)+"/"+action)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, nil, idempotencyKey, &result)
	return result, meta, err
}

func (s *PlatformTemplatesService) Versions(ctx context.Context, scope PlatformScope, templateID string) (CursorPage[PlatformTemplateVersion], ResponseMeta, error) {
	var result CursorPage[PlatformTemplateVersion]
	path, err := platformEnvironmentPath(scope, "/templates/"+segment(templateID)+"/versions")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

func (s *PlatformTemplatesService) CreateVersion(ctx context.Context, scope PlatformScope, templateID string, request CreatePlatformTemplateVersionRequest, idempotencyKey string) (PlatformTemplateVersion, ResponseMeta, error) {
	var result PlatformTemplateVersion
	path, err := platformEnvironmentPath(scope, "/templates/"+segment(templateID)+"/versions")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, request, idempotencyKey, &result)
	return result, meta, err
}

func (s *PlatformTemplatesService) GetVersion(ctx context.Context, scope PlatformScope, templateID, versionRef string) (PlatformTemplateVersion, ResponseMeta, error) {
	var result PlatformTemplateVersion
	path, err := platformEnvironmentPath(scope, "/templates/"+segment(templateID)+"/versions/"+segment(versionRef))
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

func (s *PlatformTemplatesService) Compare(ctx context.Context, scope PlatformScope, templateID, from, to string) (TemplateComparison, ResponseMeta, error) {
	var result TemplateComparison
	path, err := platformEnvironmentPath(scope, "/templates/"+segment(templateID)+"/versions/compare")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	path = withValues(path, url.Values{"from": []string{from}, "to": []string{to}})
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

func (s *PlatformTemplatesService) Publish(ctx context.Context, scope PlatformScope, templateID, versionRef, idempotencyKey string) (PlatformTemplateVersion, ResponseMeta, error) {
	return s.versionTransition(ctx, scope, templateID, versionRef, "publish", idempotencyKey)
}
func (s *PlatformTemplatesService) Rollback(ctx context.Context, scope PlatformScope, templateID, versionRef, idempotencyKey string) (PlatformTemplateVersion, ResponseMeta, error) {
	return s.versionTransition(ctx, scope, templateID, versionRef, "rollback", idempotencyKey)
}
func (s *PlatformTemplatesService) versionTransition(ctx context.Context, scope PlatformScope, templateID, versionRef, action, idempotencyKey string) (PlatformTemplateVersion, ResponseMeta, error) {
	var result PlatformTemplateVersion
	path, err := platformEnvironmentPath(scope, "/templates/"+segment(templateID)+"/versions/"+segment(versionRef)+"/"+action)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, nil, idempotencyKey, &result)
	return result, meta, err
}

func (s *PlatformTemplatesService) Render(ctx context.Context, scope PlatformScope, templateID, versionRef string, request RenderTemplateRequest) (RenderedTemplate, ResponseMeta, error) {
	var result RenderedTemplate
	path, err := platformEnvironmentPath(scope, "/templates/"+segment(templateID)+"/versions/"+segment(versionRef)+"/render")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithoutIdempotency(s.client, ctx, http.MethodPost, path, request, &result)
	return result, meta, err
}

type PlatformWebhooksService struct{ client *Client }

func (s *PlatformWebhooksService) List(ctx context.Context, scope PlatformScope, options PlatformPageOptions) (CursorPage[ScopedWebhook], ResponseMeta, error) {
	var result CursorPage[ScopedWebhook]
	path, err := platformEnvironmentPath(scope, "/webhooks")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, platformPathWithQuery(path, options), nil, "", &result)
	return result, meta, err
}

func (s *PlatformWebhooksService) Create(ctx context.Context, scope PlatformScope, request CreateScopedWebhookRequest, idempotencyKeys ...string) (CreatedScopedWebhook, ResponseMeta, error) {
	var result CreatedScopedWebhook
	key, err := platformRequiredKey(idempotencyKeys)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	path, err := platformEnvironmentPath(scope, "/webhooks")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, request, key, &result)
	return result, meta, err
}

func (s *PlatformWebhooksService) Get(ctx context.Context, scope PlatformScope, webhookID string) (ScopedWebhook, ResponseMeta, error) {
	var result ScopedWebhook
	path, err := platformEnvironmentPath(scope, "/webhooks/"+segment(webhookID))
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

func (s *PlatformWebhooksService) Update(ctx context.Context, scope PlatformScope, webhookID string, request UpdateScopedWebhookRequest, idempotencyKeys ...string) (ScopedWebhook, ResponseMeta, error) {
	var result ScopedWebhook
	key, err := platformOptionalKey(idempotencyKeys)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	path, err := platformEnvironmentPath(scope, "/webhooks/"+segment(webhookID))
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithPolicy(s.client, ctx, http.MethodPatch, path, request, key, &result, true)
	return result, meta, err
}

func (s *PlatformWebhooksService) Delete(ctx context.Context, scope PlatformScope, webhookID string, idempotencyKeys ...string) (ResponseMeta, error) {
	key, err := platformOptionalKey(idempotencyKeys)
	if err != nil {
		return ResponseMeta{}, err
	}
	path, err := platformEnvironmentPath(scope, "/webhooks/"+segment(webhookID))
	if err != nil {
		return ResponseMeta{}, err
	}
	return requestTypedWithPolicy(s.client, ctx, http.MethodDelete, path, nil, key, (*JSON)(nil), true)
}

func (s *PlatformWebhooksService) RotateSecret(ctx context.Context, scope PlatformScope, webhookID string, request map[string]any, idempotencyKeys ...string) (RotatedScopedWebhookSecret, ResponseMeta, error) {
	var result RotatedScopedWebhookSecret
	key, err := platformOptionalKey(idempotencyKeys)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	path, err := platformEnvironmentPath(scope, "/webhooks/"+segment(webhookID)+"/rotate-secret")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithPolicy(s.client, ctx, http.MethodPost, path, request, key, &result, true)
	return result, meta, err
}

func (s *PlatformWebhooksService) Deliveries(ctx context.Context, scope PlatformScope, options PlatformPageOptions) (CursorPage[ScopedWebhookDelivery], ResponseMeta, error) {
	var result CursorPage[ScopedWebhookDelivery]
	path, err := platformEnvironmentPath(scope, "/webhook-deliveries")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, platformPathWithQuery(path, options), nil, "", &result)
	return result, meta, err
}

func (s *PlatformWebhooksService) Delivery(ctx context.Context, scope PlatformScope, deliveryID string) (ScopedWebhookDelivery, ResponseMeta, error) {
	var result ScopedWebhookDelivery
	path, err := platformEnvironmentPath(scope, "/webhook-deliveries/"+segment(deliveryID))
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

func (s *PlatformWebhooksService) ReplayDelivery(ctx context.Context, scope PlatformScope, deliveryID string, idempotencyKeys ...string) (ScopedWebhookDelivery, ResponseMeta, error) {
	var result ScopedWebhookDelivery
	key, err := platformOptionalKey(idempotencyKeys)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	path, err := platformEnvironmentPath(scope, "/webhook-deliveries/"+segment(deliveryID)+"/replay")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithPolicy(s.client, ctx, http.MethodPost, path, map[string]any{}, key, &result, true)
	return result, meta, err
}

type MessageTimelineOptions struct {
	Cursor                                                                       string
	Limit                                                                        int
	Q, MessageKind, MessageRef, EventType, Status, OccurredAfter, OccurredBefore string
}

func (o MessageTimelineOptions) values() url.Values {
	return platformPageValues(PlatformPageOptions{Cursor: o.Cursor, Limit: o.Limit, Query: url.Values{"q": []string{o.Q}, "message_kind": []string{o.MessageKind}, "message_ref": []string{o.MessageRef}, "event_type": []string{o.EventType}, "status": []string{o.Status}, "occurred_after": []string{o.OccurredAfter}, "occurred_before": []string{o.OccurredBefore}}})
}

type MessageTimelineService struct{ client *Client }

func (s *MessageTimelineService) List(ctx context.Context, scope PlatformScope, options MessageTimelineOptions) (CursorPage[MessageTimelineEntry], ResponseMeta, error) {
	var result CursorPage[MessageTimelineEntry]
	path, err := platformEnvironmentPath(scope, "/message-timeline")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, withValues(path, options.values()), nil, "", &result)
	return result, meta, err
}

type PlatformInboundService struct{ client *Client }
type ReceivedMessageListOptions struct {
	Cursor                                               string
	Limit                                                int
	ParseStatus, DomainID, ReceivedAfter, ReceivedBefore string
}

func (o ReceivedMessageListOptions) values() url.Values {
	return platformPageValues(PlatformPageOptions{Cursor: o.Cursor, Limit: o.Limit, Query: url.Values{"parse_status": []string{o.ParseStatus}, "domain_id": []string{o.DomainID}, "received_after": []string{o.ReceivedAfter}, "received_before": []string{o.ReceivedBefore}}})
}
func (s *PlatformInboundService) List(ctx context.Context, scope PlatformScope, options ReceivedMessageListOptions) (CursorPage[InboundMessageSummary], ResponseMeta, error) {
	var result CursorPage[InboundMessageSummary]
	path, err := platformEnvironmentPath(scope, "/received-messages")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, withValues(path, options.values()), nil, "", &result)
	return result, meta, err
}
func (s *PlatformInboundService) Get(ctx context.Context, scope PlatformScope, messageID string) (InboundMessage, ResponseMeta, error) {
	var result InboundMessage
	path, err := platformEnvironmentPath(scope, "/received-messages/"+segment(messageID))
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}
func (s *PlatformInboundService) AttachmentURL(ctx context.Context, scope PlatformScope, messageID, attachmentID string) (InboundAttachmentDownload, ResponseMeta, error) {
	var result InboundAttachmentDownload
	path, err := platformEnvironmentPath(scope, "/received-messages/"+segment(messageID)+"/attachments/"+segment(attachmentID))
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

// AcceptProviderMessage is the provider callback contract. It intentionally
// takes bounded JSON and explicit provider headers; it does not add bearer
// authentication or invent an idempotency key.
func (s *PlatformInboundService) AcceptProviderMessage(ctx context.Context, provider string, payload JSON, headers http.Header) (JSON, ResponseMeta, error) {
	var result JSON
	path := "/api/v1/inbound/" + segment(provider)
	meta, err := requestTypedWithHeaders(s.client, ctx, http.MethodPost, path, payload, "", &result, true, headers)
	return result, meta, err
}
func (s *PlatformInboundService) GetReceivingDomain(ctx context.Context, scope PlatformScope, domainID string) (ReceivingDomain, ResponseMeta, error) {
	var result ReceivingDomain
	path, err := platformEnvironmentPath(scope, "/domains/"+segment(domainID)+"/receiving")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}
func (s *PlatformInboundService) UpdateReceivingDomain(ctx context.Context, scope PlatformScope, domainID string, request UpdateReceivingDomainRequest) (ReceivingDomain, ResponseMeta, error) {
	var result ReceivingDomain
	path, err := platformEnvironmentPath(scope, "/domains/"+segment(domainID)+"/receiving")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithoutIdempotency(s.client, ctx, http.MethodPut, path, request, &result)
	return result, meta, err
}
func (s *PlatformInboundService) DisableReceiving(ctx context.Context, scope PlatformScope, domainID string) (ReceivingDomain, ResponseMeta, error) {
	var result ReceivingDomain
	path, err := platformEnvironmentPath(scope, "/domains/"+segment(domainID)+"/receiving")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithoutIdempotency(s.client, ctx, http.MethodDelete, path, nil, &result)
	return result, meta, err
}

type PlatformPreferencesService struct{ client *Client }

func (s *PlatformPreferencesService) Topics(ctx context.Context, scope PlatformScope) (CursorPage[PlatformPreferenceTopic], ResponseMeta, error) {
	var result CursorPage[PlatformPreferenceTopic]
	path, err := platformEnvironmentPath(scope, "/preference-topics")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}
func (s *PlatformPreferencesService) CreateTopic(ctx context.Context, scope PlatformScope, request CreatePlatformPreferenceTopicRequest, idempotencyKeys ...string) (PlatformPreferenceTopic, ResponseMeta, error) {
	var result PlatformPreferenceTopic
	key, err := platformRequiredKey(idempotencyKeys)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	path, err := platformEnvironmentPath(scope, "/preference-topics")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, request, key, &result)
	return result, meta, err
}
func (s *PlatformPreferencesService) IssueToken(ctx context.Context, scope PlatformScope, request IssuePreferenceTokenRequest) (PreferenceToken, ResponseMeta, error) {
	var result PreferenceToken
	path, err := platformEnvironmentPath(scope, "/preference-tokens")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithoutIdempotency(s.client, ctx, http.MethodPost, path, request, &result)
	return result, meta, err
}
func (s *PlatformPreferencesService) GetCenter(ctx context.Context, token string) (PlatformPreferenceCenter, ResponseMeta, error) {
	var result PlatformPreferenceCenter
	meta, err := requestTypedWithPolicy(s.client, ctx, http.MethodGet, "/api/v1/preferences/"+segment(token), nil, "", &result, true)
	return result, meta, err
}
func (s *PlatformPreferencesService) UpdateCenter(ctx context.Context, token string, request UpdatePreferenceCenterRequest) (PlatformPreferenceCenter, ResponseMeta, error) {
	var result PlatformPreferenceCenter
	meta, err := requestTypedWithoutIdempotency(s.client, ctx, http.MethodPost, "/api/v1/preferences/"+segment(token), request, &result)
	return result, meta, err
}
func (s *PlatformPreferencesService) Unsubscribe(ctx context.Context, token string) (ResponseMeta, error) {
	return requestTypedWithoutIdempotency(s.client, ctx, http.MethodPost, "/api/v1/unsubscribe/"+segment(token), nil, (*JSON)(nil))
}

type PlatformSuppressionsService struct{ client *Client }
type PlatformSuppressionListOptions struct {
	Cursor              string
	Limit               int
	Active              *bool
	Reason, EmailSHA256 string
}

func (o PlatformSuppressionListOptions) values() url.Values {
	query := platformPageValues(PlatformPageOptions{Cursor: o.Cursor, Limit: o.Limit, Query: url.Values{"reason": []string{o.Reason}, "email_sha256": []string{o.EmailSHA256}}})
	if o.Active != nil {
		query.Set("active", strconv.FormatBool(*o.Active))
	}
	return query
}
func (s *PlatformSuppressionsService) List(ctx context.Context, scope PlatformScope, options PlatformSuppressionListOptions) (CursorPage[SuppressionEntry], ResponseMeta, error) {
	var result CursorPage[SuppressionEntry]
	path, err := platformEnvironmentPath(scope, "/suppressions")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, withValues(path, options.values()), nil, "", &result)
	return result, meta, err
}
func (s *PlatformSuppressionsService) Create(ctx context.Context, scope PlatformScope, request CreatePlatformSuppressionRequest, idempotencyKeys ...string) (SuppressionEntry, ResponseMeta, error) {
	var result SuppressionEntry
	key, err := platformRequiredKey(idempotencyKeys)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	path, err := platformEnvironmentPath(scope, "/suppressions")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, request, key, &result)
	return result, meta, err
}
func (s *PlatformSuppressionsService) Delete(ctx context.Context, scope PlatformScope, suppressionID string) (ResponseMeta, error) {
	path, err := platformEnvironmentPath(scope, "/suppressions/"+segment(suppressionID))
	if err != nil {
		return ResponseMeta{}, err
	}
	return requestTypedWithoutIdempotency(s.client, ctx, http.MethodDelete, path, nil, (*JSON)(nil))
}
func (s *PlatformSuppressionsService) Reactivate(ctx context.Context, scope PlatformScope, suppressionID string) (SuppressionEntry, ResponseMeta, error) {
	var result SuppressionEntry
	path, err := platformEnvironmentPath(scope, "/suppressions/"+segment(suppressionID)+"/reactivate")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithoutIdempotency(s.client, ctx, http.MethodPost, path, nil, &result)
	return result, meta, err
}

type SMTPService struct{ client *Client }
type SMTPListOptions struct {
	Page, Limit          int
	IncludeRevoked       *bool
	Status, CredentialID string
}

func (o SMTPListOptions) values() url.Values {
	query := platformPageValues(PlatformPageOptions{Page: o.Page, Limit: o.Limit, Status: o.Status})
	if o.IncludeRevoked != nil {
		query.Set("includeRevoked", strconv.FormatBool(*o.IncludeRevoked))
	}
	if o.CredentialID != "" {
		query.Set("credentialId", o.CredentialID)
	}
	return query
}
func (s *SMTPService) ListCredentials(ctx context.Context, scope PlatformScope, options SMTPListOptions) (Page[SMTPCredential], ResponseMeta, error) {
	var result Page[SMTPCredential]
	path, err := platformEnvironmentPath(scope, "/smtp-credentials")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, withValues(path, options.values()), nil, "", &result)
	return result, meta, err
}
func (s *SMTPService) CreateCredential(ctx context.Context, scope PlatformScope, request SMTPCredentialCreateRequest, idempotencyKey string) (CreatedSMTPCredential, ResponseMeta, error) {
	var result CreatedSMTPCredential
	path, err := platformEnvironmentPath(scope, "/smtp-credentials")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, request, idempotencyKey, &result)
	return result, meta, err
}
func (s *SMTPService) GetCredential(ctx context.Context, scope PlatformScope, credentialID string) (SMTPCredential, ResponseMeta, error) {
	var result SMTPCredential
	path, err := platformEnvironmentPath(scope, "/smtp-credentials/"+segment(credentialID))
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}
func (s *SMTPService) RevokeCredential(ctx context.Context, scope PlatformScope, credentialID, idempotencyKey string) (RevokedSMTPCredential, ResponseMeta, error) {
	var result RevokedSMTPCredential
	path, err := platformEnvironmentPath(scope, "/smtp-credentials/"+segment(credentialID)+"/revoke")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, nil, idempotencyKey, &result)
	return result, meta, err
}
func (s *SMTPService) RevokeCredentialDelete(ctx context.Context, scope PlatformScope, credentialID, idempotencyKey string) (RevokedSMTPCredential, ResponseMeta, error) {
	var result RevokedSMTPCredential
	path, err := platformEnvironmentPath(scope, "/smtp-credentials/"+segment(credentialID))
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodDelete, path, nil, idempotencyKey, &result)
	return result, meta, err
}
func (s *SMTPService) RotateCredential(ctx context.Context, scope PlatformScope, credentialID, idempotencyKey string) (CreatedSMTPCredential, ResponseMeta, error) {
	var result CreatedSMTPCredential
	path, err := platformEnvironmentPath(scope, "/smtp-credentials/"+segment(credentialID)+"/rotate")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, nil, idempotencyKey, &result)
	return result, meta, err
}
func (s *SMTPService) ListSubmissions(ctx context.Context, scope PlatformScope, options SMTPListOptions) (Page[SMTPSubmission], ResponseMeta, error) {
	var result Page[SMTPSubmission]
	path, err := platformEnvironmentPath(scope, "/smtp-submissions")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, withValues(path, options.values()), nil, "", &result)
	return result, meta, err
}
func (s *SMTPService) GetSubmission(ctx context.Context, scope PlatformScope, submissionID string) (SMTPSubmission, ResponseMeta, error) {
	var result SMTPSubmission
	path, err := platformEnvironmentPath(scope, "/smtp-submissions/"+segment(submissionID))
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

type PlatformContactsService struct{ client *Client }

func (s *PlatformContactsService) List(ctx context.Context, scope PlatformScope, options PlatformPageOptions) (Page[PlatformContact], ResponseMeta, error) {
	var result Page[PlatformContact]
	path, err := platformEnvironmentPath(scope, "/contacts")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, platformPathWithQuery(path, options), nil, "", &result)
	return result, meta, err
}
func (s *PlatformContactsService) Create(ctx context.Context, scope PlatformScope, request CreatePlatformContactRequest, idempotencyKeys ...string) (PlatformContact, ResponseMeta, error) {
	var result PlatformContact
	key, err := platformRequiredKey(idempotencyKeys)
	if err != nil {
		return result, ResponseMeta{}, err
	}
	path, err := platformEnvironmentPath(scope, "/contacts")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, request, key, &result)
	return result, meta, err
}
func (s *PlatformContactsService) Get(ctx context.Context, scope PlatformScope, contactID string) (PlatformContact, ResponseMeta, error) {
	var result PlatformContact
	path, err := platformEnvironmentPath(scope, "/contacts/"+segment(contactID))
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}
func (s *PlatformContactsService) Update(ctx context.Context, scope PlatformScope, contactID string, request UpdatePlatformContactRequest) (PlatformContact, ResponseMeta, error) {
	var result PlatformContact
	path, err := platformEnvironmentPath(scope, "/contacts/"+segment(contactID))
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithoutIdempotency(s.client, ctx, http.MethodPatch, path, request, &result)
	return result, meta, err
}
func (s *PlatformContactsService) Transition(ctx context.Context, scope PlatformScope, contactID string, request PlatformContactTransitionRequest) (PlatformContact, ResponseMeta, error) {
	var result PlatformContact
	path, err := platformEnvironmentPath(scope, "/contacts/"+segment(contactID)+"/transition")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithoutIdempotency(s.client, ctx, http.MethodPost, path, request, &result)
	return result, meta, err
}

type PlatformSegmentsService struct{ client *Client }

func (s *PlatformSegmentsService) List(ctx context.Context, scope PlatformScope, options PlatformPageOptions) (Page[PlatformSegment], ResponseMeta, error) {
	var result Page[PlatformSegment]
	path, err := platformEnvironmentPath(scope, "/segments")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, platformPathWithQuery(path, options), nil, "", &result)
	return result, meta, err
}
func (s *PlatformSegmentsService) Create(ctx context.Context, scope PlatformScope, request CreatePlatformSegmentRequest) (PlatformSegment, ResponseMeta, error) {
	var result PlatformSegment
	path, err := platformEnvironmentPath(scope, "/segments")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithoutIdempotency(s.client, ctx, http.MethodPost, path, request, &result)
	return result, meta, err
}
func (s *PlatformSegmentsService) Get(ctx context.Context, scope PlatformScope, segmentID string) (PlatformSegment, ResponseMeta, error) {
	var result PlatformSegment
	path, err := platformEnvironmentPath(scope, "/segments/"+segment(segmentID))
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}
func (s *PlatformSegmentsService) Preview(ctx context.Context, scope PlatformScope, segmentID string) (SegmentPreview, ResponseMeta, error) {
	var result SegmentPreview
	path, err := platformEnvironmentPath(scope, "/segments/"+segment(segmentID)+"/preview")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithoutIdempotency(s.client, ctx, http.MethodPost, path, nil, &result)
	return result, meta, err
}

type PlatformAutomationsService struct{ client *Client }

func (s *PlatformAutomationsService) List(ctx context.Context, scope PlatformScope, options PlatformPageOptions) (Page[PlatformAutomation], ResponseMeta, error) {
	var result Page[PlatformAutomation]
	path, err := platformEnvironmentPath(scope, "/automations")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, platformPathWithQuery(path, options), nil, "", &result)
	return result, meta, err
}
func (s *PlatformAutomationsService) Create(ctx context.Context, scope PlatformScope, request CreatePlatformAutomationRequest) (PlatformAutomation, ResponseMeta, error) {
	var result PlatformAutomation
	path, err := platformEnvironmentPath(scope, "/automations")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithoutIdempotency(s.client, ctx, http.MethodPost, path, request, &result)
	return result, meta, err
}
func (s *PlatformAutomationsService) Get(ctx context.Context, scope PlatformScope, automationID string) (PlatformAutomation, ResponseMeta, error) {
	var result PlatformAutomation
	path, err := platformEnvironmentPath(scope, "/automations/"+segment(automationID))
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}
func (s *PlatformAutomationsService) Transition(ctx context.Context, scope PlatformScope, automationID string, request PlatformAutomationTransitionRequest) (PlatformAutomation, ResponseMeta, error) {
	var result PlatformAutomation
	path, err := platformEnvironmentPath(scope, "/automations/"+segment(automationID)+"/transition")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithoutIdempotency(s.client, ctx, http.MethodPost, path, request, &result)
	return result, meta, err
}
func (s *PlatformAutomationsService) StartRun(ctx context.Context, scope PlatformScope, automationID string, request StartPlatformAutomationRunRequest, idempotencyKey string) (PlatformAutomationRun, ResponseMeta, error) {
	var result PlatformAutomationRun
	path, err := platformEnvironmentPath(scope, "/automations/"+segment(automationID)+"/runs")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, request, idempotencyKey, &result)
	return result, meta, err
}
func (s *PlatformAutomationsService) GetRun(ctx context.Context, scope PlatformScope, runID string) (PlatformAutomationRun, ResponseMeta, error) {
	var result PlatformAutomationRun
	path, err := platformEnvironmentPath(scope, "/automation-runs/"+segment(runID))
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

type PlatformRoutingService struct{ client *Client }

func (s *PlatformRoutingService) GetPolicy(ctx context.Context, scope PlatformScope) (PlatformRoutingPolicy, ResponseMeta, error) {
	var result PlatformRoutingPolicy
	path, err := platformEnvironmentPath(scope, "/routing/policy")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}
func (s *PlatformRoutingService) PutPolicy(ctx context.Context, scope PlatformScope, request PlatformRoutingPolicyRequest, idempotencyKey string) (PlatformRoutingPolicy, ResponseMeta, error) {
	var result PlatformRoutingPolicy
	path, err := platformEnvironmentPath(scope, "/routing/policy")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPut, path, request, idempotencyKey, &result)
	return result, meta, err
}
func (s *PlatformRoutingService) ListPools(ctx context.Context, scope PlatformScope) (CursorPage[RoutingPool], ResponseMeta, error) {
	var result CursorPage[RoutingPool]
	path, err := platformEnvironmentPath(scope, "/routing/pools")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}
func (s *PlatformRoutingService) GetPool(ctx context.Context, scope PlatformScope, poolID string) (RoutingPool, ResponseMeta, error) {
	var result RoutingPool
	path, err := platformEnvironmentPath(scope, "/routing/pools/"+segment(poolID))
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}
func (s *PlatformRoutingService) PutPool(ctx context.Context, scope PlatformScope, poolID string, request RoutingPoolRequest, idempotencyKey string) (RoutingPool, ResponseMeta, error) {
	var result RoutingPool
	path, err := platformEnvironmentPath(scope, "/routing/pools/"+segment(poolID))
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPut, path, request, idempotencyKey, &result)
	return result, meta, err
}
func (s *PlatformRoutingService) PutIP(ctx context.Context, scope PlatformScope, poolID, ipID string, request RoutingIPRequest, idempotencyKey string) (RoutingPool, ResponseMeta, error) {
	var result RoutingPool
	path, err := platformEnvironmentPath(scope, "/routing/pools/"+segment(poolID)+"/ips/"+segment(ipID))
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPut, path, request, idempotencyKey, &result)
	return result, meta, err
}
func (s *PlatformRoutingService) ListProviders(ctx context.Context, scope PlatformScope) (CursorPage[RoutingProvider], ResponseMeta, error) {
	var result CursorPage[RoutingProvider]
	path, err := platformEnvironmentPath(scope, "/routing/providers")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}
func (s *PlatformRoutingService) PutProvider(ctx context.Context, scope PlatformScope, providerID string, request RoutingProviderRequest, idempotencyKey string) (RoutingProvider, ResponseMeta, error) {
	var result RoutingProvider
	path, err := platformEnvironmentPath(scope, "/routing/providers/"+segment(providerID))
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPut, path, request, idempotencyKey, &result)
	return result, meta, err
}
func (s *PlatformRoutingService) ListRoutes(ctx context.Context, scope PlatformScope, providerID string) (CursorPage[RoutingRoute], ResponseMeta, error) {
	var result CursorPage[RoutingRoute]
	path, err := platformEnvironmentPath(scope, "/routing/providers/"+segment(providerID)+"/routes")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}
func (s *PlatformRoutingService) PutRoutes(ctx context.Context, scope PlatformScope, providerID string, request RoutingRoutesRequest, idempotencyKey string) (CursorPage[RoutingRoute], ResponseMeta, error) {
	var result CursorPage[RoutingRoute]
	path, err := platformEnvironmentPath(scope, "/routing/providers/"+segment(providerID)+"/routes")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPut, path, request, idempotencyKey, &result)
	return result, meta, err
}
func (s *PlatformRoutingService) Resolve(ctx context.Context, scope PlatformScope, request ResolveRoutingRequest) (RoutingResolution, ResponseMeta, error) {
	var result RoutingResolution
	path, err := platformEnvironmentPath(scope, "/routing/resolve")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithoutIdempotency(s.client, ctx, http.MethodPost, path, request, &result)
	return result, meta, err
}

type PlatformDomainsService struct{ client *Client }

func (s *PlatformDomainsService) List(ctx context.Context, scope PlatformScope) (Page[PlatformDomain], ResponseMeta, error) {
	var result Page[PlatformDomain]
	path, err := platformEnvironmentPath(scope, "/domains")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}
func (s *PlatformDomainsService) StartOnboarding(ctx context.Context, scope PlatformScope, request StartDomainOnboardingRequest, idempotencyKey string) (DomainOnboarding, ResponseMeta, error) {
	var result DomainOnboarding
	path, err := platformEnvironmentPath(scope, "/domains")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, request, idempotencyKey, &result)
	return result, meta, err
}
func (s *PlatformDomainsService) BeginConnector(ctx context.Context, scope PlatformScope, domainID string, request BeginDomainConnectorRequest, idempotencyKey string) (DomainConnector, ResponseMeta, error) {
	var result DomainConnector
	path, err := platformEnvironmentPath(scope, "/domains/"+segment(domainID)+"/connector")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, request, idempotencyKey, &result)
	return result, meta, err
}
func (s *PlatformDomainsService) CompleteConnector(ctx context.Context, scope PlatformScope, domainID string, request CompleteDomainConnectorRequest, idempotencyKey string) (DomainConnector, ResponseMeta, error) {
	var result DomainConnector
	path, err := platformEnvironmentPath(scope, "/domains/"+segment(domainID)+"/connector/callback")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, request, idempotencyKey, &result)
	return result, meta, err
}
func (s *PlatformDomainsService) Verify(ctx context.Context, scope PlatformScope, domainID, idempotencyKey string) (PlatformDomain, ResponseMeta, error) {
	var result PlatformDomain
	path, err := platformEnvironmentPath(scope, "/domains/"+segment(domainID)+"/verify")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodPost, path, nil, idempotencyKey, &result)
	return result, meta, err
}

type PlatformDomainHealthService struct{ client *Client }
type PlatformDomainHealthListOptions struct {
	Cursor, From, To string
	Limit            int
}

func (o PlatformDomainHealthListOptions) values() url.Values {
	return platformPageValues(PlatformPageOptions{Cursor: o.Cursor, Limit: o.Limit, From: o.From, To: o.To})
}
func (s *PlatformDomainHealthService) Check(ctx context.Context, scope PlatformScope, domainID string) (PlatformDomainHealthReport, ResponseMeta, error) {
	var result PlatformDomainHealthReport
	path, err := platformEnvironmentPath(scope, "/domains/"+segment(domainID)+"/health/check")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithoutIdempotency(s.client, ctx, http.MethodPost, path, nil, &result)
	return result, meta, err
}
func (s *PlatformDomainHealthService) Reports(ctx context.Context, scope PlatformScope, domainID string, options PlatformDomainHealthListOptions) (CursorPage[PlatformDomainHealthReport], ResponseMeta, error) {
	var result CursorPage[PlatformDomainHealthReport]
	path, err := platformEnvironmentPath(scope, "/domains/"+segment(domainID)+"/health/reports")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, withValues(path, options.values()), nil, "", &result)
	return result, meta, err
}

type UsageContractService struct{ client *Client }
type UsageFactListOptions struct {
	From, To, FactType, Family, SourceKind, Cursor string
	Limit                                          int
}

func (o UsageFactListOptions) values() url.Values {
	return platformPageValues(PlatformPageOptions{Cursor: o.Cursor, Limit: o.Limit, From: o.From, To: o.To, Query: url.Values{"fact_type": []string{o.FactType}, "family": []string{o.Family}, "source_kind": []string{o.SourceKind}}})
}
func (s *UsageContractService) ExportFacts(ctx context.Context, scope PlatformScope, options UsageFactListOptions) (CursorPage[UsageFact], ResponseMeta, error) {
	var result CursorPage[UsageFact]
	path, err := platformEnvironmentPath(scope, "/usage/facts")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, withValues(path, options.values()), nil, "", &result)
	return result, meta, err
}
func (s *UsageContractService) CorrectFact(ctx context.Context, scope PlatformScope, factID string, request UsageCorrectionRequest) (UsageFact, ResponseMeta, error) {
	var result UsageFact
	path, err := platformEnvironmentPath(scope, "/usage/facts/"+segment(factID)+"/corrections")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithoutIdempotency(s.client, ctx, http.MethodPost, path, request, &result)
	return result, meta, err
}
func (s *UsageContractService) Vocabulary(ctx context.Context, scope PlatformScope) ([]UsageVocabularyEntry, ResponseMeta, error) {
	var result []UsageVocabularyEntry
	path, err := platformEnvironmentPath(scope, "/usage/vocabulary")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}

type UsageReconciliationListOptions struct {
	Status, Cursor string
	Limit          int
}

func (o UsageReconciliationListOptions) values() url.Values {
	return platformPageValues(PlatformPageOptions{Cursor: o.Cursor, Limit: o.Limit, Status: o.Status})
}
func (s *UsageContractService) ListReconciliations(ctx context.Context, scope PlatformScope, options UsageReconciliationListOptions) (CursorPage[UsageReconciliation], ResponseMeta, error) {
	var result CursorPage[UsageReconciliation]
	path, err := platformEnvironmentPath(scope, "/usage/reconciliations")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, withValues(path, options.values()), nil, "", &result)
	return result, meta, err
}
func (s *UsageContractService) StartReconciliation(ctx context.Context, scope PlatformScope, request StartUsageReconciliationRequest) (UsageReconciliation, ResponseMeta, error) {
	var result UsageReconciliation
	path, err := platformEnvironmentPath(scope, "/usage/reconciliations")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithoutIdempotency(s.client, ctx, http.MethodPost, path, request, &result)
	return result, meta, err
}
func (s *UsageContractService) GetReconciliation(ctx context.Context, scope PlatformScope, reconciliationID string) (UsageReconciliation, ResponseMeta, error) {
	var result UsageReconciliation
	path, err := platformEnvironmentPath(scope, "/usage/reconciliations/"+segment(reconciliationID))
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, path, nil, "", &result)
	return result, meta, err
}
func (s *UsageContractService) FinishReconciliation(ctx context.Context, scope PlatformScope, reconciliationID string, request FinishUsageReconciliationRequest) (UsageReconciliation, ResponseMeta, error) {
	var result UsageReconciliation
	path, err := platformEnvironmentPath(scope, "/usage/reconciliations/"+segment(reconciliationID)+"/finish")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithoutIdempotency(s.client, ctx, http.MethodPost, path, request, &result)
	return result, meta, err
}
func (s *UsageContractService) ListReconciliationItems(ctx context.Context, scope PlatformScope, reconciliationID string, options PlatformPageOptions) (CursorPage[UsageReconciliationItem], ResponseMeta, error) {
	var result CursorPage[UsageReconciliationItem]
	path, err := platformEnvironmentPath(scope, "/usage/reconciliations/"+segment(reconciliationID)+"/items")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTyped(s.client, ctx, http.MethodGet, platformPathWithQuery(path, options), nil, "", &result)
	return result, meta, err
}
func (s *UsageContractService) AppendReconciliationItem(ctx context.Context, scope PlatformScope, reconciliationID string, request UsageReconciliationItemRequest) (CursorPage[UsageReconciliationItem], ResponseMeta, error) {
	var result CursorPage[UsageReconciliationItem]
	path, err := platformEnvironmentPath(scope, "/usage/reconciliations/"+segment(reconciliationID)+"/items")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithoutIdempotency(s.client, ctx, http.MethodPost, path, request, &result)
	return result, meta, err
}

// The following account/environment resources are intentionally small typed
// wrappers for the remaining Phase 1 platform routes.
type PlatformAudiencesService struct{ client *Client }
type CreateAudienceRequest struct {
	Name string `json:"name"`
}
type Audience struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Revision   int               `json:"revision"`
	Definition map[string]string `json:"definition"`
	CreatedAt  string            `json:"createdAt"`
	UpdatedAt  string            `json:"updatedAt"`
}

func (s *PlatformAudiencesService) List(ctx context.Context, scope PlatformScope) (Page[Audience], ResponseMeta, error) {
	var r Page[Audience]
	p, e := platformEnvironmentPath(scope, "/audiences")
	if e != nil {
		return r, ResponseMeta{}, e
	}
	m, e := requestTyped(s.client, ctx, http.MethodGet, p, nil, "", &r)
	return r, m, e
}
func (s *PlatformAudiencesService) Create(ctx context.Context, scope PlatformScope, request CreateAudienceRequest, key string) (Audience, ResponseMeta, error) {
	var r Audience
	p, e := platformEnvironmentPath(scope, "/audiences")
	if e != nil {
		return r, ResponseMeta{}, e
	}
	m, e := requestTyped(s.client, ctx, http.MethodPost, p, request, key, &r)
	return r, m, e
}
func (s *PlatformAudiencesService) BulkIngest(ctx context.Context, scope PlatformScope, audienceID string, contacts []map[string]any, key string) (BulkIngestResult, ResponseMeta, error) {
	var r BulkIngestResult
	p, e := platformEnvironmentPath(scope, "/audiences/"+segment(audienceID)+"/contacts/bulk-ingest")
	if e != nil {
		return r, ResponseMeta{}, e
	}
	m, e := requestTyped(s.client, ctx, http.MethodPost, p, map[string]any{"contacts": contacts}, key, &r)
	return r, m, e
}

type PlatformBroadcastsService struct{ client *Client }
type PlatformBroadcast struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	AudienceReference string `json:"audienceReference"`
	AudienceSourceID  string `json:"audienceSourceId"`
	EnvironmentID     string `json:"environmentId"`
	SenderIdentityID  string `json:"senderIdentityId"`
	Status            string `json:"status"`
	ScheduledAt       string `json:"scheduledAt"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
	EstimatedTargets  int    `json:"estimatedTargets"`
}
type CreatePlatformBroadcastRequest struct {
	Name              string         `json:"name"`
	AudienceReference string         `json:"audienceReference,omitempty"`
	AudienceSourceID  string         `json:"audienceSourceId,omitempty"`
	EnvironmentID     string         `json:"environmentId,omitempty"`
	SenderIdentityID  string         `json:"senderIdentityId,omitempty"`
	Message           map[string]any `json:"message"`
}
type SchedulePlatformBroadcastRequest struct {
	ScheduledAt string `json:"scheduledAt"`
}
type BroadcastPreflight struct {
	BroadcastID, Status, CompletedAt    string
	EstimatedTargets, SuppressedTargets int
}

func (s *PlatformBroadcastsService) List(ctx context.Context, accountID string, options PlatformPageOptions) (Page[PlatformBroadcast], ResponseMeta, error) {
	var r Page[PlatformBroadcast]
	p, e := platformAccountPath(AccountScope{accountID}, "/broadcasts")
	if e != nil {
		return r, ResponseMeta{}, e
	}
	m, e := requestTyped(s.client, ctx, http.MethodGet, platformPathWithQuery(p, options), nil, "", &r)
	return r, m, e
}
func (s *PlatformBroadcastsService) Create(ctx context.Context, accountID string, request CreatePlatformBroadcastRequest, key string) (PlatformBroadcast, ResponseMeta, error) {
	var r PlatformBroadcast
	p, e := platformAccountPath(AccountScope{accountID}, "/broadcasts")
	if e != nil {
		return r, ResponseMeta{}, e
	}
	m, e := requestTyped(s.client, ctx, http.MethodPost, p, request, key, &r)
	return r, m, e
}
func (s *PlatformBroadcastsService) Get(ctx context.Context, accountID, broadcastID string) (PlatformBroadcast, ResponseMeta, error) {
	var r PlatformBroadcast
	p, e := platformAccountPath(AccountScope{accountID}, "/broadcasts/"+segment(broadcastID))
	if e != nil {
		return r, ResponseMeta{}, e
	}
	m, e := requestTyped(s.client, ctx, http.MethodGet, p, nil, "", &r)
	return r, m, e
}
func (s *PlatformBroadcastsService) Preflight(ctx context.Context, accountID, broadcastID, key string) (BroadcastPreflight, ResponseMeta, error) {
	var r BroadcastPreflight
	p, e := platformAccountPath(AccountScope{accountID}, "/broadcasts/"+segment(broadcastID)+"/preflight")
	if e != nil {
		return r, ResponseMeta{}, e
	}
	m, e := requestTyped(s.client, ctx, http.MethodPost, p, nil, key, &r)
	return r, m, e
}
func (s *PlatformBroadcastsService) Approve(ctx context.Context, accountID, broadcastID, key string) (PlatformBroadcast, ResponseMeta, error) {
	var r PlatformBroadcast
	p, e := platformAccountPath(AccountScope{accountID}, "/broadcasts/"+segment(broadcastID)+"/approve")
	if e != nil {
		return r, ResponseMeta{}, e
	}
	m, e := requestTyped(s.client, ctx, http.MethodPost, p, nil, key, &r)
	return r, m, e
}
func (s *PlatformBroadcastsService) Schedule(ctx context.Context, accountID, broadcastID string, request SchedulePlatformBroadcastRequest, key string) (PlatformBroadcast, ResponseMeta, error) {
	var r PlatformBroadcast
	p, e := platformAccountPath(AccountScope{accountID}, "/broadcasts/"+segment(broadcastID)+"/schedule")
	if e != nil {
		return r, ResponseMeta{}, e
	}
	m, e := requestTyped(s.client, ctx, http.MethodPost, p, request, key, &r)
	return r, m, e
}

func (s *PlatformBroadcastsService) Snapshot(ctx context.Context, scope PlatformScope, broadcastID, segmentID string) (BroadcastSnapshot, ResponseMeta, error) {
	var result BroadcastSnapshot
	path, err := platformEnvironmentPath(scope, "/broadcasts/"+segment(broadcastID)+"/segments/"+segment(segmentID)+"/snapshot")
	if err != nil {
		return result, ResponseMeta{}, err
	}
	meta, err := requestTypedWithoutIdempotency(s.client, ctx, http.MethodPost, path, nil, &result)
	return result, meta, err
}

type PlatformProviderBindingsService struct{ client *Client }
type ProviderBinding struct {
	ID                 string `json:"id"`
	Provider           string `json:"provider"`
	ProviderAccountRef string `json:"providerAccountRef"`
	SourceKind         string `json:"sourceKind"`
	Status             string `json:"status"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
}
type CreateProviderBindingRequest struct {
	Provider           string `json:"provider"`
	ProviderAccountRef string `json:"providerAccountRef"`
	SourceKind         string `json:"sourceKind"`
}

func (s *PlatformProviderBindingsService) List(ctx context.Context, scope PlatformScope) (Page[ProviderBinding], ResponseMeta, error) {
	var r Page[ProviderBinding]
	p, e := platformEnvironmentPath(scope, "/provider-bindings")
	if e != nil {
		return r, ResponseMeta{}, e
	}
	m, e := requestTyped(s.client, ctx, http.MethodGet, p, nil, "", &r)
	return r, m, e
}
func (s *PlatformProviderBindingsService) Create(ctx context.Context, scope PlatformScope, request CreateProviderBindingRequest, key string) (ProviderBinding, ResponseMeta, error) {
	var r ProviderBinding
	p, e := platformEnvironmentPath(scope, "/provider-bindings")
	if e != nil {
		return r, ResponseMeta{}, e
	}
	m, e := requestTyped(s.client, ctx, http.MethodPost, p, request, key, &r)
	return r, m, e
}
