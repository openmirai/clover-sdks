package clover

import "time"

// String returns a pointer suitable for optional request fields.
func String(value string) *string { return &value }

// Bool returns a pointer suitable for optional request fields.
func Bool(value bool) *bool { return &value }

// Page is the common page payload returned by Clover list endpoints. Some
// native full-product responses also include next_cursor; it is retained here
// so clients can move between page- and cursor-based endpoints without losing
// forward-compatible fields.
type Page[T any] struct {
	Items      []T        `json:"items"`
	Pagination Pagination `json:"pagination"`
	NextCursor *string    `json:"next_cursor,omitempty"`
}

type Pagination struct {
	Page        int  `json:"page"`
	Limit       int  `json:"limit"`
	Total       int  `json:"total"`
	HasNext     bool `json:"hasNext"`
	HasPrevious bool `json:"hasPrevious"`
}

// Scope identifies the tenant boundary required by the native lifecycle
// endpoints. IDs intentionally remain strings so this dependency-free module
// can be used with UUIDs, ULIDs, or test fixtures alike.
type Scope struct {
	OrganizationID string `json:"organization_id,omitempty"`
	ProjectID      string `json:"project_id,omitempty"`
	Environment    string `json:"environment,omitempty"`
	TenantID       string `json:"tenant_id,omitempty"`
}

type Domain struct {
	ID               string      `json:"id"`
	Name             string      `json:"name"`
	Status           string      `json:"status"`
	Provider         string      `json:"provider"`
	Region           string      `json:"region"`
	SendingEnabled   bool        `json:"sendingEnabled"`
	ReceivingEnabled bool        `json:"receivingEnabled"`
	TrackingDomain   *string     `json:"trackingDomain"`
	TLSPolicy        string      `json:"tlsPolicy"`
	DNSRecords       []DNSRecord `json:"dnsRecords,omitempty"`
	LastVerifiedAt   *time.Time  `json:"lastVerifiedAt"`
	NextCheckAt      *time.Time  `json:"nextCheckAt"`
	CreatedAt        *time.Time  `json:"createdAt"`
	UpdatedAt        *time.Time  `json:"updatedAt"`
}

type DNSRecord struct {
	ID             string     `json:"id"`
	Purpose        string     `json:"purpose"`
	Type           string     `json:"type"`
	Name           string     `json:"name"`
	ExpectedValues []string   `json:"expectedValues"`
	Priority       *int       `json:"priority,omitempty"`
	Required       bool       `json:"required"`
	Status         string     `json:"status"`
	ObservedValues []string   `json:"observedValues,omitempty"`
	LastCheckedAt  *time.Time `json:"lastCheckedAt,omitempty"`
	LastErrorCode  *string    `json:"lastErrorCode,omitempty"`
}

// ProvisionDNSRequest asks the provider adapter to provision the domain's
// authoritative DNS records before verification is queued.
type ProvisionDNSRequest struct {
	Force *bool `json:"force,omitempty"`
}

// DNSProvisionAccepted is returned by the asynchronous DNS provisioning
// endpoint (HTTP 202).
type DNSProvisionAccepted struct {
	DomainID       string      `json:"domainId"`
	IdentityID     string      `json:"identityId"`
	Provider       string      `json:"provider"`
	Status         string      `json:"status"`
	VerificationID string      `json:"verificationId"`
	NextCheckAt    *time.Time  `json:"nextCheckAt"`
	DNSRecords     []DNSRecord `json:"dnsRecords"`
}

type DomainVerification struct {
	DomainID       string     `json:"domainId"`
	Status         string     `json:"status"`
	VerificationID string     `json:"verificationId"`
	NextCheckAt    *time.Time `json:"nextCheckAt"`
}

type CreateDomainRequest struct {
	Name             string  `json:"name"`
	Provider         string  `json:"provider"`
	Region           string  `json:"region"`
	ReceivingEnabled bool    `json:"receivingEnabled,omitempty"`
	TrackingDomain   *string `json:"trackingDomain,omitempty"`
	TLSPolicy        string  `json:"tlsPolicy,omitempty"`
}

type UpdateDomainRequest struct {
	Provider         *string `json:"provider,omitempty"`
	Region           *string `json:"region,omitempty"`
	SendingEnabled   *bool   `json:"sendingEnabled,omitempty"`
	ReceivingEnabled *bool   `json:"receivingEnabled,omitempty"`
	TrackingDomain   *string `json:"trackingDomain,omitempty"`
	TLSPolicy        *string `json:"tlsPolicy,omitempty"`
}

