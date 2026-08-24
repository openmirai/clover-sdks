package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

// PlatformScope identifies the account and environment used by the clean
// Phase 1 API.  It deliberately does not carry the legacy organization,
// project, or tenant fields; those values were part of the retired route
// family and must never be sent to /platform routes.
type PlatformScope struct {
	AccountID     string
	EnvironmentID string
}

func platformScopeFromValues(accountID, environmentID string) (PlatformScope, error) {
	scope := PlatformScope{AccountID: strings.TrimSpace(accountID), EnvironmentID: strings.TrimSpace(environmentID)}
	if scope.AccountID == "" {
		return PlatformScope{}, errors.New("account-id is required")
	}
	if scope.EnvironmentID == "" {
		return PlatformScope{}, errors.New("environment-id is required")
	}
	return scope, nil
}

func (scope PlatformScope) path(suffix string) string {
	return "/api/v1/platform/accounts/" + url.PathEscape(scope.AccountID) + "/environments/" + url.PathEscape(scope.EnvironmentID) + suffix
}

func platformAccountPath(accountID, suffix string) string {
	return "/api/v1/platform/accounts/" + url.PathEscape(strings.TrimSpace(accountID)) + suffix
}

func (c *Client) platformRequest(ctx context.Context, method, path string, payload any, key string, requireKey bool) (json.RawMessage, ResponseMeta, error) {
	// Scoped paths are assembled before entering the shared transport helper.
	// Keep validation here as a second line of defense for library callers that
	// construct PlatformScope directly instead of using the CLI flag parser.
	if strings.Contains(path, "/platform/accounts//") {
		return nil, ResponseMeta{}, errors.New("account-id is required")
	}
	if strings.Contains(path, "/environments//") {
		return nil, ResponseMeta{}, errors.New("environment-id is required")
	}
	return c.requestWithHeadersRequirement(ctx, method, path, payload, key, nil, requireKey)
}

func (c *Client) PlatformListAccounts(ctx context.Context, query url.Values) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, withQuery("/api/v1/platform/accounts", query), nil, "", false)
}

func (c *Client) PlatformCreateAccount(ctx context.Context, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, "/api/v1/platform/accounts", payload, key, true)
}

func (c *Client) PlatformGetAccount(ctx context.Context, accountID string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, platformAccountPath(accountID, ""), nil, "", false)
}

func (c *Client) PlatformUpdateAccount(ctx context.Context, accountID string, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPatch, platformAccountPath(accountID, ""), payload, key, true)
}

func (c *Client) PlatformListEnvironments(ctx context.Context, accountID string, query url.Values) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, withQuery(platformAccountPath(accountID, "/environments"), query), nil, "", false)
}

func (c *Client) PlatformCreateEnvironment(ctx context.Context, accountID string, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, platformAccountPath(accountID, "/environments"), payload, key, true)
}

func (c *Client) PlatformSend(ctx context.Context, scope PlatformScope, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/messages"), payload, key, true)
}

func (c *Client) PlatformSendBatch(ctx context.Context, scope PlatformScope, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/messages/batch"), payload, key, true)
}

func (c *Client) PlatformListMessages(ctx context.Context, scope PlatformScope, query url.Values) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, withQuery(scope.path("/messages"), query), nil, "", false)
}

func (c *Client) PlatformGetMessage(ctx context.Context, scope PlatformScope, id string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, scope.path("/messages/"+url.PathEscape(id)), nil, "", false)
}

func (c *Client) PlatformScheduleMessage(ctx context.Context, scope PlatformScope, id string, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/messages/"+url.PathEscape(id)+"/schedule"), payload, key, true)
}

func (c *Client) PlatformCancelMessage(ctx context.Context, scope PlatformScope, id, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/messages/"+url.PathEscape(id)+"/cancel"), nil, key, true)
}

func (c *Client) PlatformListTemplates(ctx context.Context, scope PlatformScope, query url.Values) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, withQuery(scope.path("/templates"), query), nil, "", false)
}

func (c *Client) PlatformCreateTemplate(ctx context.Context, scope PlatformScope, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/templates"), payload, key, true)
}

