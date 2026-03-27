package main

const (
	templateTypeCommon     = "common"
	templateTypeDepartment = "department"

	sendModeAutomatic = "automatic"
	sendModeManual    = "manual"

	sendStatusSent    = "sent"
	sendStatusSkipped = "skipped"
	sendStatusFailed  = "failed"

	queueStatusPending = "pending"
	queueStatusFailed  = "failed"

	exclusionRuleEmailDomain = "email_domain"
	exclusionRuleUsername    = "username"
	exclusionRuleUserID      = "user_id"
	exclusionRuleGroupName   = "group_name"
	exclusionRuleGroupID     = "group_id"
	exclusionRuleIsBot       = "is_bot"
	exclusionRuleDeleted     = "deleted"
)

type onboardingTemplate struct {
	ID             string `json:"template_id"`
	Type           string `json:"template_type"`
	DepartmentCode string `json:"dept_code,omitempty"`
	Lang           string `json:"lang"`
	Title          string `json:"title"`
	Body           string `json:"body"`
	Active         bool   `json:"active_yn"`
	Version        int    `json:"version"`
	UpdatedBy      string `json:"updated_by,omitempty"`
	UpdatedAt      string `json:"updated_at,omitempty"`
}

type onboardingLink struct {
	ID          string `json:"link_id"`
	TemplateID  string `json:"template_id"`
	Title       string `json:"link_title"`
	Description string `json:"link_desc"`
	URL         string `json:"link_url"`
	SortOrder   int    `json:"sort_order"`
	Active      bool   `json:"active_yn"`
}

type departmentMapping struct {
	SourceDepartmentCode   string `json:"source_dept_code"`
	SourceDepartmentName   string `json:"source_dept_name"`
	TemplateDepartmentCode string `json:"template_dept_code"`
	DepartmentName         string `json:"template_dept_name,omitempty"`
	OrganizationName       string `json:"organization_name,omitempty"`
	Active                 bool   `json:"active_yn"`
}

type onboardingExclusionRule struct {
	ID        string `json:"exclusion_id"`
	RuleType  string `json:"rule_type"`
	RuleValue string `json:"rule_value"`
	Active    bool   `json:"active_yn"`
}

type onboardingSendLog struct {
	ID             string   `json:"log_id"`
	UserID         string   `json:"user_id"`
	Username       string   `json:"username"`
	DepartmentCode string   `json:"dept_code,omitempty"`
	TemplateIDs    []string `json:"template_ids"`
	SentAt         int64    `json:"sent_at"`
	Status         string   `json:"status"`
	ErrorMessage   string   `json:"error_message,omitempty"`
	Mode           string   `json:"mode"`
	RequestedBy    string   `json:"requested_by,omitempty"`
}

type onboardingQueueItem struct {
	ID            string `json:"queue_id"`
	TargetUserID  string `json:"target_user_id"`
	RequestedBy   string `json:"requested_by"`
	RequestedAt   int64  `json:"requested_at"`
	Status        string `json:"status"`
	AttemptCount  int    `json:"attempt_count"`
	LastError     string `json:"last_error,omitempty"`
	NextAttemptAt int64  `json:"next_attempt_at"`
}

type sendState struct {
	Status    string `json:"status"`
	LogID     string `json:"log_id,omitempty"`
	UpdatedAt int64  `json:"updated_at"`
}

type runtimeConfiguration struct {
	Enabled                bool
	SenderBotUsername      string
	SenderBotDisplayName   string
	DefaultLanguage        string
	FallbackDepartmentCode string
	InitialDelaySeconds    int
	RetryIntervalMinutes   int
	RetryMaxAttempts       int
	Templates              []onboardingTemplate
	Links                  []onboardingLink
	Mappings               []departmentMapping
	Exclusions             []onboardingExclusionRule
}

type resolvedOnboardingProfile struct {
	DepartmentCode string
	DepartmentName string
	Organization   string
	Language       string
	StartDate      string
}

type onboardingRenderResult struct {
	Message               string
	Language              string
	DepartmentCode        string
	DepartmentName        string
	CommonTemplateID      string
	DepartmentTemplateID  string
	AppliedTemplateIDs    []string
	SkippedLinkCount      int
}

type onboardingStats struct {
	TotalSends     int64            `json:"total_sends"`
	Successful     int64            `json:"successful"`
	Failed         int64            `json:"failed"`
	Skipped        int64            `json:"skipped"`
	ManualResends  int64            `json:"manual_resends"`
	ByDepartment   map[string]int64 `json:"by_department"`
	RecentFailures []string         `json:"recent_failures"`
}