type VerifyDomainRequest struct {
	Force *bool `json:"force,omitempty"`
}

type APIKey struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Prefix      string     `json:"prefix"`
	PublicKeyID string     `json:"public_key_id"`
	Permission  string     `json:"permission"`
	DomainID    *string    `json:"domain_id"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	CreatedAt   *time.Time `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	RevokedAt   *time.Time `json:"revoked_at"`
}

type CreateAPIKeyRequest struct {
	Name       string  `json:"name"`
	Permission string  `json:"permission,omitempty"`
	DomainID   *string `json:"domain_id,omitempty"`
}

type UpdateAPIKeyRequest struct {
	Name       *string `json:"name,omitempty"`
	Permission *string `json:"permission,omitempty"`
	DomainID   *string `json:"domain_id,omitempty"`
}

type CreateAPIKeyResponse struct {
	Key   APIKey `json:"key"`
	Token string `json:"token"`
}

type EmailAddress struct {
	Address string  `json:"address"`
	Name    *string `json:"name,omitempty"`
}

type SendEmailRequest struct {
	From        EmailAddress           `json:"from"`
	To          []EmailAddress         `json:"to"`
	CC          []EmailAddress         `json:"cc,omitempty"`
	BCC         []EmailAddress         `json:"bcc,omitempty"`
	ReplyTo     []EmailAddress         `json:"reply_to,omitempty"`
	Subject     string                 `json:"subject"`
	HTML        *string                `json:"html,omitempty"`
	Text        *string                `json:"text,omitempty"`
	Attachments []EmailAttachmentInput `json:"attachments,omitempty"`
	Headers     map[string]string      `json:"headers,omitempty"`
	Tags        map[string]string      `json:"tags,omitempty"`
	ScheduledAt *time.Time             `json:"scheduled_at,omitempty"`
	Priority    string                 `json:"priority,omitempty"`
	DeliverBy   *string                `json:"deliver_by,omitempty"`
}

type EmailAttachmentInput struct {
	Filename    string  `json:"filename"`
	ContentType string  `json:"content_type"`
	Disposition string  `json:"disposition"`
	ContentID   *string `json:"content_id,omitempty"`
	Content     *string `json:"content,omitempty"`
	UploadToken *string `json:"upload_token,omitempty"`
}

type ScheduleEmailRequest struct {
	ScheduledAt time.Time `json:"scheduled_at"`
}

type EmailAccepted struct {
	ID          string     `json:"id"`
	Status      string     `json:"status"`
	ScheduledAt *time.Time `json:"scheduled_at"`
	RequestID   string     `json:"request_id"`
}

type EmailBatchAcceptedItem struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type EmailBatchAccepted struct {
	Data      []EmailBatchAcceptedItem `json:"data"`
	RequestID string                   `json:"request_id"`
}

type EmailSummary struct {
	ID                string       `json:"id"`
	From              EmailAddress `json:"from"`
	ToCount           int          `json:"to_count"`
	Subject           string       `json:"subject,omitempty"`
	Status            string       `json:"status"`
	Priority          string       `json:"priority"`
	DeliverBy         *string      `json:"deliver_by,omitempty"`
	ScheduledAt       *time.Time   `json:"scheduled_at"`
	ProviderMessageID *string      `json:"provider_message_id"`
	CreatedAt         *time.Time   `json:"created_at"`
	UpdatedAt         *time.Time   `json:"updated_at"`
}

type EmailRecipient struct {
	Address string  `json:"address"`
	Name    *string `json:"name,omitempty"`
	Kind    string  `json:"kind"`
}

type EmailAttachment struct {
	ID          string  `json:"id"`
	Filename    string  `json:"filename"`
	ContentType string  `json:"content_type"`
	Disposition string  `json:"disposition"`
	ContentID   *string `json:"content_id,omitempty"`
	SizeBytes   int64   `json:"size_bytes"`
	SHA256      string  `json:"sha256"`
	UploadToken *string `json:"upload_token,omitempty"`
}

type EmailEventSummary struct {
	ID              string     `json:"id"`
	Type            string     `json:"type"`
	OccurredAt      *time.Time `json:"occurred_at"`
	ProviderEventID *string    `json:"provider_event_id,omitempty"`
	ErrorCode       *string    `json:"error_code,omitempty"`
}