func (c *Client) PlatformGetTemplate(ctx context.Context, scope PlatformScope, id string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, scope.path("/templates/"+url.PathEscape(id)), nil, "", false)
}

func (c *Client) PlatformUpdateTemplate(ctx context.Context, scope PlatformScope, id string, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPatch, scope.path("/templates/"+url.PathEscape(id)), payload, key, true)
}

func (c *Client) PlatformTransitionTemplate(ctx context.Context, scope PlatformScope, id, action, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/templates/"+url.PathEscape(id)+"/"+url.PathEscape(action)), nil, key, true)
}

func (c *Client) PlatformListTemplateVersions(ctx context.Context, scope PlatformScope, templateID string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, scope.path("/templates/"+url.PathEscape(templateID)+"/versions"), nil, "", false)
}

func (c *Client) PlatformCreateTemplateVersion(ctx context.Context, scope PlatformScope, templateID string, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/templates/"+url.PathEscape(templateID)+"/versions"), payload, key, true)
}

func (c *Client) PlatformGetTemplateVersion(ctx context.Context, scope PlatformScope, templateID, versionRef string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, scope.path("/templates/"+url.PathEscape(templateID)+"/versions/"+url.PathEscape(versionRef)), nil, "", false)
}

func (c *Client) PlatformCompareTemplateVersions(ctx context.Context, scope PlatformScope, templateID, from, to string) (json.RawMessage, ResponseMeta, error) {
	query := url.Values{"from": {from}, "to": {to}}
	return c.platformRequest(ctx, http.MethodGet, withQuery(scope.path("/templates/"+url.PathEscape(templateID)+"/versions/compare"), query), nil, "", false)
}

func (c *Client) PlatformPublishTemplateVersion(ctx context.Context, scope PlatformScope, templateID, versionRef, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/templates/"+url.PathEscape(templateID)+"/versions/"+url.PathEscape(versionRef)+"/publish"), nil, key, true)
}

func (c *Client) PlatformRenderTemplateVersion(ctx context.Context, scope PlatformScope, templateID, versionRef string, payload json.RawMessage) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/templates/"+url.PathEscape(templateID)+"/versions/"+url.PathEscape(versionRef)+"/render"), payload, "", false)
}

func (c *Client) PlatformRollbackTemplateVersion(ctx context.Context, scope PlatformScope, templateID, versionRef, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/templates/"+url.PathEscape(templateID)+"/versions/"+url.PathEscape(versionRef)+"/rollback"), nil, key, true)
}

func (c *Client) PlatformListWebhooks(ctx context.Context, scope PlatformScope, query url.Values) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, withQuery(scope.path("/webhooks"), query), nil, "", false)
}

func (c *Client) PlatformCreateWebhook(ctx context.Context, scope PlatformScope, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/webhooks"), payload, key, true)
}

func (c *Client) PlatformGetWebhook(ctx context.Context, scope PlatformScope, id string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, scope.path("/webhooks/"+url.PathEscape(id)), nil, "", false)
}

func (c *Client) PlatformUpdateWebhook(ctx context.Context, scope PlatformScope, id string, payload json.RawMessage) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPatch, scope.path("/webhooks/"+url.PathEscape(id)), payload, "", false)
}

func (c *Client) PlatformDeleteWebhook(ctx context.Context, scope PlatformScope, id string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodDelete, scope.path("/webhooks/"+url.PathEscape(id)), nil, "", false)
}

func (c *Client) PlatformRotateWebhookSecret(ctx context.Context, scope PlatformScope, id string, payload json.RawMessage) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/webhooks/"+url.PathEscape(id)+"/rotate-secret"), payload, "", false)
}

func (c *Client) PlatformListWebhookDeliveries(ctx context.Context, scope PlatformScope, query url.Values) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, withQuery(scope.path("/webhook-deliveries"), query), nil, "", false)
}

func (c *Client) PlatformGetWebhookDelivery(ctx context.Context, scope PlatformScope, id string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, scope.path("/webhook-deliveries/"+url.PathEscape(id)), nil, "", false)
}

