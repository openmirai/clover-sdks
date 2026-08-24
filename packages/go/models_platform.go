package clover

// This file contains the account/environment scoped Phase 1 contract.  The
// older models in models.go remain available for source compatibility, but
// new resources intentionally use AccountID and EnvironmentID instead of the
// former tenant/project/environment query scope.

import (
	"encoding/json"
	"net/url"
)

// PlatformScope selects one client account environment.  Both identifiers
// are placed in the path; they are never emitted as tenant/project query
// parameters by the platform services.
type PlatformScope struct {
	AccountID     string
	EnvironmentID string
}

// AccountScope selects account-level platform resources.
type AccountScope struct{ AccountID string }

// PlatformPageOptions is shared by account/environment list endpoints. Query
// is copied before use so callers may safely reuse a url.Values instance.
type PlatformPageOptions struct {
	Page    int
	Limit   int
	Cursor  string
	Status  string
	Enabled *bool
	From    string
	To      string
	Query   url.Values
}

type ClientAccount struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	ExternalReference string `json:"externalReference"`
	Status            string `json:"status"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
}

type CreateClientAccountRequest struct {
	Name              string `json:"name"`
	ExternalReference string `json:"externalReference,omitempty"`
}

type UpdateClientAccountRequest struct {
	Name string `json:"name"`
}

type ClientEnvironment struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	CreatedAt string `json:"createdAt"`
}

type CreateEnvironmentRequest struct {
	Name string `json:"name"`
}

type PlatformAddress struct {
	Address string `json:"address"`
	Name    string `json:"name,omitempty"`
}

type PlatformAttachment struct {
	ContentID   string `json:"contentId,omitempty"`
	ContentType string `json:"contentType,omitempty"`
	Disposition string `json:"disposition,omitempty"`
	Filename    string `json:"filename,omitempty"`
	ObjectKey   string `json:"objectKey,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
	SizeBytes   int64  `json:"sizeBytes,omitempty"`
}

type SendMessageRequest struct {
	From        PlatformAddress      `json:"from"`
	To          []PlatformAddress    `json:"to"`
	CC          []PlatformAddress    `json:"cc,omitempty"`
	BCC         []PlatformAddress    `json:"bcc,omitempty"`
	ReplyTo     []PlatformAddress    `json:"replyTo,omitempty"`
	Subject     string               `json:"subject"`
	HTML        string               `json:"html,omitempty"`
	Text        string               `json:"text,omitempty"`
	Attachments []PlatformAttachment `json:"attachments,omitempty"`
	Headers     map[string]string    `json:"headers,omitempty"`
	Metadata    map[string]any       `json:"metadata,omitempty"`
	Tags        map[string]string    `json:"tags,omitempty"`
	ScheduledAt string               `json:"scheduledAt,omitempty"`
}

type SendMessageBatchRequest struct {
	Items []SendMessageRequest `json:"items"`
}

type ScheduleMessageRequest struct {
	ScheduledAt string `json:"scheduledAt"`
}

type MessageAccepted struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	ScheduledAt string `json:"scheduledAt"`
	RequestID   string `json:"requestId"`
}

type BatchAcceptedItem struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type BatchAccepted struct {
	Items     []BatchAcceptedItem `json:"items"`
	RequestID string              `json:"requestId"`
}

type MessageSummary struct {
	ID                string          `json:"id"`
	EnvironmentID     string          `json:"environmentId"`
	From              PlatformAddress `json:"from"`
	Subject           string          `json:"subject"`
	Status            string          `json:"status"`
	ToCount           int             `json:"toCount"`
	RequestID         string          `json:"requestId"`
	ProviderMessageID string          `json:"providerMessageId"`
	ScheduledAt       string          `json:"scheduledAt"`
	CreatedAt         string          `json:"createdAt"`
	UpdatedAt         string          `json:"updatedAt"`
}

