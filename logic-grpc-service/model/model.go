package model

import "time"

type User struct {
	ID           int64  `gorm:"primaryKey"`
	Username     string
	Password     string
	Role         int32  `gorm:"column:role"` // Deprecated: kept for migration compatibility
	Email        string
	AccountType  string `gorm:"column:account_type;default:candidate"`
	Status       string `gorm:"column:status;default:active"`
	TokenVersion int32  `gorm:"column:token_version;default:1"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ── RBAC models ────────────────────────────────────────────────────────

// Role represents a named collection of permissions.
type Role struct {
	ID          uint64    `gorm:"primaryKey"`
	RoleKey     string    `gorm:"column:role_key;uniqueIndex:uk_roles_role_key"`
	Name        string    `gorm:"column:name"`
	Description string    `gorm:"column:description"`
	IsSystem    int32     `gorm:"column:is_system;default:1"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (Role) TableName() string { return "roles" }

// Permission represents a single protected business capability.
type Permission struct {
	ID            uint64    `gorm:"primaryKey"`
	PermissionKey string    `gorm:"column:permission_key;uniqueIndex:uk_permissions_permission_key"`
	Resource      string    `gorm:"column:resource"`
	Action        string    `gorm:"column:action"`
	Description   string    `gorm:"column:description"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

func (Permission) TableName() string { return "permissions" }

// RolePermission maps a permission to a role.
type RolePermission struct {
	ID           uint64    `gorm:"primaryKey"`
	RoleID       uint64    `gorm:"column:role_id;uniqueIndex:uk_role_permission"`
	PermissionID uint64    `gorm:"column:permission_id;uniqueIndex:uk_role_permission;index:idx_permission_id"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (RolePermission) TableName() string { return "role_permissions" }

// UserRole assigns a role to a user.
type UserRole struct {
	ID         uint64     `gorm:"primaryKey"`
	UserID     uint64     `gorm:"column:user_id;uniqueIndex:uk_user_role_active"`
	RoleID     uint64     `gorm:"column:role_id;uniqueIndex:uk_user_role_active;index:idx_user_roles_role"`
	AssignedBy *uint64    `gorm:"column:assigned_by"`
	AssignedAt time.Time  `gorm:"column:assigned_at"`
	RevokedAt  *time.Time `gorm:"column:revoked_at;uniqueIndex:uk_user_role_active"`
}

func (UserRole) TableName() string { return "user_roles" }

// UserDataScope constrains a user's permission usage to specific resources.
type UserDataScope struct {
	ID           uint64     `gorm:"primaryKey"`
	UserID       uint64     `gorm:"column:user_id;index:idx_user_scope"`
	ScopeKey     string     `gorm:"column:scope_key;index:idx_user_scope;index:idx_scope_resource"`
	ResourceType string     `gorm:"column:resource_type;index:idx_scope_resource"`
	ResourceID   uint64     `gorm:"column:resource_id;index:idx_scope_resource"`
	AssignedBy   *uint64    `gorm:"column:assigned_by"`
	AssignedAt   time.Time  `gorm:"column:assigned_at"`
	RevokedAt    *time.Time `gorm:"column:revoked_at;index:idx_user_scope"`
}

func (UserDataScope) TableName() string { return "user_data_scopes" }

// AuthorizationAuditLog records authorization decisions for audit purposes.
type AuthorizationAuditLog struct {
	ID             uint64    `gorm:"primaryKey"`
	ActorUserID    uint64    `gorm:"column:actor_user_id;index:idx_actor_created"`
	ActorRoles     string    `gorm:"column:actor_roles"`
	PermissionKey  string    `gorm:"column:permission_key;index:idx_permission_created"`
	ResourceType   string    `gorm:"column:resource_type"`
	ResourceID     uint64    `gorm:"column:resource_id"`
	Decision       string    `gorm:"column:decision;index:idx_decision_created"`
	Reason         string    `gorm:"column:reason"`
	RequestID      string    `gorm:"column:request_id"`
	ClientIP       string    `gorm:"column:client_ip"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}

func (AuthorizationAuditLog) TableName() string { return "authorization_audit_logs" }

type Job struct {
	ID           int64 `gorm:"primaryKey"`
	HrID         int64 `gorm:"column:hr_id"`
	Title        string
	Department   string
	DepartmentID *int64 `gorm:"column:department_id"`
	Location     string
	LocationID   *int64 `gorm:"column:location_id"`
	SalaryRange  string
	Description  string
	Requirements string
	Status       int32
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Department struct {
	ID               int64      `gorm:"primaryKey"`
	ParentID         int64      `gorm:"column:parent_id"`
	Name             string     `gorm:"column:name"`
	FullName         string     `gorm:"column:full_name"`
	Path             string     `gorm:"column:path"`
	Depth            int        `gorm:"column:depth"`
	SortOrder        int        `gorm:"column:sort_order"`
	IsActive         int32      `gorm:"column:is_active"`
	InheritLocations int32      `gorm:"column:inherit_locations"`
	CreatedBy        *int64     `gorm:"column:created_by"`
	UpdatedBy        *int64     `gorm:"column:updated_by"`
	DeletedAt        *time.Time `gorm:"column:deleted_at"`
	DeletedBy        *int64     `gorm:"column:deleted_by"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at"`
}

func (Department) TableName() string { return "departments" }

type JobLocation struct {
	ID        int64      `gorm:"primaryKey"`
	Name      string     `gorm:"column:name"`
	Code      *string    `gorm:"column:code"`
	SortOrder int        `gorm:"column:sort_order"`
	IsActive  int32      `gorm:"column:is_active"`
	CreatedBy *int64     `gorm:"column:created_by"`
	UpdatedBy *int64     `gorm:"column:updated_by"`
	DeletedAt *time.Time `gorm:"column:deleted_at"`
	DeletedBy *int64     `gorm:"column:deleted_by"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at"`
}

func (JobLocation) TableName() string { return "job_locations" }

type DepartmentLocation struct {
	ID           int64      `gorm:"primaryKey"`
	DepartmentID int64      `gorm:"column:department_id"`
	LocationID   int64      `gorm:"column:location_id"`
	IsActive     int32      `gorm:"column:is_active"`
	CreatedBy    *int64     `gorm:"column:created_by"`
	UpdatedBy    *int64     `gorm:"column:updated_by"`
	DeletedAt    *time.Time `gorm:"column:deleted_at"`
	DeletedBy    *int64     `gorm:"column:deleted_by"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at"`
}

func (DepartmentLocation) TableName() string { return "department_locations" }

type CandidateProfile struct {
	ID             int64  `gorm:"primaryKey"`
	UserID         int64  `gorm:"column:user_id"`
	RealName       string `gorm:"column:real_name"`
	Phone          string `gorm:"column:phone"`
	Education      string `gorm:"column:education"`
	School         string `gorm:"column:school"`
	WorkExperience string `gorm:"column:work_experience"`
	Skills         string `gorm:"column:skills"`
	IsComplete     int32  `gorm:"column:is_complete"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Resume struct {
	ID         int64 `gorm:"primaryKey"`
	UserID     int64
	OSSKey     string `gorm:"column:oss_key"`
	FileName   string
	FileType   string
	FileSize   int64
	ParsedText string `gorm:"column:parsed_text"`
	ParsedAt   *time.Time
	IsValid    int32
	UploadedAt time.Time
}

type Application struct {
	ID        int64  `gorm:"primaryKey"`
	JobID     int64  `gorm:"column:job_id"`
	UserID    int64  `gorm:"column:user_id"`
	ResumeID  int64  `gorm:"column:resume_id"`
	Status    int32  `gorm:"column:status"`
	StatusKey string `gorm:"column:status_key;default:applied;size:64"`
	RoundNo   int32  `gorm:"column:round_no"`
	IsCurrent int32  `gorm:"column:is_current"`
	AppliedAt time.Time `gorm:"column:applied_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (Application) TableName() string { return "applications" }

// ApplicationStatusTransition records every status change for audit trail purposes.
type ApplicationStatusTransition struct {
	ID              uint64    `gorm:"primaryKey"`
	ApplicationID   int64     `gorm:"column:application_id;index:idx_transition_app;not null"`
	FromStatus      string    `gorm:"column:from_status;size:64;not null"`
	ToStatus        string    `gorm:"column:to_status;size:64;not null"`
	ActorUserID     int64     `gorm:"column:actor_user_id;not null"`
	ActorAccountType string   `gorm:"column:actor_account_type;size:32;not null"`
	Reason          string    `gorm:"column:reason;size:512"`
	MetadataJSON    string    `gorm:"column:metadata_json;type:text"`
	CreatedAt       time.Time `gorm:"column:created_at"`
}

func (ApplicationStatusTransition) TableName() string { return "application_status_transitions" }

type AIChatHistory struct {
	ID        int64 `gorm:"primaryKey"`
	SessionID int64 `gorm:"column:session_id"`
	HrID      int64 `gorm:"column:hr_id"`
	OwnerRole int32 `gorm:"column:owner_role;default:2"` // 1 candidate / 2 HR
	OwnerID   int64 `gorm:"column:owner_id;default:0"`
	Role      string
	Content   string
	CreatedAt time.Time
}

type AIChatSession struct {
	ID            int64 `gorm:"primaryKey"`
	HrID          int64 `gorm:"column:hr_id"`
	OwnerRole     int32 `gorm:"column:owner_role;default:2"` // 1 candidate / 2 HR
	OwnerID       int64 `gorm:"column:owner_id;default:0"`
	Title         string
	ApplicationID int64 `gorm:"column:application_id"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time `gorm:"column:deleted_at"`
}

type AISessionSummary struct {
	ID               uint64 `gorm:"primaryKey"`
	SessionID        uint64 `gorm:"column:session_id"`
	HrID             uint64 `gorm:"column:hr_id"`
	Summary          string `gorm:"column:summary"`
	CoveredMessageID uint64 `gorm:"column:covered_message_id"`
	MessageCount     int    `gorm:"column:message_count"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type AIToolTrace struct {
	ID            uint64 `gorm:"primaryKey"`
	SessionID     uint64 `gorm:"column:session_id"`
	HrID          uint64 `gorm:"column:hr_id"`
	ToolCallID    string `gorm:"column:tool_call_id"`
	ToolName      string `gorm:"column:tool_name"`
	ArgumentsJSON string `gorm:"column:arguments_json"`
	ResultJSON    string `gorm:"column:result_json"`
	ResultSummary string `gorm:"column:result_summary"`
	Status        string `gorm:"column:status"`
	ErrorMessage  string `gorm:"column:error_message"`
	CreatedAt     time.Time
}

type AIMemory struct {
	ID         uint64     `gorm:"primaryKey"`
	HrID       uint64     `gorm:"column:hr_id"`
	ScopeType  string     `gorm:"column:scope_type"`
	ScopeID    uint64     `gorm:"column:scope_id"`
	MemoryType string     `gorm:"column:memory_type"`
	Content    string     `gorm:"column:content"`
	Source     string     `gorm:"column:source"`
	Confidence float64    `gorm:"column:confidence"`
	ExpiresAt  *time.Time `gorm:"column:expires_at"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (CandidateProfile) TableName() string { return "candidate_profiles" }
func (AIChatHistory) TableName() string    { return "ai_chat_history" }
func (AIChatSession) TableName() string    { return "ai_chat_sessions" }
func (AISessionSummary) TableName() string { return "ai_session_summaries" }
func (AIToolTrace) TableName() string      { return "ai_tool_traces" }
func (AIMemory) TableName() string         { return "ai_memories" }
func (Notification) TableName() string     { return "notifications" }

type Notification struct {
	ID                  int64      `gorm:"primaryKey"`
	EventID             *string    `gorm:"column:event_id;uniqueIndex:uk_notification_event_id"`
	ReceiverID          int64      `gorm:"column:receiver_id;uniqueIndex:uk_notification_once,priority:1"`
	ReceiverAccountType string     `gorm:"column:receiver_account_type;uniqueIndex:uk_notification_once,priority:2"`
	ReceiverRole        int32      `gorm:"column:receiver_role"` // Deprecated: use ReceiverAccountType
	Type                string     `gorm:"column:type;uniqueIndex:uk_notification_once,priority:5"`
	Title               string     `gorm:"column:title"`
	Content             string     `gorm:"column:content"`
	Link                string     `gorm:"column:link"`
	BizType             string     `gorm:"column:biz_type;uniqueIndex:uk_notification_once,priority:3"`
	BizID               int64      `gorm:"column:biz_id;uniqueIndex:uk_notification_once,priority:4"`
	IsRead              int32      `gorm:"column:is_read"`
	CreatedAt           time.Time  `gorm:"column:created_at"`
	ReadAt              *time.Time `gorm:"column:read_at"`
}

const (
	EventOutboxStatusPending    int32 = 0
	EventOutboxStatusPublished  int32 = 1
	EventOutboxStatusDead       int32 = 2
	EventOutboxStatusProcessing int32 = 3
)

// EventOutbox table: event_outbox
type EventOutbox struct {
	ID            uint64     `gorm:"primaryKey"`
	EventID       string     `gorm:"column:event_id"`
	EventType     string     `gorm:"column:event_type"`
	AggregateType string     `gorm:"column:aggregate_type"`
	AggregateID   uint64     `gorm:"column:aggregate_id"`
	RoutingKey    string     `gorm:"column:routing_key"`
	Payload       string     `gorm:"column:payload;type:json"`
	Status        int32      `gorm:"column:status;default:0"`
	RetryCount    int32      `gorm:"column:retry_count;default:0"`
	NextRetryAt   *time.Time `gorm:"column:next_retry_at"`
	LastError     string     `gorm:"column:last_error"`
	LockedAt      *time.Time `gorm:"column:locked_at"`
	LockedBy      string     `gorm:"column:locked_by"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
}

func (EventOutbox) TableName() string { return "event_outbox" }

type InviteCode struct {
	ID        int64      `gorm:"primaryKey"`
	Code      string     `gorm:"column:code"`
	CreatedBy int64      `gorm:"column:created_by"`
	ExpiresAt *time.Time `gorm:"column:expires_at"`
	IsActive  int32      `gorm:"column:is_active;default:1"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at"`
}

func (InviteCode) TableName() string { return "invite_codes" }

type ThirdPartyUsageLog struct {
	ID              int64     `gorm:"primaryKey"`
	UserID          int64     `gorm:"column:user_id"`
	Role            int32     `gorm:"column:role"`
	ServiceType     string    `gorm:"column:service_type"`
	Endpoint        string    `gorm:"column:endpoint"`
	Provider        string    `gorm:"column:provider"`
	Model           string    `gorm:"column:model"`
	RequestChars    int       `gorm:"column:request_chars"`
	ResponseChars   int       `gorm:"column:response_chars"`
	EstimatedTokens int       `gorm:"column:estimated_tokens"`
	ObjectKey       string    `gorm:"column:object_key"`
	ObjectSize      int64     `gorm:"column:object_size"`
	Status          string    `gorm:"column:status"`
	ErrorCode       string    `gorm:"column:error_code"`
	CostMs          int       `gorm:"column:cost_ms"`
	RequestID       string    `gorm:"column:request_id"`
	IP              string    `gorm:"column:ip"`
	CreatedAt       time.Time `gorm:"column:created_at"`
}

func (ThirdPartyUsageLog) TableName() string { return "third_party_usage_logs" }

// ── Interview Schedule ─────────────────────────────────────────────────────

type InterviewSchedule struct {
	ID              int64      `gorm:"primaryKey"`
	ApplicationID   int64      `gorm:"column:application_id;not null"`
	InterviewerID   int64      `gorm:"column:interviewer_id;not null"`
	RoundNo         int32      `gorm:"column:round_no;default:1"`
	ScheduledAt     *time.Time `gorm:"column:scheduled_at"`
	Status          string     `gorm:"column:status;default:pending;size:32"`
	CreatedBy       *int64     `gorm:"column:created_by"`
	Title           string     `gorm:"column:title;size:128"`
	Mode            string     `gorm:"column:mode;size:32"`           // video / phone / onsite
	MeetingURL      string     `gorm:"column:meeting_url;size:512"`
	Location        string     `gorm:"column:location;size:256"`
	DurationMinutes int32      `gorm:"column:duration_minutes"`
	CandidateNote   string     `gorm:"column:candidate_note;size:1024"`
	InternalNote    string     `gorm:"column:internal_note;size:1024"`
	CancelReason    string     `gorm:"column:cancel_reason;size:512"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
	DeletedAt       *time.Time `gorm:"column:deleted_at"`
}

func (InterviewSchedule) TableName() string { return "interview_schedules" }

// ── Interview Feedback ─────────────────────────────────────────────────────

type InterviewFeedback struct {
	ID                int64      `gorm:"primaryKey"`
	InterviewID       int64      `gorm:"column:interview_id;uniqueIndex:uk_interview_feedback_once,priority:1"`
	ApplicationID     int64      `gorm:"column:application_id;not null;uniqueIndex:uk_interview_feedback_once,priority:2"`
	InterviewerID     int64      `gorm:"column:interviewer_id;not null;uniqueIndex:uk_interview_feedback_once,priority:3"`
	Recommendation    string     `gorm:"column:recommendation;size:32"` // positive / negative / pending
	Score             int32      `gorm:"column:score"`
	DimensionScoresJSON string   `gorm:"column:dimension_scores_json;type:text"`
	Comments          string     `gorm:"column:comments;type:text"`
	SubmittedAt       time.Time  `gorm:"column:submitted_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at"`
}

func (InterviewFeedback) TableName() string { return "interview_feedback" }

// ── Offer ─────────────────────────────────────────────────────────────────────

type Offer struct {
	ID               int64      `gorm:"primaryKey"`
	ApplicationID    int64      `gorm:"column:application_id;not null;index:idx_offer_application"`
	CandidateUserID  int64      `gorm:"column:candidate_user_id;not null;index:idx_offer_candidate"`
	JobID            int64      `gorm:"column:job_id;not null;index:idx_offer_job"`
	Status           string     `gorm:"column:status;size:32;default:draft;index:idx_offer_status"`
	Title            string     `gorm:"column:title;size:128;not null"`
	SalaryRange      string     `gorm:"column:salary_range;size:64"`
	Level            string     `gorm:"column:level;size:64"`
	WorkLocation     string     `gorm:"column:work_location;size:128"`
	StartDate        string     `gorm:"column:start_date;size:32"`
	ExpiresAt        *time.Time `gorm:"column:expires_at"`
	TermsJSON        string     `gorm:"column:terms_json;type:text"`
	SentSnapshotJSON string     `gorm:"column:sent_snapshot_json;type:text"`
	CreatedBy        int64      `gorm:"column:created_by;not null;index:idx_offer_created_by"`
	SentBy           *int64     `gorm:"column:sent_by"`
	DecidedAt        *time.Time `gorm:"column:decided_at"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at"`
}

func (Offer) TableName() string { return "offers" }

type OfferEvent struct {
	ID               uint64    `gorm:"primaryKey"`
	OfferID          int64     `gorm:"column:offer_id;not null;index:idx_offer_event_offer"`
	EventType        string    `gorm:"column:event_type;size:64;not null;index:idx_offer_event_type"`
	ActorUserID      int64     `gorm:"column:actor_user_id;not null"`
	ActorAccountType string    `gorm:"column:actor_account_type;size:32;not null"`
	Reason           string    `gorm:"column:reason;size:512"`
	MetadataJSON     string    `gorm:"column:metadata_json;type:text"`
	CreatedAt        time.Time `gorm:"column:created_at"`
}

func (OfferEvent) TableName() string { return "offer_events" }

// ── Phase 4: Candidate Collaboration ──────────────────────────────────────────────────

type CandidateNote struct {
	ID              uint64    `gorm:"primaryKey"`
	CandidateUserID uint64    `gorm:"column:candidate_user_id;not null"`
	ApplicationID   *uint64   `gorm:"column:application_id"`
	AuthorUserID    uint64    `gorm:"column:author_user_id;not null"`
	Content         string    `gorm:"column:content;type:text;not null"`
	Visibility      string    `gorm:"column:visibility;size:32;default:internal"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at"`
}

func (CandidateNote) TableName() string { return "candidate_notes" }

type CandidateTag struct {
	ID        uint64    `gorm:"primaryKey"`
	Name      string    `gorm:"column:name;size:64;not null;uniqueIndex:uk_tag_name"`
	Color     string    `gorm:"column:color;size:16;default:#409eff"`
	CreatedBy *uint64   `gorm:"column:created_by"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (CandidateTag) TableName() string { return "candidate_tags" }

type CandidateTagAssignment struct {
	ID              uint64    `gorm:"primaryKey"`
	TagID           uint64    `gorm:"column:tag_id;not null;uniqueIndex:uk_tag_candidate,priority:1"`
	CandidateUserID uint64    `gorm:"column:candidate_user_id;not null;uniqueIndex:uk_tag_candidate,priority:2"`
	CreatedBy       *uint64   `gorm:"column:created_by"`
	CreatedAt       time.Time `gorm:"column:created_at"`
}

func (CandidateTagAssignment) TableName() string { return "candidate_tag_assignments" }

type FollowUpTask struct {
	ID              uint64     `gorm:"primaryKey"`
	CandidateUserID uint64     `gorm:"column:candidate_user_id;not null"`
	ApplicationID   *uint64    `gorm:"column:application_id"`
	AssigneeUserID  uint64     `gorm:"column:assignee_user_id;not null"`
	CreatedBy       uint64     `gorm:"column:created_by;not null"`
	Title           string     `gorm:"column:title;size:256;not null"`
	Description     string     `gorm:"column:description;type:text"`
	DueAt           *time.Time `gorm:"column:due_at"`
	Status          string     `gorm:"column:status;size:32;default:pending"`
	CompletedAt     *time.Time `gorm:"column:completed_at"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
}

func (FollowUpTask) TableName() string { return "follow_up_tasks" }