func (c *Client) PlatformReplayWebhookDelivery(ctx context.Context, scope PlatformScope, id string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/webhook-deliveries/"+url.PathEscape(id)+"/replay"), map[string]any{}, "", false)
}

func (c *Client) PlatformListMessageTimeline(ctx context.Context, scope PlatformScope, query url.Values) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, withQuery(scope.path("/message-timeline"), query), nil, "", false)
}

func (c *Client) PlatformListPreferenceTopics(ctx context.Context, scope PlatformScope) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, scope.path("/preference-topics"), nil, "", false)
}

func (c *Client) PlatformCreatePreferenceTopic(ctx context.Context, scope PlatformScope, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/preference-topics"), payload, key, true)
}

func (c *Client) PlatformCreatePreferenceToken(ctx context.Context, scope PlatformScope, payload json.RawMessage) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/preference-tokens"), payload, "", false)
}

func (c *Client) PlatformListSuppressions(ctx context.Context, scope PlatformScope, query url.Values) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, withQuery(scope.path("/suppressions"), query), nil, "", false)
}

func (c *Client) PlatformCreateSuppression(ctx context.Context, scope PlatformScope, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/suppressions"), payload, key, true)
}

func (c *Client) PlatformDeleteSuppression(ctx context.Context, scope PlatformScope, id string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodDelete, scope.path("/suppressions/"+url.PathEscape(id)), nil, "", false)
}

func (c *Client) PlatformReactivateSuppression(ctx context.Context, scope PlatformScope, id string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/suppressions/"+url.PathEscape(id)+"/reactivate"), nil, "", false)
}

func (c *Client) PlatformListReceivedMessages(ctx context.Context, scope PlatformScope, query url.Values) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, withQuery(scope.path("/received-messages"), query), nil, "", false)
}

func (c *Client) PlatformGetReceivedMessage(ctx context.Context, scope PlatformScope, id string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, scope.path("/received-messages/"+url.PathEscape(id)), nil, "", false)
}

func (c *Client) PlatformGetReceivedAttachment(ctx context.Context, scope PlatformScope, messageID, attachmentID string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, scope.path("/received-messages/"+url.PathEscape(messageID)+"/attachments/"+url.PathEscape(attachmentID)), nil, "", false)
}

func (c *Client) PlatformListSMTPCredentials(ctx context.Context, scope PlatformScope, query url.Values) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, withQuery(scope.path("/smtp-credentials"), query), nil, "", false)
}

func (c *Client) PlatformCreateSMTPCredential(ctx context.Context, scope PlatformScope, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/smtp-credentials"), payload, key, true)
}

func (c *Client) PlatformGetSMTPCredential(ctx context.Context, scope PlatformScope, id string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, scope.path("/smtp-credentials/"+url.PathEscape(id)), nil, "", false)
}

func (c *Client) PlatformDeleteSMTPCredential(ctx context.Context, scope PlatformScope, id, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodDelete, scope.path("/smtp-credentials/"+url.PathEscape(id)), nil, key, true)
}

func (c *Client) PlatformRevokeSMTPCredential(ctx context.Context, scope PlatformScope, id, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/smtp-credentials/"+url.PathEscape(id)+"/revoke"), nil, key, true)
}

func (c *Client) PlatformRotateSMTPCredential(ctx context.Context, scope PlatformScope, id, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/smtp-credentials/"+url.PathEscape(id)+"/rotate"), nil, key, true)
}

func (c *Client) PlatformListSMTPSubmissions(ctx context.Context, scope PlatformScope, query url.Values) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, withQuery(scope.path("/smtp-submissions"), query), nil, "", false)
}

func (c *Client) PlatformGetSMTPSubmission(ctx context.Context, scope PlatformScope, id string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, scope.path("/smtp-submissions/"+url.PathEscape(id)), nil, "", false)
}

func (c *Client) PlatformListContacts(ctx context.Context, scope PlatformScope, query url.Values) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, withQuery(scope.path("/contacts"), query), nil, "", false)
}

func (c *Client) PlatformCreateContact(ctx context.Context, scope PlatformScope, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/contacts"), payload, key, true)
}