type MessageRecipient struct {
	ID      string `json:"id"`
	Address string `json:"address"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
}

type MessageAttempt struct {
	ID                int    `json:"id"`
	AttemptNumber     int    `json:"attemptNumber"`
	AttemptedAt       string `json:"attemptedAt"`
	ErrorCode         string `json:"errorCode"`
	Outcome           string `json:"outcome"`
	ProviderBindingID string `json:"providerBindingId"`
	ProviderMessageID string `json:"providerMessageId"`
}

type MessageEvent struct {
	ID         int            `json:"id"`
	OccurredAt string         `json:"occurredAt"`
	Payload    map[string]any `json:"payload,omitempty"`
	Type       string         `json:"type"`
}

type MessageDetail struct {
	MessageSummary
	Attachments []PlatformAttachment `json:"attachments"`
	Attempts    []MessageAttempt     `json:"attempts"`
	Events      []MessageEvent       `json:"events"`
	HTML        string               `json:"html"`
	Text        string               `json:"text"`
	Headers     map[string]string    `json:"headers"`
	Metadata    map[string]any       `json:"metadata"`
	Recipients  []MessageRecipient   `json:"recipients"`
	ReplyTo     []PlatformAddress    `json:"replyTo"`
}

type PlatformTemplate struct {
	ID               string `json:"id"`
	AccountID        string `json:"accountId"`
	EnvironmentID    string `json:"environmentId"`
	Key              string `json:"key"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	Status           string `json:"status"`
	CurrentVersion   int    `json:"currentVersion"`
	CurrentVersionID string `json:"currentVersionId"`
	ArchivedAt       string `json:"archivedAt"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
}

type CreatePlatformTemplateRequest struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type UpdatePlatformTemplateRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type CreatePlatformTemplateVersionRequest struct {
	Compiler           string         `json:"compiler,omitempty"`
	CompilerVersion    string         `json:"compiler_version,omitempty"`
	ComponentMetadata  map[string]any `json:"component_metadata,omitempty"`
	ComponentRefs      []string       `json:"component_refs,omitempty"`
	DerivedFromVersion int            `json:"derived_from_version,omitempty"`
	HTML               string         `json:"html,omitempty"`
	PreviewData        map[string]any `json:"preview_data,omitempty"`
	SourceDigest       string         `json:"source_digest,omitempty"`
	SourceFormat       string         `json:"source_format,omitempty"`
	Subject            string         `json:"subject,omitempty"`
	Text               string         `json:"text,omitempty"`
}

type PlatformTemplateVersion struct {
	ID                 string         `json:"id"`
	TemplateID         string         `json:"templateId"`
	AccountID          string         `json:"accountId"`
	EnvironmentID      string         `json:"environmentId"`
	Number             int            `json:"number"`
	Status             string         `json:"status"`
	Compiler           string         `json:"compiler"`
	CompilerVersion    string         `json:"compilerVersion"`
	SourceFormat       string         `json:"sourceFormat"`
	SourceDigest       string         `json:"sourceDigest"`
	HTML               string         `json:"html"`
	Text               string         `json:"text"`
	Subject            string         `json:"subject"`
	Variables          []string       `json:"variables"`
	ComponentRefs      []string       `json:"componentRefs"`
	ComponentMetadata  map[string]any `json:"componentMetadata"`
	PreviewData        map[string]any `json:"previewData"`
	DerivedFromVersion int            `json:"derivedFromVersion"`
	CreatedBy          string         `json:"createdBy"`
	CreatedAt          string         `json:"createdAt"`
	PublishedAt        string         `json:"publishedAt"`
	ArchivedAt         string         `json:"archivedAt"`
}

type TemplateComparison struct {
	TemplateID               string   `json:"templateId"`
	From                     int      `json:"from"`
	To                       int      `json:"to"`
	AddedVariables           []string `json:"addedVariables"`
	RemovedVariables         []string `json:"removedVariables"`
	ChangedFields            []string `json:"changedFields"`
	SourceDigestChanged      bool     `json:"sourceDigestChanged"`
	RenderedHTMLChanged      bool     `json:"renderedHtmlChanged"`
	RenderedTextChanged      bool     `json:"renderedTextChanged"`
	PreviewDataChanged       bool     `json:"previewDataChanged"`
	ComponentMetadataChanged bool     `json:"componentMetadataChanged"`
}

type RenderTemplateRequest struct {
	Values map[string]string `json:"values"`
}

type PinnedTemplateReference struct {
	TemplateID   string `json:"templateId"`
	VersionID    string `json:"versionId"`
	Version      int    `json:"version"`
	SourceDigest string `json:"sourceDigest"`
}

type RenderedTemplate struct {
	HTML      string                  `json:"html"`
	Text      string                  `json:"text"`
	Subject   string                  `json:"subject"`
	Reference PinnedTemplateReference `json:"reference"`
}

type ScopedWebhook struct {
	ID            string   `json:"id"`
	AccountID     string   `json:"client_account_id"`
	EnvironmentID string   `json:"environment_id"`
	URL           string   `json:"url"`
	Description   string   `json:"description"`
	Subscriptions []string `json:"subscriptions"`
	Enabled       bool     `json:"enabled"`
	FailureCount  int      `json:"failure_count"`
	LastSuccessAt string   `json:"last_success_at"`
	LastFailureAt string   `json:"last_failure_at"`
	DisabledAt    string   `json:"disabled_at"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

type CreateScopedWebhookRequest struct {
	URL           string   `json:"url"`
	Description   string   `json:"description,omitempty"`
	Subscriptions []string `json:"subscriptions,omitempty"`
	Enabled       bool     `json:"enabled"`
}

type UpdateScopedWebhookRequest = CreateScopedWebhookRequest

type CreatedScopedWebhook struct {
	Endpoint ScopedWebhook `json:"endpoint"`
	Secret   string        `json:"secret"`
}

type RotatedScopedWebhookSecret struct {
	EndpointID   string `json:"endpoint_id"`
	OverlapUntil string `json:"overlap_until"`
	Secret       string `json:"secret"`
}

type ScopedWebhookDelivery struct {
	ID             string `json:"id"`
	EndpointID     string `json:"endpoint_id"`
	EventID        string `json:"event_id"`
	EventType      string `json:"event_type"`
	Status         string `json:"status"`
	AttemptCount   int    `json:"attempt_count"`
	LastHTTPStatus int    `json:"last_http_status"`
	LastErrorCode  string `json:"last_error_code"`
	NextAttemptAt  string `json:"next_attempt_at"`
	DeliveredAt    string `json:"delivered_at"`
	ReplayOf       string `json:"replay_of"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type MessageTimelineEntry struct {
	ID              string         `json:"id"`
	ClientAccountID string         `json:"client_account_id"`
	EnvironmentID   string         `json:"environment_id"`
	MessageID       string         `json:"message_id"`
	MessageKind     string         `json:"message_kind"`
	MessageRef      string         `json:"message_ref"`
	EventID         string         `json:"event_id"`
	EventType       string         `json:"event_type"`
	Status          string         `json:"status"`
	RecipientDomain string         `json:"recipient_domain"`
	RecipientSHA256 string         `json:"recipient_sha256"`
	SubjectRedacted string         `json:"subject_redacted"`
	RedactedPayload map[string]any `json:"redacted_payload"`
	OccurredAt      string         `json:"occurred_at"`
	CreatedAt       string         `json:"created_at"`
}

type InboundMessageSummary struct {
	ID                string `json:"id"`
	DomainID          string `json:"domain_id"`
	Provider          string `json:"provider"`
	ProviderMessageID string `json:"provider_message_id"`
	SenderAddress     string `json:"sender_address"`
	Subject           string `json:"subject"`
	ParseStatus       string `json:"parse_status"`
	ReceivedAt        string `json:"received_at"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

type InboundObjectReference struct {
	Key         string `json:"key"`
	ContentType string `json:"content_type"`
	SHA256      string `json:"sha256"`
	SizeBytes   int64  `json:"size_bytes"`
}

type PlatformInboundAttachment struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Disposition string `json:"disposition"`
	ContentID   string `json:"content_id"`
	ObjectKey   string `json:"object_key"`
	SHA256      string `json:"sha256"`
	SizeBytes   int64  `json:"size_bytes"`
}

type InboundRecipient struct {
	ID      string `json:"id"`
	Address string `json:"address"`
	Kind    string `json:"kind"`
}

type InboundMessage struct {
	InboundMessageSummary
	Headers     map[string]any              `json:"headers"`
	Recipients  []InboundRecipient          `json:"recipients"`
	Attachments []PlatformInboundAttachment `json:"attachments"`
	HTMLBody    InboundObjectReference      `json:"html_body"`
	TextBody    InboundObjectReference      `json:"text_body"`
	RawMIME     InboundObjectReference      `json:"raw_mime"`
}

type InboundAttachmentDownload struct {
	Attachment PlatformInboundAttachment `json:"attachment"`
	ExpiresAt  string                    `json:"expires_at"`
	URL        string                    `json:"url"`
}

type ReceivingDomain struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	ReceivingEnabled   bool   `json:"receiving_enabled"`
	ReceivingProvider  string `json:"receiving_provider"`
	VerificationStatus string `json:"verification_status"`
	UpdatedAt          string `json:"updated_at"`
}