type EmailTraceStep struct {
	EventID    string     `json:"event_id"`
	Type       string     `json:"type"`
	OccurredAt *time.Time `json:"occurred_at"`
	ObservedAt *time.Time `json:"observed_at"`
	Status     string     `json:"status"`
	Payload    any        `json:"payload,omitempty"`
	EvidenceID string     `json:"evidence_id"`
}

type EmailTrace struct {
	EmailID     string           `json:"email_id"`
	Scope       Scope            `json:"scope"`
	Steps       []EmailTraceStep `json:"steps"`
	StartedAt   *time.Time       `json:"started_at"`
	FinishedAt  *time.Time       `json:"finished_at,omitempty"`
	EvidenceIDs []string         `json:"evidence_ids"`
	Complete    bool             `json:"complete"`
}

type ReplayEmailRequest struct {
	Reason      string   `json:"reason"`
	EvidenceIDs []string `json:"evidence_ids"`
}

type ReplayPlan struct {
	ID             string     `json:"id"`
	Scope          Scope      `json:"scope"`
	EmailID        string     `json:"email_id"`
	Reason         string     `json:"reason"`
	EvidenceIDs    []string   `json:"evidence_ids"`
	Status         string     `json:"status"`
	CreatedAt      *time.Time `json:"created_at"`
	IdempotencyKey string     `json:"idempotency_key,omitempty"`
}

type EmailDetail struct {
	EmailSummary
	RequestID   string              `json:"request_id,omitempty"`
	Recipients  []EmailRecipient    `json:"recipients"`
	ReplyTo     []EmailAddress      `json:"reply_to,omitempty"`
	HTML        *string             `json:"html,omitempty"`
	Text        *string             `json:"text,omitempty"`
	Attachments []EmailAttachment   `json:"attachments"`
	Tags        map[string]string   `json:"tags,omitempty"`
	Events      []EmailEventSummary `json:"events"`
}

type AttachmentUploadRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	SHA256      string `json:"sha256"`
}

type AttachmentUpload struct {
	Token           string            `json:"token"`
	UploadURL       string            `json:"upload_url"`
	ExpiresAt       *time.Time        `json:"expires_at"`
	MaxBytes        int64             `json:"max_bytes"`
	ContentType     string            `json:"content_type"`
	RequiredHeaders map[string]string `json:"required_headers,omitempty"`
}

type Webhook struct {
	ID            string     `json:"id"`
	URL           string     `json:"url"`
	Description   *string    `json:"description"`
	Subscriptions []string   `json:"subscriptions"`
	Enabled       bool       `json:"enabled"`
	FailureCount  int        `json:"failure_count"`
	LastSuccessAt *time.Time `json:"last_success_at"`
	LastFailureAt *time.Time `json:"last_failure_at"`
	DisabledAt    *time.Time `json:"disabled_at"`
	CreatedAt     *time.Time `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at"`
}

type WebhookCreated struct {
	Webhook
	Secret string `json:"secret"`
}

type WebhookSecretRotation struct {
	WebhookID    string    `json:"webhook_id"`
	Secret       string    `json:"secret"`
	OverlapUntil time.Time `json:"overlap_until"`
}

type CreateWebhookRequest struct {
	URL           string   `json:"url"`
	Description   *string  `json:"description,omitempty"`
	Subscriptions []string `json:"subscriptions,omitempty"`
	Enabled       *bool    `json:"enabled,omitempty"`
}

type UpdateWebhookRequest struct {
	URL           *string   `json:"url,omitempty"`
	Description   *string   `json:"description,omitempty"`
	Subscriptions *[]string `json:"subscriptions,omitempty"`
	Enabled       *bool     `json:"enabled,omitempty"`
}