func (c *Client) PlatformGetContact(ctx context.Context, scope PlatformScope, id string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, scope.path("/contacts/"+url.PathEscape(id)), nil, "", false)
}

func (c *Client) PlatformUpdateContact(ctx context.Context, scope PlatformScope, id string, payload json.RawMessage) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPatch, scope.path("/contacts/"+url.PathEscape(id)), payload, "", false)
}

func (c *Client) PlatformTransitionContact(ctx context.Context, scope PlatformScope, id string, payload json.RawMessage) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/contacts/"+url.PathEscape(id)+"/transition"), payload, "", false)
}

func (c *Client) PlatformListSegments(ctx context.Context, scope PlatformScope, query url.Values) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, withQuery(scope.path("/segments"), query), nil, "", false)
}

func (c *Client) PlatformCreateSegment(ctx context.Context, scope PlatformScope, payload json.RawMessage) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/segments"), payload, "", false)
}

func (c *Client) PlatformGetSegment(ctx context.Context, scope PlatformScope, id string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, scope.path("/segments/"+url.PathEscape(id)), nil, "", false)
}

func (c *Client) PlatformPreviewSegment(ctx context.Context, scope PlatformScope, id string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/segments/"+url.PathEscape(id)+"/preview"), nil, "", false)
}

func (c *Client) PlatformListAutomations(ctx context.Context, scope PlatformScope, query url.Values) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, withQuery(scope.path("/automations"), query), nil, "", false)
}

func (c *Client) PlatformCreateAutomation(ctx context.Context, scope PlatformScope, payload json.RawMessage) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/automations"), payload, "", false)
}

func (c *Client) PlatformGetAutomation(ctx context.Context, scope PlatformScope, id string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, scope.path("/automations/"+url.PathEscape(id)), nil, "", false)
}

func (c *Client) PlatformTransitionAutomation(ctx context.Context, scope PlatformScope, id string, payload json.RawMessage) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/automations/"+url.PathEscape(id)+"/transition"), payload, "", false)
}

func (c *Client) PlatformStartAutomationRun(ctx context.Context, scope PlatformScope, id string, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/automations/"+url.PathEscape(id)+"/runs"), payload, key, true)
}

func (c *Client) PlatformGetAutomationRun(ctx context.Context, scope PlatformScope, runID string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, scope.path("/automation-runs/"+url.PathEscape(runID)), nil, "", false)
}

func (c *Client) PlatformListDomainHealthReports(ctx context.Context, scope PlatformScope, domainID string, query url.Values) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, withQuery(scope.path("/domains/"+url.PathEscape(domainID)+"/health/reports"), query), nil, "", false)
}

func (c *Client) PlatformCheckDomainHealth(ctx context.Context, scope PlatformScope, domainID string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/domains/"+url.PathEscape(domainID)+"/health/check"), nil, "", false)
}

func (c *Client) PlatformListDomains(ctx context.Context, scope PlatformScope) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, scope.path("/domains"), nil, "", false)
}

func (c *Client) PlatformStartDomain(ctx context.Context, scope PlatformScope, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/domains"), payload, key, true)
}

func (c *Client) PlatformVerifyDomain(ctx context.Context, scope PlatformScope, domainID string, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/domains/"+url.PathEscape(domainID)+"/verify"), nil, key, true)
}

func (c *Client) PlatformGetDomainReceiving(ctx context.Context, scope PlatformScope, domainID string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, scope.path("/domains/"+url.PathEscape(domainID)+"/receiving"), nil, "", false)
}

func (c *Client) PlatformSetDomainReceiving(ctx context.Context, scope PlatformScope, domainID string, payload json.RawMessage) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPut, scope.path("/domains/"+url.PathEscape(domainID)+"/receiving"), payload, "", false)
}

func (c *Client) PlatformDeleteDomainReceiving(ctx context.Context, scope PlatformScope, domainID string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodDelete, scope.path("/domains/"+url.PathEscape(domainID)+"/receiving"), nil, "", false)
}