type UpdateReceivingDomainRequest struct {
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider,omitempty"`
}

type PlatformPreferenceTopic struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Slug              string `json:"slug"`
	DefaultSubscribed bool   `json:"default_subscribed"`
	Status            string `json:"status"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

type CreatePlatformPreferenceTopicRequest struct {
	Name              string `json:"name"`
	Slug              string `json:"slug"`
	DefaultSubscribed bool   `json:"default_subscribed"`
}

type IssuePreferenceTokenRequest struct {
	Email   string `json:"email"`
	Purpose string `json:"purpose"`
}

type PreferenceToken struct {
	Token     string `json:"token"`
	Purpose   string `json:"purpose"`
	ExpiresAt string `json:"expires_at"`
}

type PreferenceTopicChoice struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	Subscribed bool   `json:"subscribed"`
}

type PlatformPreferenceCenter struct {
	Email        string                  `json:"email"`
	ExpiresAt    string                  `json:"expires_at"`
	Topics       []PreferenceTopicChoice `json:"topics"`
	Unsubscribed bool                    `json:"unsubscribed"`
}

type PreferenceSubscriptionUpdate struct {
	TopicID    string `json:"topic_id"`
	Subscribed bool   `json:"subscribed"`
}

type UpdatePreferenceCenterRequest struct {
	Subscriptions []PreferenceSubscriptionUpdate `json:"subscriptions"`
}

type CreatePlatformSuppressionRequest struct {
	Email       string `json:"email,omitempty"`
	EmailSHA256 string `json:"email_sha256,omitempty"`
	Reason      string `json:"reason"`
	Source      string `json:"source,omitempty"`
}

type SuppressionEntry struct {
	ID          string `json:"id"`
	EmailSHA256 string `json:"email_sha256"`
	Reason      string `json:"reason"`
	Source      string `json:"source"`
	Active      bool   `json:"active"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type SMTPCredentialCreateRequest struct {
	Name                  string   `json:"name"`
	Username              string   `json:"username"`
	AllowedAuthMechanisms []string `json:"allowedAuthMechanisms,omitempty"`
	AllowedFromDomains    []string `json:"allowedFromDomains,omitempty"`
	DailyQuota            int      `json:"dailyQuota,omitempty"`
	ExpiresAt             string   `json:"expiresAt,omitempty"`
	MaxMessageBytes       int      `json:"maxMessageBytes,omitempty"`
	MaxRecipients         int      `json:"maxRecipients,omitempty"`
	RateLimitPerMinute    int      `json:"rateLimitPerMinute,omitempty"`
	RequireTLS            bool     `json:"requireTLS,omitempty"`
}