type WebhookDelivery struct {
	ID             string     `json:"id"`
	EndpointID     string     `json:"endpoint_id"`
	EventID        string     `json:"event_id"`
	EventType      string     `json:"event_type"`
	EmailID        *string    `json:"email_id"`
	Status         string     `json:"status"`
	AttemptCount   int        `json:"attempt_count"`
	LastHTTPStatus *int       `json:"last_http_status"`
	LastErrorCode  *string    `json:"last_error_code"`
	NextAttemptAt  *time.Time `json:"next_attempt_at"`
	DeliveredAt    *time.Time `json:"delivered_at"`
	CreatedAt      *time.Time `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
	Event          JSON       `json:"event,omitempty"`
}

type InboundAttachment struct {
	ID                   string  `json:"id"`
	Filename             string  `json:"filename"`
	ContentType          string  `json:"content_type"`
	SizeBytes            int64   `json:"size_bytes"`
	SHA256               string  `json:"sha256"`
	ProviderAttachmentID *string `json:"provider_attachment_id,omitempty"`
}

type InboundEmail struct {
	ID                string              `json:"id"`
	ProviderMessageID string              `json:"provider_message_id"`
	ParseStatus       string              `json:"parse_status"`
	From              *string             `json:"from"`
	Subject           *string             `json:"subject"`
	DeliveredTo       []string            `json:"delivered_to"`
	ReceivedAt        *time.Time          `json:"received_at"`
	CreatedAt         *time.Time          `json:"created_at"`
	To                []string            `json:"to,omitempty"`
	CC                []string            `json:"cc,omitempty"`
	ReplyTo           []string            `json:"reply_to,omitempty"`
	Text              *string             `json:"text,omitempty"`
	HTML              *string             `json:"html,omitempty"`
	Headers           map[string]string   `json:"headers,omitempty"`
	Attachments       []InboundAttachment `json:"attachments"`
}

type InboundAttachmentURL struct {
	Attachment  InboundAttachment `json:"attachment"`
	DownloadURL string            `json:"download_url"`
	ExpiresAt   *time.Time        `json:"expires_at"`
}

type Suppression struct {
	ID             string     `json:"id"`
	AddressSHA256  string     `json:"address_sha256"`
	DisplayAddress *string    `json:"display_address"`
	Reason         string     `json:"reason"`
	Source         string     `json:"source"`
	Active         bool       `json:"active"`
	CreatedAt      *time.Time `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
	RemovedAt      *time.Time `json:"removed_at"`
}

type CreateSuppressionRequest struct {
	Address string `json:"address"`
	Reason  string `json:"reason"`
}

type EmailMetricBucket struct {
	Start              *time.Time `json:"start"`
	End                *time.Time `json:"end"`
	Accepted           int64      `json:"accepted"`
	Sent               int64      `json:"sent"`
	Delivered          int64      `json:"delivered"`
	Delayed            int64      `json:"delayed"`
	Bounced            int64      `json:"bounced"`
	BounceTransient    int64      `json:"bounce_transient,omitempty"`
	BouncePermanent    int64      `json:"bounce_permanent,omitempty"`
	BounceUndetermined int64      `json:"bounce_undetermined,omitempty"`
	Complained         int64      `json:"complained"`
	Failed             int64      `json:"failed"`
	Opened             int64      `json:"opened"`
	Clicked            int64      `json:"clicked"`
}

type EmailMetrics struct {
	Interval string              `json:"interval"`
	Start    *time.Time          `json:"start"`
	End      *time.Time          `json:"end"`
	Data     []EmailMetricBucket `json:"data"`
}

type RequestLog struct {
	ID             string     `json:"id"`
	RequestID      string     `json:"request_id"`
	Operation      string     `json:"operation"`
	Method         string     `json:"method"`
	PathTemplate   string     `json:"path_template"`
	Status         int        `json:"status"`
	Source         string     `json:"source"`
	APIKeyID       *string    `json:"api_key_id"`
	UserAgentClass *string    `json:"user_agent_class"`
	LatencyMS      int64      `json:"latency_ms"`
	StartedAt      *time.Time `json:"started_at"`
}

type PreferenceTopic struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	CreatedAt *time.Time `json:"created_at"`
}

type CreatePreferenceTopicRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type TopicChoice struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	Subscribed bool   `json:"subscribed"`
}

type PreferenceCenter struct {
	Email        string        `json:"email"`
	Topics       []TopicChoice `json:"topics"`
	Unsubscribed bool          `json:"unsubscribed"`
}

type SubscriptionUpdate struct {
	TopicID    string `json:"topic_id"`
	Subscribed bool   `json:"subscribed"`
}

type UpdatePreferenceRequest struct {
	Subscriptions []SubscriptionUpdate `json:"subscriptions"`
}