func (c *Client) PlatformGetRoutingPolicy(ctx context.Context, scope PlatformScope) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, scope.path("/routing/policy"), nil, "", false)
}

func (c *Client) PlatformPutRoutingPolicy(ctx context.Context, scope PlatformScope, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPut, scope.path("/routing/policy"), payload, key, true)
}

func (c *Client) PlatformListRoutingPools(ctx context.Context, scope PlatformScope) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, scope.path("/routing/pools"), nil, "", false)
}

func (c *Client) PlatformGetRoutingPool(ctx context.Context, scope PlatformScope, id string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, scope.path("/routing/pools/"+url.PathEscape(id)), nil, "", false)
}

func (c *Client) PlatformPutRoutingPool(ctx context.Context, scope PlatformScope, id string, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPut, scope.path("/routing/pools/"+url.PathEscape(id)), payload, key, true)
}

func (c *Client) PlatformPutRoutingIP(ctx context.Context, scope PlatformScope, poolID, ipID string, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPut, scope.path("/routing/pools/"+url.PathEscape(poolID)+"/ips/"+url.PathEscape(ipID)), payload, key, true)
}

func (c *Client) PlatformListRoutingProviders(ctx context.Context, scope PlatformScope) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, scope.path("/routing/providers"), nil, "", false)
}

func (c *Client) PlatformPutRoutingProvider(ctx context.Context, scope PlatformScope, id string, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPut, scope.path("/routing/providers/"+url.PathEscape(id)), payload, key, true)
}

func (c *Client) PlatformListRoutingProviderRoutes(ctx context.Context, scope PlatformScope, providerID string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, scope.path("/routing/providers/"+url.PathEscape(providerID)+"/routes"), nil, "", false)
}

func (c *Client) PlatformPutRoutingProviderRoutes(ctx context.Context, scope PlatformScope, providerID string, payload json.RawMessage, key string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPut, scope.path("/routing/providers/"+url.PathEscape(providerID)+"/routes"), payload, key, true)
}

func (c *Client) PlatformResolveRouting(ctx context.Context, scope PlatformScope, payload json.RawMessage) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/routing/resolve"), payload, "", false)
}

func (c *Client) PlatformListUsageFacts(ctx context.Context, scope PlatformScope, query url.Values) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, withQuery(scope.path("/usage/facts"), query), nil, "", false)
}

func (c *Client) PlatformGetUsageVocabulary(ctx context.Context, scope PlatformScope) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, scope.path("/usage/vocabulary"), nil, "", false)
}

func (c *Client) PlatformCorrectUsageFact(ctx context.Context, scope PlatformScope, factID string, payload json.RawMessage) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/usage/facts/"+url.PathEscape(factID)+"/corrections"), payload, "", false)
}

func (c *Client) PlatformListReconciliations(ctx context.Context, scope PlatformScope, query url.Values) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, withQuery(scope.path("/usage/reconciliations"), query), nil, "", false)
}

func (c *Client) PlatformStartReconciliation(ctx context.Context, scope PlatformScope, payload json.RawMessage) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/usage/reconciliations"), payload, "", false)
}

func (c *Client) PlatformGetReconciliation(ctx context.Context, scope PlatformScope, id string) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, scope.path("/usage/reconciliations/"+url.PathEscape(id)), nil, "", false)
}

func (c *Client) PlatformFinishReconciliation(ctx context.Context, scope PlatformScope, id string, payload json.RawMessage) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/usage/reconciliations/"+url.PathEscape(id)+"/finish"), payload, "", false)
}

func (c *Client) PlatformListReconciliationItems(ctx context.Context, scope PlatformScope, id string, query url.Values) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodGet, withQuery(scope.path("/usage/reconciliations/"+url.PathEscape(id)+"/items"), query), nil, "", false)
}

func (c *Client) PlatformAddReconciliationItem(ctx context.Context, scope PlatformScope, id string, payload json.RawMessage) (json.RawMessage, ResponseMeta, error) {
	return c.platformRequest(ctx, http.MethodPost, scope.path("/usage/reconciliations/"+url.PathEscape(id)+"/items"), payload, "", false)
}