type SMTPCredential struct {
	ID                    string   `json:"id"`
	Name                  string   `json:"name"`
	Username              string   `json:"username"`
	Status                string   `json:"status"`
	AllowedAuthMechanisms []string `json:"allowedAuthMechanisms"`
	AllowedFromDomains    []string `json:"allowedFromDomains"`
	DailyQuota            int      `json:"dailyQuota"`
	RateLimitPerMinute    int      `json:"rateLimitPerMinute"`
	MaxMessageBytes       int      `json:"maxMessageBytes"`
	MaxRecipients         int      `json:"maxRecipients"`
	RequireTLS            bool     `json:"requireTLS"`
	SecretVersion         int      `json:"secretVersion"`
	ExpiresAt             string   `json:"expiresAt"`
	CreatedAt             string   `json:"createdAt"`
	UpdatedAt             string   `json:"updatedAt"`
	LastUsedAt            string   `json:"lastUsedAt"`
	RotatedAt             string   `json:"rotatedAt"`
	RevokedAt             string   `json:"revokedAt"`
}

type CreatedSMTPCredential struct {
	Credential SMTPCredential `json:"credential"`
	Secret     string         `json:"secret"`
}

type RevokedSMTPCredential struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type SMTPSubmission struct {
	ID                     string   `json:"id"`
	CredentialID           string   `json:"credentialId"`
	TransactionalMessageID string   `json:"transactionalMessageId"`
	RequestID              string   `json:"requestId"`
	From                   string   `json:"from"`
	To                     []string `json:"to"`
	CC                     []string `json:"cc"`
	BCCCount               int      `json:"bccCount"`
	MessageBytes           int      `json:"messageBytes"`
	MessageSHA256          string   `json:"messageSha256"`
	Status                 string   `json:"status"`
	FailureCode            string   `json:"failureCode"`
	QueuedAt               string   `json:"queuedAt"`
	FailedAt               string   `json:"failedAt"`
	CreatedAt              string   `json:"createdAt"`
	UpdatedAt              string   `json:"updatedAt"`
}

type PlatformContact struct {
	ID                string            `json:"id"`
	ClientAccountID   string            `json:"client_account_id"`
	EnvironmentID     string            `json:"environment_id"`
	Email             string            `json:"email"`
	Name              string            `json:"name"`
	ExternalReference string            `json:"external_reference"`
	Status            string            `json:"status"`
	Attributes        map[string]string `json:"attributes"`
	CreatedAt         string            `json:"created_at"`
	UpdatedAt         string            `json:"updated_at"`
}