type Template struct {
	Scope          Scope      `json:"scope"`
	ID             string     `json:"id"`
	Key            string     `json:"key"`
	Name           string     `json:"name"`
	Description    string     `json:"description,omitempty"`
	Status         string     `json:"status"`
	CurrentVersion int        `json:"current_version"`
	CreatedAt      *time.Time `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
	ArchivedAt     *time.Time `json:"archived_at,omitempty"`
}

type TemplateVersion struct {
	Scope              Scope      `json:"scope"`
	ID                 string     `json:"id"`
	TemplateID         string     `json:"template_id"`
	Number             int        `json:"number"`
	Status             string     `json:"status"`
	Subject            string     `json:"subject"`
	HTML               *string    `json:"html,omitempty"`
	Text               *string    `json:"text,omitempty"`
	Variables          []string   `json:"variables,omitempty"`
	CreatedBy          string     `json:"created_by"`
	CreatedAt          *time.Time `json:"created_at"`
	PublishedAt        *time.Time `json:"published_at,omitempty"`
	ArchivedAt         *time.Time `json:"archived_at,omitempty"`
	Compiler           string     `json:"compiler,omitempty"`
	CompilerVersion    string     `json:"compiler_version,omitempty"`
	ComponentMetadata  JSON       `json:"component_metadata,omitempty"`
	ComponentRefs      []string   `json:"component_refs,omitempty"`
	DerivedFromVersion int        `json:"derived_from_version,omitempty"`
	PreviewData        JSON       `json:"preview_data,omitempty"`
	SourceDigest       string     `json:"source_digest,omitempty"`
	SourceFormat       string     `json:"source_format,omitempty"`
}

// TemplateVersionComparison is the additive diff returned by the compare
// endpoint. Version references are strings on the wire (number, UUID, or
// latest), while the resolved response values are numeric version numbers.
type TemplateVersionComparison struct {
	TemplateID               string   `json:"template_id"`
	From                     int      `json:"from"`
	To                       int      `json:"to"`
	AddedVariables           []string `json:"added_variables,omitempty"`
	RemovedVariables         []string `json:"removed_variables,omitempty"`
	ChangedFields            []string `json:"changed_fields,omitempty"`
	ComponentMetadataChanged bool     `json:"component_metadata_changed"`
	PreviewDataChanged       bool     `json:"preview_data_changed"`
	RenderedHTMLChanged      bool     `json:"rendered_html_changed"`
	RenderedTextChanged      bool     `json:"rendered_text_changed"`
	SourceDigestChanged      bool     `json:"source_digest_changed"`
}

type GetDomainHealthRequest struct {
	DomainID string `json:"domain_id"`
}

type VerifyDomainHealthRequest = GetDomainHealthRequest

// DomainHealthReport is intentionally open-ended because the backend's
// report schema is additive and currently documented as fullproduct.ListData.
// Known report fields are promoted for common consumers while Extra retains
// future fields when callers decode a raw JSON value themselves.
type DomainHealthReport struct {
	ID            string         `json:"id"`
	DomainID      string         `json:"domain_id"`
	DomainName    string         `json:"domain_name"`
	Status        string         `json:"status"`
	Score         int            `json:"score"`
	Checks        []JSON         `json:"checks,omitempty"`
	Findings      []JSON         `json:"findings,omitempty"`
	CheckedAt     *time.Time     `json:"checked_at,omitempty"`
	NextCheckAt   *time.Time     `json:"next_check_at,omitempty"`
	ObservationID string         `json:"observation_id,omitempty"`
	Extra         map[string]any `json:"-"`
}

type DomainHealthPage struct {
	Data       []DomainHealthReport `json:"data"`
	NextCursor *string              `json:"next_cursor,omitempty"`
}

type CreateTemplateRequest struct {
	Scope       Scope  `json:"scope"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type TemplateTransitionRequest struct {
	Event string `json:"event"`
}

type CreateTemplateVersionRequest struct {
	Scope             Scope    `json:"scope"`
	Subject           string   `json:"subject"`
	HTML              string   `json:"html,omitempty"`
	Text              string   `json:"text,omitempty"`
	SourceFormat      string   `json:"source_format,omitempty"`
	SourceDigest      string   `json:"source_digest,omitempty"`
	Compiler          string   `json:"compiler,omitempty"`
	CompilerVersion   string   `json:"compiler_version,omitempty"`
	PreviewData       JSON     `json:"preview_data,omitempty"`
	ComponentMetadata JSON     `json:"component_metadata,omitempty"`
	ComponentRefs     []string `json:"component_refs,omitempty"`
}

type Contact struct {
	Scope          Scope             `json:"scope"`
	ID             string            `json:"id"`
	ExternalID     string            `json:"external_id,omitempty"`
	Email          string            `json:"email"`
	Name           string            `json:"name,omitempty"`
	Attributes     map[string]string `json:"attributes,omitempty"`
	Status         string            `json:"status"`
	CreatedAt      *time.Time        `json:"created_at"`
	UpdatedAt      *time.Time        `json:"updated_at"`
	UnsubscribedAt *time.Time        `json:"unsubscribed_at,omitempty"`
	SuppressedAt   *time.Time        `json:"suppressed_at,omitempty"`
	ComplainedAt   *time.Time        `json:"complained_at,omitempty"`
}

type CreateContactRequest struct {
	Scope      Scope             `json:"scope"`
	ExternalID string            `json:"external_id,omitempty"`
	Email      string            `json:"email"`
	Name       string            `json:"name,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

type ContactTransitionRequest struct {
	Event string `json:"event"`
}

type Condition struct {
	Field    string   `json:"field"`
	Operator string   `json:"operator"`
	Value    string   `json:"value,omitempty"`
	Values   []string `json:"values,omitempty"`
}

type Segment struct {
	Scope       Scope       `json:"scope"`
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Match       string      `json:"match"`
	Conditions  []Condition `json:"conditions"`
	Status      string      `json:"status"`
	CreatedAt   *time.Time  `json:"created_at"`
	UpdatedAt   *time.Time  `json:"updated_at"`
}

type CreateSegmentRequest struct {
	Scope       Scope       `json:"scope"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Match       string      `json:"match"`
	Conditions  []Condition `json:"conditions"`
}

type Broadcast struct {
	Scope       Scope      `json:"scope"`
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	TemplateID  string     `json:"template_id"`
	SegmentID   string     `json:"segment_id"`
	Status      string     `json:"status"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
	Total       int        `json:"total"`
	Sent        int        `json:"sent"`
	Failed      int        `json:"failed"`
}

type CreateBroadcastRequest struct {
	Scope      Scope  `json:"scope"`
	Name       string `json:"name"`
	TemplateID string `json:"template_id"`
	SegmentID  string `json:"segment_id"`
}

type ScheduleBroadcastRequest struct {
	ScheduledAt time.Time `json:"scheduled_at"`
}

type AutomationTrigger struct {
	Kind      string            `json:"kind"`
	EventType string            `json:"event_type,omitempty"`
	Filter    map[string]string `json:"filter,omitempty"`
	Cron      string            `json:"cron,omitempty"`
	Timezone  string            `json:"timezone,omitempty"`
}

type AutomationAction struct {
	Kind       string            `json:"kind"`
	TemplateID string            `json:"template_id,omitempty"`
	Delay      int64             `json:"delay,omitempty"`
	Tag        string            `json:"tag,omitempty"`
	WebhookURL string            `json:"webhook_url,omitempty"`
	Payload    map[string]string `json:"payload,omitempty"`
	Condition  *Condition        `json:"condition,omitempty"`
	TrueStep   int               `json:"true_step,omitempty"`
	FalseStep  int               `json:"false_step,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
	EventType  string            `json:"event_type,omitempty"`
}

type AutomationStep struct {
	ID       string           `json:"id"`
	Position int              `json:"position"`
	Action   AutomationAction `json:"action"`
}

type Automation struct {
	Scope       Scope             `json:"scope"`
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Trigger     AutomationTrigger `json:"trigger"`
	Steps       []AutomationStep  `json:"steps"`
	Version     int               `json:"version"`
	Status      string            `json:"status"`
	CreatedAt   *time.Time        `json:"created_at"`
	UpdatedAt   *time.Time        `json:"updated_at"`
	ActivatedAt *time.Time        `json:"activated_at,omitempty"`
	ArchivedAt  *time.Time        `json:"archived_at,omitempty"`
}

type CreateAutomationRequest struct {
	Scope       Scope             `json:"scope"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Trigger     AutomationTrigger `json:"trigger"`
	Steps       []AutomationStep  `json:"steps"`
}

// UpdateAutomationRequest is the complete, scoped automation definition sent
// to PATCH /api/v1/automations/{automation_id}. The backend treats an update
// as a new definition revision, so callers send the full definition rather
// than a partial field patch. Scope is duplicated in the request body and URL
// query by Automations.Update to keep ownership explicit at both boundaries.
type UpdateAutomationRequest struct {
	Scope       Scope             `json:"scope"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Trigger     AutomationTrigger `json:"trigger"`
	Steps       []AutomationStep  `json:"steps"`
}

type StartAutomationRunRequest struct {
	ContactID string `json:"contact_id"`
	EventID   string `json:"event_id,omitempty"`
}

type AutomationRun struct {
	Scope          Scope      `json:"scope"`
	ID             string     `json:"id"`
	AutomationID   string     `json:"automation_id"`
	ContactID      string     `json:"contact_id"`
	EventID        string     `json:"event_id,omitempty"`
	IdempotencyKey string     `json:"idempotency_key"`
	Status         string     `json:"status"`
	StepIndex      int        `json:"step_index"`
	NextAt         *time.Time `json:"next_at,omitempty"`
	LastError      string     `json:"last_error,omitempty"`
	CreatedAt      *time.Time `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
}

type IngestAutomationEventRequest struct {
	EventID           string     `json:"event_id,omitempty"`
	EventType         string     `json:"event_type"`
	ContactID         string     `json:"contact_id,omitempty"`
	ContactEmail      string     `json:"email,omitempty"`
	ContactExternalID string     `json:"external_id,omitempty"`
	Payload           JSON       `json:"payload,omitempty"`
	OccurredAt        *time.Time `json:"occurred_at,omitempty"`
}

type AutomationEventResult struct {
	Scope      Scope      `json:"scope"`
	ID         string     `json:"id"`
	Type       string     `json:"type"`
	ContactID  string     `json:"contact_id"`
	Triggered  int        `json:"triggered"`
	RunIDs     []string   `json:"run_ids,omitempty"`
	OccurredAt *time.Time `json:"occurred_at"`
}

type AuditEvent struct {
	ID             string     `json:"id"`
	Actor          JSON       `json:"actor"`
	Action         string     `json:"action"`
	Outcome        string     `json:"outcome"`
	ResourceType   string     `json:"resource_type"`
	ResourceID     *string    `json:"resource_id,omitempty"`
	RequestID      string     `json:"request_id,omitempty"`
	CorrelationID  *string    `json:"correlation_id,omitempty"`
	CausationID    *string    `json:"causation_id,omitempty"`
	Metadata       JSON       `json:"metadata,omitempty"`
	BeforeDigest   string     `json:"before_digest,omitempty"`
	AfterDigest    string     `json:"after_digest,omitempty"`
	OccurredAt     *time.Time `json:"occurred_at,omitempty"`
	CreatedAt      *time.Time `json:"created_at,omitempty"`
	ObservedAt     *time.Time `json:"observed_at,omitempty"`
	RetentionUntil *time.Time `json:"retention_until,omitempty"`
	SchemaVersion  int        `json:"schema_version,omitempty"`
	Sequence       int64      `json:"sequence,omitempty"`
	PreviousHash   string     `json:"previous_hash,omitempty"`
	Hash           string     `json:"hash,omitempty"`
	LegalHold      bool       `json:"legal_hold,omitempty"`
	IdempotencyKey string     `json:"idempotency_key,omitempty"`
}

type AuditEventPage struct {
	Data       []AuditEvent `json:"data"`
	NextCursor *string      `json:"next_cursor,omitempty"`
}

type AppendAuditEventRequest struct {
	Actor         JSON    `json:"actor"`
	Action        string  `json:"action"`
	Outcome       string  `json:"outcome"`
	ResourceType  string  `json:"resource_type"`
	ResourceID    *string `json:"resource_id,omitempty"`
	RequestID     string  `json:"request_id,omitempty"`
	CorrelationID *string `json:"correlation_id,omitempty"`
	CausationID   *string `json:"causation_id,omitempty"`
	Metadata      JSON    `json:"metadata,omitempty"`
	BeforeDigest  string  `json:"before_digest,omitempty"`
	AfterDigest   string  `json:"after_digest,omitempty"`
}

type AuditHold struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Reason     string     `json:"reason"`
	CreatedAt  *time.Time `json:"created_at,omitempty"`
	ReleasedAt *time.Time `json:"released_at,omitempty"`
}

type AuditHoldList struct {
	Data []AuditHold `json:"data"`
}

type CreateAuditHoldRequest struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// Provider-neutral routing control-plane models. Field names intentionally
// follow the backend's camelCase JSON contract.
type RoutingPolicyRequest struct {
	AllowedRegions     []string `json:"allowedRegions"`
	PreferredRegion    string   `json:"preferredRegion"`
	RequiredResidency  string   `json:"requiredResidency"`
	RequireDedicatedIP bool     `json:"requireDedicatedIP"`
}

type RoutingPolicy struct {
	OrganizationID     string   `json:"organizationId"`
	AllowedRegions     []string `json:"allowedRegions"`
	PreferredRegion    string   `json:"preferredRegion,omitempty"`
	RequiredResidency  string   `json:"requiredResidency"`
	RequireDedicatedIP bool     `json:"requireDedicatedIP"`
}

type ProviderRoute struct {
	Region         string   `json:"region"`
	Residency      string   `json:"residency"`
	Available      bool     `json:"available"`
	Healthy        bool     `json:"healthy"`
	Weight         int      `json:"weight"`
	TrafficClasses []string `json:"trafficClasses,omitempty"`
}

type RoutingCapability struct {
	Provider             string          `json:"provider"`
	Routes               []ProviderRoute `json:"routes"`
	DedicatedIPSupported bool            `json:"dedicatedIpSupported"`
	DedicatedIPRegions   []string        `json:"dedicatedIpRegions"`
	ObservedAt           time.Time       `json:"observedAt"`
}

type WarmupRequest struct {
	StartAt              time.Time `json:"startAt"`
	InitialDailyCapacity int64     `json:"initialDailyCapacity"`
	DailyIncrement       int64     `json:"dailyIncrement"`
	MaxDailyCapacity     int64     `json:"maxDailyCapacity"`
}

type Warmup struct {
	StartAt              time.Time `json:"startAt"`
	InitialDailyCapacity int64     `json:"initialDailyCapacity"`
	DailyIncrement       int64     `json:"dailyIncrement"`
	MaxDailyCapacity     int64     `json:"maxDailyCapacity"`
}

type DedicatedIPRequest struct {
	Address    string `json:"address"`
	State      string `json:"state,omitempty"`
	Health     string `json:"health,omitempty"`
	Reputation int    `json:"reputation,omitempty"`
}

type DedicatedIP struct {
	ID            string     `json:"id"`
	PoolID        string     `json:"poolId"`
	Address       string     `json:"address"`
	State         string     `json:"state"`
	Health        string     `json:"health"`
	Reputation    int        `json:"reputation"`
	HoldUntil     *time.Time `json:"holdUntil,omitempty"`
	HoldReason    string     `json:"holdReason,omitempty"`
	DailySent     int64      `json:"dailySent"`
	DailySentDate time.Time  `json:"dailySentDate,omitempty"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type CreatePoolRequest struct {
	Provider      string               `json:"provider"`
	Region        string               `json:"region"`
	Residency     string               `json:"residency"`
	State         string               `json:"state,omitempty"`
	TrafficWeight int                  `json:"trafficWeight,omitempty"`
	Warmup        WarmupRequest        `json:"warmup"`
	IPs           []DedicatedIPRequest `json:"ips"`
}

type DedicatedPool struct {
	ID            string        `json:"id"`
	Provider      string        `json:"provider"`
	Region        string        `json:"region"`
	Residency     string        `json:"residency"`
	State         string        `json:"state"`
	Health        string        `json:"health"`
	TrafficWeight int           `json:"trafficWeight"`
	Warmup        Warmup        `json:"warmup"`
	HoldUntil     *time.Time    `json:"holdUntil,omitempty"`
	HoldReason    string        `json:"holdReason,omitempty"`
	IPs           []DedicatedIP `json:"ips"`
	Version       int64         `json:"version"`
	CreatedAt     time.Time     `json:"createdAt"`
	UpdatedAt     time.Time     `json:"updatedAt"`
}

type PoolCommandRequest struct {
	Action          string     `json:"action"`
	ExpectedVersion int64      `json:"expectedVersion,omitempty"`
	HoldUntil       *time.Time `json:"holdUntil,omitempty"`
	Reason          string     `json:"reason,omitempty"`
}

type IPCommandRequest struct {
	Action          string `json:"action"`
	ExpectedVersion int64  `json:"expectedVersion,omitempty"`
	Reason          string `json:"reason,omitempty"`
}

type RoutingTransition struct {
	ID         string    `json:"id"`
	EntityType string    `json:"entityType"`
	EntityID   string    `json:"entityId"`
	FromState  string    `json:"fromState"`
	ToState    string    `json:"toState"`
	Action     string    `json:"action"`
	CommandID  string    `json:"commandId"`
	Reason     string    `json:"reason,omitempty"`
	OccurredAt time.Time `json:"occurredAt"`
}

type RoutingAuditOptions struct {
	EntityType string
	EntityID   string
}

type RoutingCapabilities struct {
	Items []RoutingCapability `json:"items"`
}

type RoutingPools struct {
	Items []DedicatedPool `json:"items"`
}

type RoutingAudit struct {
	Items []RoutingTransition `json:"items"`
}