type BulkIngestResult struct {
	Accepted   int `json:"accepted"`
	Suppressed int `json:"suppressed"`
}

type CreatePlatformContactRequest struct {
	Email             string            `json:"email"`
	Name              string            `json:"name,omitempty"`
	ExternalReference string            `json:"external_reference,omitempty"`
	Attributes        map[string]string `json:"attributes,omitempty"`
}

type BroadcastSnapshotRecipient struct {
	ID        string            `json:"id"`
	ContactID string            `json:"contact_id"`
	Address   string            `json:"address"`
	Data      map[string]string `json:"data"`
}

type BroadcastSnapshot struct {
	ID               string                       `json:"id"`
	BroadcastID      string                       `json:"broadcast_id"`
	SegmentID        string                       `json:"segment_id"`
	SegmentRevision  int                          `json:"segment_revision"`
	DefinitionDigest string                       `json:"definition_digest"`
	RecipientCount   int                          `json:"recipient_count"`
	Recipients       []BroadcastSnapshotRecipient `json:"recipients"`
	CreatedAt        string                       `json:"created_at"`
}
type UpdatePlatformContactRequest = CreatePlatformContactRequest
type PlatformContactTransitionRequest struct {
	Event string `json:"event"`
}

type SegmentCondition struct {
	Field    string   `json:"field"`
	Operator string   `json:"operator"`
	Value    string   `json:"value,omitempty"`
	Values   []string `json:"values,omitempty"`
}

type PlatformSegment struct {
	ID              string             `json:"id"`
	ClientAccountID string             `json:"client_account_id"`
	EnvironmentID   string             `json:"environment_id"`
	Name            string             `json:"name"`
	Description     string             `json:"description"`
	Match           string             `json:"match"`
	Conditions      []SegmentCondition `json:"conditions"`
	Revision        int                `json:"revision"`
	Status          string             `json:"status"`
	CreatedAt       string             `json:"created_at"`
	UpdatedAt       string             `json:"updated_at"`
}
type CreatePlatformSegmentRequest struct {
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Match       string             `json:"match"`
	Conditions  []SegmentCondition `json:"conditions"`
}
type SegmentPreview struct {
	SegmentID       string            `json:"segment_id"`
	SegmentRevision int               `json:"segment_revision"`
	EvaluatedCount  int               `json:"evaluated_count"`
	MatchedCount    int               `json:"matched_count"`
	Contacts        []PlatformContact `json:"contacts"`
}

type PlatformAutomation struct {
	ID              string `json:"id"`
	ClientAccountID string `json:"client_account_id"`
	EnvironmentID   string `json:"environment_id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Status          string `json:"status"`
	Version         int    `json:"version"`
	Trigger         any    `json:"trigger"`
	Steps           any    `json:"steps"`
	ActivatedAt     string `json:"activated_at"`
	ArchivedAt      string `json:"archived_at"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}
type CreatePlatformAutomationRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Trigger     any    `json:"trigger"`
	Steps       any    `json:"steps"`
}
type PlatformAutomationTransitionRequest struct {
	Event string `json:"event"`
}
type StartPlatformAutomationRunRequest struct {
	ContactID      string `json:"contact_id"`
	TriggerEventID string `json:"trigger_event_id,omitempty"`
}
type PlatformAutomationRun struct {
	ID             string `json:"id"`
	AutomationID   string `json:"automation_id"`
	ContactID      string `json:"contact_id"`
	TriggerEventID string `json:"trigger_event_id"`
	Status         string `json:"status"`
	StepIndex      int    `json:"step_index"`
	NextAt         string `json:"next_at"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
	IdempotencyKey string `json:"idempotency_key"`
}

type PlatformRoutingPolicyRequest struct {
	AllowedRegions     []string `json:"allowedRegions,omitempty"`
	PreferredRegion    string   `json:"preferredRegion,omitempty"`
	ProviderAllowlist  []string `json:"providerAllowlist,omitempty"`
	RequireDedicatedIP bool     `json:"requireDedicatedIP,omitempty"`
	RequiredResidency  string   `json:"requiredResidency,omitempty"`
}
type PlatformRoutingPolicy struct {
	PlatformRoutingPolicyRequest
	Version   int    `json:"version"`
	UpdatedAt string `json:"updatedAt"`
}
type RoutingIPRequest struct {
	ID            string `json:"id,omitempty"`
	Address       string `json:"address,omitempty"`
	AssignmentRef string `json:"assignmentRef,omitempty"`
	DailySent     int    `json:"dailySent,omitempty"`
	DailySentDate string `json:"dailySentDate,omitempty"`
	Health        string `json:"health,omitempty"`
	HoldReason    string `json:"holdReason,omitempty"`
	HoldUntil     string `json:"holdUntil,omitempty"`
	Reputation    int    `json:"reputation,omitempty"`
	State         string `json:"state,omitempty"`
}
type RoutingIP = RoutingIPRequest
type RoutingWarmupRequest struct {
	DailyIncrement       int    `json:"dailyIncrement,omitempty"`
	InitialDailyCapacity int    `json:"initialDailyCapacity,omitempty"`
	MaxDailyCapacity     int    `json:"maxDailyCapacity,omitempty"`
	StartAt              string `json:"startAt,omitempty"`
}
type RoutingPoolRequest struct {
	Provider          string                `json:"provider"`
	ProviderAccountID string                `json:"providerAccountId,omitempty"`
	Region            string                `json:"region"`
	Residency         string                `json:"residency,omitempty"`
	Health            string                `json:"health,omitempty"`
	State             string                `json:"state,omitempty"`
	HoldReason        string                `json:"holdReason,omitempty"`
	HoldUntil         string                `json:"holdUntil,omitempty"`
	TrafficWeight     int                   `json:"trafficWeight,omitempty"`
	IPs               []RoutingIPRequest    `json:"ips,omitempty"`
	Warmup            *RoutingWarmupRequest `json:"warmup,omitempty"`
}
type RoutingPool struct {
	RoutingPoolRequest
	ID        string `json:"id"`
	Version   int    `json:"version"`
	UpdatedAt string `json:"updatedAt"`
}
type RoutingProviderRequest struct {
	Provider           string `json:"provider"`
	ProviderAccountRef string `json:"providerAccountRef,omitempty"`
	State              string `json:"state,omitempty"`
	ObservedAt         string `json:"observedAt,omitempty"`
}
type RoutingProvider struct {
	RoutingProviderRequest
	ID     string         `json:"id"`
	Routes []RoutingRoute `json:"routes"`
}
type RoutingRouteRequest struct {
	ID                 string   `json:"id,omitempty"`
	Region             string   `json:"region"`
	Residency          string   `json:"residency,omitempty"`
	Available          bool     `json:"available"`
	Healthy            bool     `json:"healthy"`
	DedicatedIPSupport bool     `json:"dedicatedIPSupport"`
	TrafficClasses     []string `json:"trafficClasses,omitempty"`
	Weight             int      `json:"weight,omitempty"`
	ObservedAt         string   `json:"observedAt,omitempty"`
}
type RoutingRoutesRequest struct {
	Provider           string                `json:"provider"`
	ProviderAccountRef string                `json:"providerAccountRef,omitempty"`
	Routes             []RoutingRouteRequest `json:"routes"`
}
type RoutingRoute struct {
	RoutingRouteRequest
	Provider             string `json:"provider"`
	ProviderAccountID    string `json:"providerAccountId"`
	ProviderAccountRef   string `json:"providerAccountRef"`
	ProviderAccountState string `json:"providerAccountState"`
}
type ResolveRoutingRequest struct {
	RoutingKey   string `json:"routingKey"`
	TrafficClass string `json:"trafficClass,omitempty"`
	Now          string `json:"now,omitempty"`
}
type RoutingResolution struct {
	Provider          string `json:"provider"`
	ProviderAccountID string `json:"providerAccountId"`
	Region            string `json:"region"`
	Residency         string `json:"residency"`
	PoolID            string `json:"poolId"`
	IPID              string `json:"ipId"`
}

type DomainHealthCheck struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Purpose        string   `json:"purpose"`
	Type           string   `json:"type"`
	ExpectationID  string   `json:"expectationId"`
	ExpectedValues []string `json:"expectedValues"`
	ObservedValues []string `json:"observedValues"`
	Required       bool     `json:"required"`
	Severity       string   `json:"severity"`
	Status         string   `json:"status"`
	ErrorCode      string   `json:"errorCode"`
	CheckedAt      string   `json:"checkedAt"`
}
type DomainHealthFinding struct {
	CheckID     string `json:"checkId"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Title       string `json:"title"`
}
type PlatformDomainHealthReport struct {
	ID            string                `json:"id"`
	DomainID      string                `json:"domainId"`
	DomainName    string                `json:"domainName"`
	ObservationID string                `json:"observationId"`
	Status        string                `json:"status"`
	Score         int                   `json:"score"`
	CheckedAt     string                `json:"checkedAt"`
	NextCheckAt   string                `json:"nextCheckAt"`
	Checks        []DomainHealthCheck   `json:"checks"`
	Findings      []DomainHealthFinding `json:"findings"`
}

type PlatformDomain struct {
	ID                string                       `json:"id"`
	Name              string                       `json:"name"`
	ProviderBindingID string                       `json:"providerBindingId"`
	VerificationState string                       `json:"verificationState"`
	RequiredRecords   PlatformDomainRecordSnapshot `json:"requiredRecords"`
	CreatedAt         string                       `json:"createdAt"`
	LastVerifiedAt    string                       `json:"lastVerifiedAt"`
	UpdatedAt         string                       `json:"updatedAt"`
}
type PlatformDomainRecord struct {
	Name     string `json:"name"`
	Priority int    `json:"priority"`
	Purpose  string `json:"purpose"`
	Type     string `json:"type"`
	Value    string `json:"value"`
}
type PlatformDomainRecordSnapshot struct {
	Domain    string                 `json:"domain"`
	Records   []PlatformDomainRecord `json:"records"`
	CreatedAt string                 `json:"createdAt"`
}
type StartDomainOnboardingRequest struct {
	Domain            string `json:"domain"`
	ProviderBindingID string `json:"providerBindingId,omitempty"`
}
type DomainOnboarding struct {
	Domain                PlatformDomain        `json:"domain"`
	AvailableSources      []DomainConnectSource `json:"availableSources"`
	DomainConnectTemplate DomainConnectTemplate `json:"domainConnectTemplate"`
}
type DomainConnectSource struct {
	Kind     string `json:"kind"`
	Provider string `json:"provider"`
}
type DomainConnectTemplate struct {
	TemplateID string `json:"templateId"`
	Version    string `json:"version"`
}
type BeginDomainConnectorRequest struct {
	Provider   string `json:"provider"`
	SourceKind string `json:"sourceKind"`
}
type CompleteDomainConnectorRequest struct {
	Binding ConnectorBinding `json:"binding"`
}
type ConnectorBinding struct {
	AttemptID         string                `json:"attemptId"`
	Domain            string                `json:"domain"`
	DomainID          string                `json:"domainId"`
	ExpiresAt         string                `json:"expiresAt"`
	ProviderBindingID string                `json:"providerBindingId"`
	Signature         string                `json:"signature"`
	Template          DomainConnectTemplate `json:"template"`
}
type DomainConnector struct {
	AuthorizationURL string           `json:"authorizationUrl"`
	Binding          ConnectorBinding `json:"binding"`
}

// CursorPage supports both snake_case cursor payloads used by events/inbound
// routes and camelCase cursors used by domain-health and usage-contract
// routes. The wire variation is an existing backend compatibility detail; the
// SDK keeps one stable field for callers.
func (p *CursorPage[T]) UnmarshalJSON(data []byte) error {
	var payload struct {
		Items           []T    `json:"items"`
		NextCursor      string `json:"next_cursor"`
		NextCursorCamel string `json:"nextCursor"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	p.Items = payload.Items
	p.NextCursor = payload.NextCursor
	if p.NextCursor == "" {
		p.NextCursor = payload.NextCursorCamel
	}
	return nil
}

type UsageFact struct {
	ID                     string            `json:"id"`
	FactType               string            `json:"factType"`
	Family                 string            `json:"family"`
	Kind                   string            `json:"kind"`
	Unit                   string            `json:"unit"`
	QuantityDelta          int               `json:"quantityDelta"`
	ClientAccountID        string            `json:"clientAccountId"`
	EnvironmentID          string            `json:"environmentId"`
	PlatformOrganizationID string            `json:"platformOrganizationId"`
	Dimensions             map[string]string `json:"dimensions"`
	Metadata               map[string]any    `json:"metadata"`
	SourceKind             string            `json:"sourceKind"`
	SourceRef              string            `json:"sourceRef"`
	IdempotencyKey         string            `json:"idempotencyKey"`
	CorrelationID          string            `json:"correlationId"`
	CausationID            string            `json:"causationId"`
	CorrectionOfID         string            `json:"correctionOfId"`
	OccurredAt             string            `json:"occurredAt"`
	RecordedAt             string            `json:"recordedAt"`
	CreatedAt              string            `json:"createdAt"`
	RetentionClass         string            `json:"retentionClass"`
	RetentionUntil         string            `json:"retentionUntil"`
	SchemaVersion          int               `json:"schemaVersion"`
	LegalHold              bool              `json:"legalHold"`
	LinkageKey             string            `json:"linkageKey"`
}
type UsageCorrectionRequest struct {
	QuantityDelta  int            `json:"quantityDelta"`
	FactType       string         `json:"factType,omitempty"`
	SourceRef      string         `json:"sourceRef,omitempty"`
	IdempotencyKey string         `json:"idempotencyKey,omitempty"`
	CorrelationID  string         `json:"correlationId,omitempty"`
	CausationID    string         `json:"causationId,omitempty"`
	OccurredAt     string         `json:"occurredAt,omitempty"`
	RecordedAt     string         `json:"recordedAt,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	RetentionUntil string         `json:"retentionUntil,omitempty"`
	LegalHold      bool           `json:"legalHold,omitempty"`
}
type UsageVocabularyEntry struct {
	FactType           string   `json:"factType"`
	Family             string   `json:"family"`
	Description        string   `json:"description"`
	Unit               string   `json:"unit"`
	CorrectionPolicy   string   `json:"correctionPolicy"`
	RetentionClass     string   `json:"retentionClass"`
	OptionalDimensions []string `json:"optionalDimensions"`
	RequiredDimensions []string `json:"requiredDimensions"`
	RetentionDays      int      `json:"retentionDays"`
	SchemaVersion      int      `json:"schemaVersion"`
}
type UsageReconciliation struct {
	ID                     string         `json:"id"`
	ClientAccountID        string         `json:"clientAccountId"`
	EnvironmentID          string         `json:"environmentId"`
	PlatformOrganizationID string         `json:"platformOrganizationId"`
	SourceSystem           string         `json:"sourceSystem"`
	SourceSnapshotKey      string         `json:"sourceSnapshotKey"`
	WindowFrom             string         `json:"windowFrom"`
	WindowTo               string         `json:"windowTo"`
	Status                 string         `json:"status"`
	StartedAt              string         `json:"startedAt"`
	CompletedAt            string         `json:"completedAt"`
	ExpectedCount          int            `json:"expectedCount"`
	ObservedCount          int            `json:"observedCount"`
	MatchedCount           int            `json:"matchedCount"`
	MissingCount           int            `json:"missingCount"`
	UnexpectedCount        int            `json:"unexpectedCount"`
	CorrectionCount        int            `json:"correctionCount"`
	Notes                  map[string]any `json:"notes"`
}
type StartUsageReconciliationRequest struct {
	SourceSnapshotKey string `json:"sourceSnapshotKey"`
	SourceSystem      string `json:"sourceSystem"`
	WindowFrom        string `json:"windowFrom"`
	WindowTo          string `json:"windowTo"`
}
type FinishUsageReconciliationRequest struct {
	CompletedAt     string         `json:"completedAt,omitempty"`
	CorrectionCount int            `json:"correctionCount,omitempty"`
	ExpectedCount   int            `json:"expectedCount,omitempty"`
	MatchedCount    int            `json:"matchedCount,omitempty"`
	MissingCount    int            `json:"missingCount,omitempty"`
	ObservedCount   int            `json:"observedCount,omitempty"`
	UnexpectedCount int            `json:"unexpectedCount,omitempty"`
	Notes           map[string]any `json:"notes,omitempty"`
	Status          string         `json:"status,omitempty"`
}
type UsageReconciliationItem struct {
	ID               int               `json:"id"`
	CanonicalKey     string            `json:"canonicalKey"`
	ExternalRef      string            `json:"externalRef"`
	FactID           string            `json:"factId"`
	ExpectedQuantity int               `json:"expectedQuantity"`
	ObservedQuantity int               `json:"observedQuantity"`
	Status           string            `json:"status"`
	Dimensions       map[string]string `json:"dimensions"`
	Details          map[string]any    `json:"details"`
	CreatedAt        string            `json:"createdAt"`
}
type UsageReconciliationItemRequest struct {
	CanonicalKey     string            `json:"canonicalKey"`
	ExternalRef      string            `json:"externalRef,omitempty"`
	FactID           string            `json:"factId,omitempty"`
	Status           string            `json:"status,omitempty"`
	ExpectedQuantity int               `json:"expectedQuantity,omitempty"`
	ObservedQuantity int               `json:"observedQuantity,omitempty"`
	Dimensions       map[string]string `json:"dimensions,omitempty"`
	Details          map[string]any    `json:"details,omitempty"`
}

// Account-level compatibility aliases make the model names discoverable
// without exposing implementation-package prefixes.
type Account = ClientAccount
type Environment = ClientEnvironment
type Message = MessageDetail
type WebhookEndpoint = ScopedWebhook
