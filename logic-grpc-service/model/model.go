package model

import "time"

type User struct {
	ID           int64 `gorm:"primaryKey"`
	Username     string
	Password     string
	Role         int32 `gorm:"column:role"` // Deprecated: kept for migration compatibility
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
	ID            uint64    `gorm:"primaryKey"`
	ActorUserID   uint64    `gorm:"column:actor_user_id;index:idx_actor_created"`
	ActorRoles    string    `gorm:"column:actor_roles"`
	PermissionKey string    `gorm:"column:permission_key;index:idx_permission_created"`
	ResourceType  string    `gorm:"column:resource_type"`
	ResourceID    uint64    `gorm:"column:resource_id"`
	Decision      string    `gorm:"column:decision;index:idx_decision_created"`
	Reason        string    `gorm:"column:reason"`
	RequestID     string    `gorm:"column:request_id"`
	ClientIP      string    `gorm:"column:client_ip"`
	CreatedAt     time.Time `gorm:"column:created_at"`
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

type ResumeParseRun struct {
	ID            uint64     `gorm:"primaryKey"`
	ResumeID      int64      `gorm:"column:resume_id;not null;index:idx_resume_parse_runs_resume"`
	UserID        int64      `gorm:"column:user_id;not null;index:idx_resume_parse_runs_user"`
	AgentRunID    *uint64    `gorm:"column:agent_run_id;index:idx_resume_parse_runs_agent_run"`
	Status        string     `gorm:"column:status;size:32;not null;default:running;index:idx_resume_parse_runs_status"`
	ParserVersion string     `gorm:"column:parser_version;size:64"`
	InputHash     string     `gorm:"column:input_hash;size:128"`
	ErrorMessage  string     `gorm:"column:error_message;type:text"`
	StartedAt     time.Time  `gorm:"column:started_at"`
	CompletedAt   *time.Time `gorm:"column:completed_at"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
}

func (ResumeParseRun) TableName() string { return "resume_parse_runs" }

type ResumeProfile struct {
	ID              uint64    `gorm:"primaryKey"`
	ResumeID        int64     `gorm:"column:resume_id;not null;index:idx_resume_profiles_resume;uniqueIndex:uk_resume_profile_version,priority:1"`
	UserID          int64     `gorm:"column:user_id;not null;index:idx_resume_profiles_user"`
	ParseRunID      uint64    `gorm:"column:parse_run_id;not null;uniqueIndex:uk_resume_profiles_parse_run"`
	Version         int32     `gorm:"column:version;not null;default:1;uniqueIndex:uk_resume_profile_version,priority:2"`
	IsCurrent       int32     `gorm:"column:is_current;not null;default:1;index:idx_resume_profiles_current"`
	CurrentKey      *int32    `gorm:"column:current_key;->"`
	FullName        string    `gorm:"column:full_name;size:128"`
	Email           string    `gorm:"column:email;size:128"`
	Phone           string    `gorm:"column:phone;size:64"`
	Location        string    `gorm:"column:location;size:128"`
	Headline        string    `gorm:"column:headline;size:256"`
	Summary         string    `gorm:"column:summary;type:text"`
	TotalExperience float64   `gorm:"column:total_experience_years"`
	HighestDegree   string    `gorm:"column:highest_degree;size:64"`
	RawJSON         string    `gorm:"column:raw_json;type:text"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at"`
}

func (ResumeProfile) TableName() string { return "resume_profiles" }

type ResumeEducation struct {
	ID              uint64     `gorm:"primaryKey"`
	ResumeProfileID uint64     `gorm:"column:resume_profile_id;not null;index:idx_resume_educations_profile"`
	School          string     `gorm:"column:school;size:128;not null"`
	Degree          string     `gorm:"column:degree;size:64"`
	Major           string     `gorm:"column:major;size:128"`
	StartDate       *time.Time `gorm:"column:start_date"`
	EndDate         *time.Time `gorm:"column:end_date"`
	Description     string     `gorm:"column:description;type:text"`
	SortOrder       int32      `gorm:"column:sort_order;not null;default:0"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
}

func (ResumeEducation) TableName() string { return "resume_educations" }

type ResumeExperience struct {
	ID               uint64     `gorm:"primaryKey"`
	ResumeProfileID  uint64     `gorm:"column:resume_profile_id;not null;index:idx_resume_experiences_profile"`
	Company          string     `gorm:"column:company;size:128;not null"`
	Title            string     `gorm:"column:title;size:128"`
	Location         string     `gorm:"column:location;size:128"`
	StartDate        *time.Time `gorm:"column:start_date"`
	EndDate          *time.Time `gorm:"column:end_date"`
	IsCurrent        int32      `gorm:"column:is_current;not null;default:0"`
	Description      string     `gorm:"column:description;type:text"`
	AchievementsJSON string     `gorm:"column:achievements_json;type:text"`
	SortOrder        int32      `gorm:"column:sort_order;not null;default:0"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at"`
}

func (ResumeExperience) TableName() string { return "resume_experiences" }

type ResumeProject struct {
	ID               uint64     `gorm:"primaryKey"`
	ResumeProfileID  uint64     `gorm:"column:resume_profile_id;not null;index:idx_resume_projects_profile"`
	Name             string     `gorm:"column:name;size:128;not null"`
	Role             string     `gorm:"column:role;size:128"`
	StartDate        *time.Time `gorm:"column:start_date"`
	EndDate          *time.Time `gorm:"column:end_date"`
	Description      string     `gorm:"column:description;type:text"`
	TechnologiesJSON string     `gorm:"column:technologies_json;type:text"`
	HighlightsJSON   string     `gorm:"column:highlights_json;type:text"`
	SortOrder        int32      `gorm:"column:sort_order;not null;default:0"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at"`
}

func (ResumeProject) TableName() string { return "resume_projects" }

type ResumeSkill struct {
	ID              uint64    `gorm:"primaryKey"`
	ResumeProfileID uint64    `gorm:"column:resume_profile_id;not null;index:idx_resume_skills_profile;uniqueIndex:uk_resume_skill_name,priority:1"`
	Name            string    `gorm:"column:name;size:128;not null;uniqueIndex:uk_resume_skill_name,priority:2"`
	Category        string    `gorm:"column:category;size:64;index:idx_resume_skills_category"`
	Level           string    `gorm:"column:level;size:32"`
	Years           float64   `gorm:"column:years"`
	Evidence        string    `gorm:"column:evidence;type:text"`
	SortOrder       int32     `gorm:"column:sort_order;not null;default:0"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at"`
}

func (ResumeSkill) TableName() string { return "resume_skills" }

type CandidateMatchEvaluation struct {
	ID                 uint64    `gorm:"primaryKey"`
	ApplicationID      int64     `gorm:"column:application_id;not null;index:idx_candidate_match_application;uniqueIndex:uk_candidate_match_version,priority:1"`
	JobID              int64     `gorm:"column:job_id;not null;index:idx_candidate_match_job"`
	CandidateUserID    int64     `gorm:"column:candidate_user_id;not null;index:idx_candidate_match_candidate"`
	ResumeProfileID    uint64    `gorm:"column:resume_profile_id;not null;index:idx_candidate_match_resume_profile"`
	AgentRunID         *uint64   `gorm:"column:agent_run_id;index:idx_candidate_match_agent_run"`
	EvaluationVersion  int32     `gorm:"column:evaluation_version;not null;default:1;uniqueIndex:uk_candidate_match_version,priority:2"`
	IsLatest           int32     `gorm:"column:is_latest;not null;default:1;index:idx_candidate_match_latest"`
	LatestKey          *int32    `gorm:"column:latest_key;->"`
	OverallScore       float64   `gorm:"column:overall_score"`
	Recommendation     string    `gorm:"column:recommendation;size:32"`
	Summary            string    `gorm:"column:summary;type:text"`
	StrengthsJSON      string    `gorm:"column:strengths_json;type:text"`
	RisksJSON          string    `gorm:"column:risks_json;type:text"`
	ScoreBreakdownJSON string    `gorm:"column:score_breakdown_json;type:text"`
	ModelName          string    `gorm:"column:model_name;size:128"`
	EvaluatedAt        time.Time `gorm:"column:evaluated_at"`
	CreatedAt          time.Time `gorm:"column:created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at"`
}

func (CandidateMatchEvaluation) TableName() string { return "candidate_match_evaluations" }

type CandidateMatchEvidence struct {
	ID           uint64    `gorm:"primaryKey"`
	EvaluationID uint64    `gorm:"column:evaluation_id;not null;index:idx_candidate_match_evidence_eval"`
	EvidenceType string    `gorm:"column:evidence_type;size:64;not null;index:idx_candidate_match_evidence_type"`
	Dimension    string    `gorm:"column:dimension;size:64"`
	SourceTable  string    `gorm:"column:source_table;size:64"`
	SourceID     *uint64   `gorm:"column:source_id"`
	Snippet      string    `gorm:"column:snippet;type:text"`
	Weight       float64   `gorm:"column:weight"`
	ScoreImpact  float64   `gorm:"column:score_impact"`
	MetadataJSON string    `gorm:"column:metadata_json;type:text"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (CandidateMatchEvidence) TableName() string { return "candidate_match_evidence" }

type Application struct {
	ID        int64     `gorm:"primaryKey"`
	JobID     int64     `gorm:"column:job_id"`
	UserID    int64     `gorm:"column:user_id"`
	ResumeID  int64     `gorm:"column:resume_id"`
	Status    int32     `gorm:"column:status"`
	StatusKey string    `gorm:"column:status_key;default:applied;size:64"`
	RoundNo   int32     `gorm:"column:round_no"`
	IsCurrent int32     `gorm:"column:is_current"`
	AppliedAt time.Time `gorm:"column:applied_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (Application) TableName() string { return "applications" }

// ApplicationStatusTransition records every status change for audit trail purposes.
type ApplicationStatusTransition struct {
	ID               uint64    `gorm:"primaryKey"`
	ApplicationID    int64     `gorm:"column:application_id;index:idx_transition_app;not null"`
	FromStatus       string    `gorm:"column:from_status;size:64;not null"`
	ToStatus         string    `gorm:"column:to_status;size:64;not null"`
	ActorUserID      int64     `gorm:"column:actor_user_id;not null"`
	ActorAccountType string    `gorm:"column:actor_account_type;size:32;not null"`
	Reason           string    `gorm:"column:reason;size:512"`
	MetadataJSON     string    `gorm:"column:metadata_json;type:text"`
	CreatedAt        time.Time `gorm:"column:created_at"`
}

func (ApplicationStatusTransition) TableName() string { return "application_status_transitions" }

type AIChatHistory struct {
	ID                  int64 `gorm:"primaryKey"`
	SessionID           int64 `gorm:"column:session_id"`
	HrID                int64 `gorm:"column:hr_id"`
	OwnerRole           int32 `gorm:"column:owner_role;default:2"` // 1 candidate / 2 HR
	OwnerID             int64 `gorm:"column:owner_id;default:0"`
	Role                string
	Content             string
	ProcessContent      string `gorm:"column:process_content;type:text"`
	ContextUsageJSON    string `gorm:"column:context_usage_json;type:text"`
	ModelID             *int64 `gorm:"column:model_id"`
	ModelName           string `gorm:"column:model_name;size:128"`
	AgentSkillIDsJSON   string `gorm:"column:agent_skill_ids_json;type:text"`
	AgentSkillNamesJSON string `gorm:"column:agent_skill_names_json;type:text"`
	CreatedAt           time.Time
}

type AIChatSession struct {
	ID                     int64 `gorm:"primaryKey"`
	HrID                   int64 `gorm:"column:hr_id"`
	OwnerRole              int32 `gorm:"column:owner_role;default:2"` // 1 candidate / 2 HR
	OwnerID                int64 `gorm:"column:owner_id;default:0"`
	Title                  string
	ApplicationID          int64  `gorm:"column:application_id"`
	LatestContextUsageJSON string `gorm:"column:latest_context_usage_json;type:text"`
	CreatedAt              time.Time
	UpdatedAt              time.Time
	DeletedAt              *time.Time `gorm:"column:deleted_at"`
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
	ID             uint64  `gorm:"primaryKey"`
	SessionID      uint64  `gorm:"column:session_id"`
	HrID           uint64  `gorm:"column:hr_id"`
	AgentRunID     *uint64 `gorm:"column:agent_run_id"`
	AgentRunStepID *uint64 `gorm:"column:agent_run_step_id"`
	ToolCallID     string  `gorm:"column:tool_call_id"`
	ToolName       string  `gorm:"column:tool_name"`
	ArgumentsJSON  string  `gorm:"column:arguments_json"`
	ResultJSON     string  `gorm:"column:result_json"`
	ResultSummary  string  `gorm:"column:result_summary"`
	Status         string  `gorm:"column:status"`
	DurationMs     int64   `gorm:"column:duration_ms"`
	ErrorMessage   string  `gorm:"column:error_message"`
	CreatedAt      time.Time
}

type AgentRun struct {
	ID           uint64     `gorm:"primaryKey"`
	SessionID    uint64     `gorm:"column:session_id"`
	MessageID    *uint64    `gorm:"column:message_id"`
	HistoryID    *uint64    `gorm:"column:history_id"`
	HrID         uint64     `gorm:"column:hr_id"`
	AgentType    string     `gorm:"column:agent_type"`
	AgentID      *uint64    `gorm:"column:agent_id"`
	AgentName    string     `gorm:"column:agent_name"`
	ModelID      *uint64    `gorm:"column:model_id"`
	ModelName    string     `gorm:"column:model_name"`
	Status       string     `gorm:"column:status"`
	PlanJSON     string     `gorm:"column:plan_json"`
	FinalAnswer  string     `gorm:"column:final_answer"`
	ErrorType    string     `gorm:"column:error_type"`
	ErrorMessage string     `gorm:"column:error_message"`
	StartedAt    time.Time  `gorm:"column:started_at"`
	CompletedAt  *time.Time `gorm:"column:completed_at"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type AgentRunStep struct {
	ID               uint64     `gorm:"primaryKey"`
	RunID            uint64     `gorm:"column:run_id"`
	StepIndex        int        `gorm:"column:step_index"`
	StepType         string     `gorm:"column:step_type"`
	CapabilitySource string     `gorm:"column:capability_source"`
	CapabilityKey    string     `gorm:"column:capability_key"`
	ToolName         string     `gorm:"column:tool_name"`
	InputJSON        string     `gorm:"column:input_json"`
	OutputJSON       string     `gorm:"column:output_json"`
	Status           string     `gorm:"column:status"`
	DurationMs       int64      `gorm:"column:duration_ms"`
	ErrorMessage     string     `gorm:"column:error_message"`
	StartedAt        time.Time  `gorm:"column:started_at"`
	CompletedAt      *time.Time `gorm:"column:completed_at"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
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
	Importance float64    `gorm:"column:importance"`
	ExpiresAt  *time.Time `gorm:"column:expires_at"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type AIEmbedding struct {
	ID             uint64    `gorm:"primaryKey"`
	ObjectType     string    `gorm:"column:object_type;uniqueIndex:uk_ai_embeddings_object_model_hash,priority:1;index:idx_ai_embeddings_object;index:idx_ai_embeddings_query,priority:1"`
	ObjectID       uint64    `gorm:"column:object_id;uniqueIndex:uk_ai_embeddings_object_model_hash,priority:2;index:idx_ai_embeddings_object"`
	ScopeType      string    `gorm:"column:scope_type;index:idx_ai_embeddings_scope,priority:1"`
	ScopeID        uint64    `gorm:"column:scope_id;index:idx_ai_embeddings_scope,priority:2"`
	TextHash       string    `gorm:"column:text_hash;uniqueIndex:uk_ai_embeddings_object_model_hash,priority:4"`
	EmbeddingModel string    `gorm:"column:embedding_model;uniqueIndex:uk_ai_embeddings_object_model_hash,priority:3;index:idx_ai_embeddings_query,priority:2"`
	EmbeddingDim   int       `gorm:"column:embedding_dim"`
	VectorJSON     *string   `gorm:"column:vector_json;type:json"`
	MetadataJSON   *string   `gorm:"column:metadata_json;type:json"`
	Status         string    `gorm:"column:status;index:idx_ai_embeddings_query,priority:3"`
	LastError      string    `gorm:"column:last_error"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

func (CandidateProfile) TableName() string { return "candidate_profiles" }
func (AIChatHistory) TableName() string    { return "ai_chat_history" }
func (AIChatSession) TableName() string    { return "ai_chat_sessions" }
func (AISessionSummary) TableName() string { return "ai_session_summaries" }
func (AIToolTrace) TableName() string      { return "ai_tool_traces" }
func (AgentRun) TableName() string         { return "agent_runs" }
func (AgentRunStep) TableName() string     { return "agent_run_steps" }
func (AIMemory) TableName() string         { return "ai_memories" }
func (AIEmbedding) TableName() string      { return "ai_embeddings" }
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

// EmailLog records email send attempts for audit and idempotency.
type EmailLog struct {
	ID        int64     `gorm:"primaryKey"`
	EventID   string    `gorm:"column:event_id;uniqueIndex:uk_email_event_id"`
	UserID    int64     `gorm:"column:user_id"`
	Email     string    `gorm:"column:email"`
	Type      string    `gorm:"column:type"`
	Subject   string    `gorm:"column:subject"`
	Status    string    `gorm:"column:status;default:sent"`
	ErrorMsg  *string   `gorm:"column:error_msg"`
	SentAt    time.Time `gorm:"column:sent_at"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (EmailLog) TableName() string { return "email_logs" }

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

// AIUsageAuthContext records RBAC context for AI usage audit.
type AIUsageAuthContext struct {
	ID            uint64    `gorm:"primaryKey"`
	UsageLogID    uint64    `gorm:"column:usage_log_id;not null;index:idx_audit_context_usage_log"`
	ActorUserID   uint64    `gorm:"column:actor_user_id;not null;index:idx_audit_context_actor,priority:1"`
	AccountType   string    `gorm:"column:account_type;size:32"`
	RoleKeys      string    `gorm:"column:role_keys;size:512"`
	PermissionKey string    `gorm:"column:permission_key;size:128;index:idx_audit_context_permission,priority:1"`
	ScopeKeys     string    `gorm:"column:scope_keys;size:512"`
	ResourceType  string    `gorm:"column:resource_type;size:64"`
	ResourceID    uint64    `gorm:"column:resource_id"`
	Decision      string    `gorm:"column:decision;size:32"`
	RequestID     string    `gorm:"column:request_id;size:64;index:idx_audit_context_request"`
	CreatedAt     time.Time `gorm:"column:created_at"`
}

func (AIUsageAuthContext) TableName() string { return "ai_usage_auth_contexts" }

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
	Mode            string     `gorm:"column:mode;size:32"` // video / phone / onsite
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
	ID                  int64     `gorm:"primaryKey"`
	InterviewID         int64     `gorm:"column:interview_id;uniqueIndex:uk_interview_feedback_once,priority:1"`
	ApplicationID       int64     `gorm:"column:application_id;not null;uniqueIndex:uk_interview_feedback_once,priority:2"`
	InterviewerID       int64     `gorm:"column:interviewer_id;not null;uniqueIndex:uk_interview_feedback_once,priority:3"`
	Recommendation      string    `gorm:"column:recommendation;size:32"` // positive / negative / pending
	Score               int32     `gorm:"column:score"`
	DimensionScoresJSON string    `gorm:"column:dimension_scores_json;type:text"`
	Comments            string    `gorm:"column:comments;type:text"`
	SubmittedAt         time.Time `gorm:"column:submitted_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at"`
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

// ── LLM Provider & Model Config (Phase M2) ─────────────────────────────

// LlmProvider represents an LLM service provider configuration.
type LlmProvider struct {
	ID              int64     `gorm:"primaryKey"`
	Name            string    `gorm:"column:name;size:128;not null"`
	BaseURL         string    `gorm:"column:base_url;size:512;not null"`
	APIKeyEncrypted string    `gorm:"column:api_key_encrypted;size:512;not null"`
	ProviderType    string    `gorm:"column:provider_type;size:64;not null"`
	ExtraHeaders    *string   `gorm:"column:extra_headers;type:json"`
	IsEnabled       int32     `gorm:"column:is_enabled;default:1"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at"`
}

func (LlmProvider) TableName() string { return "llm_providers" }

// LlmModel represents an LLM model configuration under a provider.
type LlmModel struct {
	ID                  int64     `gorm:"primaryKey"`
	ProviderID          int64     `gorm:"column:provider_id;not null"`
	ModelName           string    `gorm:"column:model_name;size:128;not null"`
	DisplayName         string    `gorm:"column:display_name;size:128"`
	Temperature         float64   `gorm:"column:temperature;default:0.7"`
	TopP                float64   `gorm:"column:top_p;default:1.0"`
	MaxTokens           int32     `gorm:"column:max_tokens;default:4096"`
	ContextWindowTokens int32     `gorm:"column:context_window_tokens;default:0"`
	MaxConcurrency      int32     `gorm:"column:max_concurrency;default:10"`
	TimeoutSeconds      int32     `gorm:"column:timeout_seconds;default:90"`
	IsEnabled           int32     `gorm:"column:is_enabled;default:1"`
	IsDefault           int32     `gorm:"column:is_default;default:0"`
	CreatedAt           time.Time `gorm:"column:created_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at"`
}

func (LlmModel) TableName() string { return "llm_models" }

// ── Prompt Templates (Phase M2) ────────────────────────────────────────

// PromptTemplate represents a prompt template definition.
type PromptTemplate struct {
	ID         int64     `gorm:"primaryKey"`
	Name       string    `gorm:"column:name;size:256;not null"`
	Content    string    `gorm:"column:content;type:text;not null"`
	Variables  *string   `gorm:"column:variables;type:json"`
	Version    int32     `gorm:"column:version;default:1"`
	IsActive   int32     `gorm:"column:is_active;default:1"`
	AgentType  string    `gorm:"column:agent_type;size:64;not null"`
	PromptRole string    `gorm:"column:prompt_role;size:32;default:system"`
	CreatedBy  *int64    `gorm:"column:created_by"`
	UpdatedBy  *int64    `gorm:"column:updated_by"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

func (PromptTemplate) TableName() string { return "prompt_templates" }

// PromptVersion represents a version snapshot of a prompt template.
type PromptVersion struct {
	ID         int64     `gorm:"primaryKey"`
	TemplateID int64     `gorm:"column:template_id;not null"`
	Version    int32     `gorm:"column:version;not null"`
	Content    string    `gorm:"column:content;type:text;not null"`
	ChangedBy  *int64    `gorm:"column:changed_by"`
	ChangeNote string    `gorm:"column:change_note;size:512"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

func (PromptVersion) TableName() string { return "prompt_versions" }

// ── Agent Configuration (P1-004) ──────────────────────────────────────

// AgentConfig represents an agent configuration entry.
type AgentConfig struct {
	ID                  int64     `gorm:"primaryKey"`
	Name                string    `gorm:"column:name;size:128;not null;uniqueIndex:uk_name"`
	DisplayName         string    `gorm:"column:display_name;size:256;not null"`
	Description         string    `gorm:"column:description;type:text"`
	AgentType           string    `gorm:"column:agent_type;size:64;not null"`
	PromptTemplateID    *int64    `gorm:"column:prompt_template_id"`
	Instruction         string    `gorm:"column:instruction;type:text"`
	MaxIterations       int32     `gorm:"column:max_iterations;default:5"`
	TemperatureOverride *float64  `gorm:"column:temperature_override"`
	IsDefault           int32     `gorm:"column:is_default;default:0"`
	IsEnabled           int32     `gorm:"column:is_enabled;default:1"`
	CreatedAt           time.Time `gorm:"column:created_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at"`
}

func (AgentConfig) TableName() string { return "agent_configs" }

// AgentToolBinding represents a tool binding for an agent.
type AgentToolBinding struct {
	ID        int64     `gorm:"primaryKey"`
	AgentID   int64     `gorm:"column:agent_id;not null;uniqueIndex:uk_agent_tool,priority:1"`
	ToolName  string    `gorm:"column:tool_name;size:128;not null;uniqueIndex:uk_agent_tool,priority:2"`
	IsEnabled int32     `gorm:"column:is_enabled;default:1"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (AgentToolBinding) TableName() string { return "agent_tool_bindings" }

// AgentCapabilityBinding represents a unified capability binding for an agent.
type AgentCapabilityBinding struct {
	ID               int64     `gorm:"primaryKey"`
	AgentID          int64     `gorm:"column:agent_id;not null;uniqueIndex:uk_agent_capability,priority:1"`
	CapabilitySource string    `gorm:"column:capability_source;size:32;not null;uniqueIndex:uk_agent_capability,priority:2"`
	CapabilityKey    string    `gorm:"column:capability_key;size:256;not null;uniqueIndex:uk_agent_capability,priority:3"`
	IsEnabled        int32     `gorm:"column:is_enabled;default:1"`
	Priority         int32     `gorm:"column:priority;default:0"`
	PolicyJSON       *string   `gorm:"column:policy_json;type:json"`
	CreatedAt        time.Time `gorm:"column:created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at"`
}

func (AgentCapabilityBinding) TableName() string { return "agent_capability_bindings" }

// ── SKILL Registry ───────────────────────────────────────────────────

type Skill struct {
	ID               int64     `gorm:"primaryKey"`
	Name             string    `gorm:"column:name;size:128;not null;uniqueIndex:uk_ai_skills_name"`
	DisplayName      string    `gorm:"column:display_name;size:256;not null"`
	Description      string    `gorm:"column:description;type:text"`
	SourceType       string    `gorm:"column:source_type;size:32;not null;default:local"`
	SourceURI        string    `gorm:"column:source_uri;type:text"`
	CurrentVersionID *int64    `gorm:"column:current_version_id"`
	IsEnabled        int32     `gorm:"column:is_enabled;default:1"`
	CreatedAt        time.Time `gorm:"column:created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at"`
}

func (Skill) TableName() string { return "ai_skills" }

type SkillVersion struct {
	ID               int64     `gorm:"primaryKey"`
	SkillID          int64     `gorm:"column:skill_id;not null;uniqueIndex:uk_ai_skill_versions_skill_version,priority:1"`
	Version          string    `gorm:"column:version;size:64;not null;uniqueIndex:uk_ai_skill_versions_skill_version,priority:2"`
	ManifestJSON     string    `gorm:"column:manifest_json;type:json;not null"`
	Instruction      string    `gorm:"column:instruction;type:text"`
	InputSchemaJSON  *string   `gorm:"column:input_schema_json;type:json"`
	OutputSchemaJSON *string   `gorm:"column:output_schema_json;type:json"`
	RuntimeType      string    `gorm:"column:runtime_type;size:32;not null;default:prompt"`
	CreatedAt        time.Time `gorm:"column:created_at"`
}

func (SkillVersion) TableName() string { return "ai_skill_versions" }

type SkillTool struct {
	ID                int64     `gorm:"primaryKey"`
	SkillVersionID    int64     `gorm:"column:skill_version_id;not null;uniqueIndex:uk_ai_skill_tools_version_tool,priority:1"`
	ToolName          string    `gorm:"column:tool_name;size:128;not null;uniqueIndex:uk_ai_skill_tools_version_tool,priority:2"`
	Description       string    `gorm:"column:description;type:text"`
	InputSchemaJSON   *string   `gorm:"column:input_schema_json;type:json"`
	RuntimeConfigJSON *string   `gorm:"column:runtime_config_json;type:json"`
	IsEnabled         int32     `gorm:"column:is_enabled;default:1"`
	CreatedAt         time.Time `gorm:"column:created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at"`
}

func (SkillTool) TableName() string { return "ai_skill_tools" }

type AgentSkill struct {
	ID                   int64     `gorm:"primaryKey"`
	Name                 string    `gorm:"column:name;size:128;not null;uniqueIndex:uk_agent_skills_name"`
	DisplayName          string    `gorm:"column:display_name;size:128;not null"`
	Description          string    `gorm:"column:description;type:text"`
	CurrentVersionID     *int64    `gorm:"column:current_version_id"`
	IsEnabled            int32     `gorm:"column:is_enabled;default:1"`
	IsManualInvocable    int32     `gorm:"column:is_manual_invocable;default:1"`
	TriggerKeywords      string    `gorm:"column:trigger_keywords;type:json"`
	AgentType            string    `gorm:"column:agent_type;size:64;not null;default:'hr_recruiting_agent'"`
	Category             string    `gorm:"column:category;size:64;not null;default:'general'"`
	Scenario             string    `gorm:"column:scenario;size:128;not null;default:''"`
	Priority             int32     `gorm:"column:priority;not null;default:0"`
	RiskLevel            string    `gorm:"column:risk_level;size:32;not null;default:'medium'"`
	RequiredCapabilities string    `gorm:"column:required_capabilities;type:json"`
	OutputSchema         string    `gorm:"column:output_schema;type:json"`
	EvaluationCriteria   string    `gorm:"column:evaluation_criteria;type:json"`
	SemanticTags         string    `gorm:"column:semantic_tags;type:json"`
	CreatedBy            *int64    `gorm:"column:created_by"`
	UpdatedBy            *int64    `gorm:"column:updated_by"`
	CreatedAt            time.Time `gorm:"column:created_at"`
	UpdatedAt            time.Time `gorm:"column:updated_at"`
}

func (AgentSkill) TableName() string { return "agent_skills" }

type AgentSkillVersion struct {
	ID              int64     `gorm:"primaryKey"`
	SkillID         int64     `gorm:"column:skill_id;not null;uniqueIndex:uk_agent_skill_versions_skill_version,priority:1"`
	Version         string    `gorm:"column:version;size:64;not null;uniqueIndex:uk_agent_skill_versions_skill_version,priority:2"`
	FlowJSON        string    `gorm:"column:flow_json;type:json"`
	SkillMD         string    `gorm:"column:skill_md;type:mediumtext;not null"`
	FrontmatterJSON string    `gorm:"column:frontmatter_json;type:json"`
	BodyMarkdown    string    `gorm:"column:body_markdown;type:mediumtext"`
	ChangeNote      string    `gorm:"column:change_note;type:text"`
	CreatedBy       *int64    `gorm:"column:created_by"`
	CreatedAt       time.Time `gorm:"column:created_at"`
}

func (AgentSkillVersion) TableName() string { return "agent_skill_versions" }

// ── MCP Server (P1-006) ─────────────────────────────────────────────

// MCPServer represents a registered MCP server.
type MCPServer struct {
	ID             int64     `gorm:"primaryKey"`
	Name           string    `gorm:"column:name;size:128;not null"`
	Description    *string   `gorm:"column:description;type:text"`
	Transport      string    `gorm:"column:transport;size:16;not null"`
	CommandOrURL   string    `gorm:"column:command_or_url;type:text;not null"`
	Args           *string   `gorm:"column:args;type:json"`
	EnvVars        *string   `gorm:"column:env_vars;type:json"`
	TimeoutSeconds int32     `gorm:"column:timeout_seconds;default:30"`
	IsEnabled      int32     `gorm:"column:is_enabled;default:0"`
	Status         string    `gorm:"column:status;size:32;default:disconnected"`
	ToolCount      int32     `gorm:"column:tool_count;default:0"`
	LastError      *string   `gorm:"column:last_error;size:512"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

func (MCPServer) TableName() string { return "mcp_servers" }

// MCPToolLog records a single MCP tool call for audit purposes.
type MCPToolLog struct {
	ID                 int64     `gorm:"primaryKey"`
	ServerID           int64     `gorm:"column:server_id;not null;index:idx_mcp_tool_logs_server_tool_created,priority:1"`
	ToolName           string    `gorm:"column:tool_name;size:128;not null;index:idx_mcp_tool_logs_server_tool_created,priority:2"`
	ArgsJSON           *string   `gorm:"column:args_json;type:json"`
	ResultContent      *string   `gorm:"column:result_content;type:text"`
	DurationMs         int32     `gorm:"column:duration_ms;default:0"`
	ErrorMsg           *string   `gorm:"column:error_msg;size:512"`
	CalledByHRID       *int64    `gorm:"column:called_by_hr_id"`
	SessionID          *int64    `gorm:"column:session_id"`
	PolicyID           *int64    `gorm:"column:policy_id"`
	PolicyDecision     string    `gorm:"column:policy_decision;size:32;default:allow"`
	PolicyReason       *string   `gorm:"column:policy_reason;size:512"`
	PolicySnapshotJSON *string   `gorm:"column:policy_snapshot_json;type:json"`
	CreatedAt          time.Time `gorm:"column:created_at;index:idx_mcp_tool_logs_server_tool_created,priority:3"`
}

func (MCPToolLog) TableName() string { return "mcp_tool_logs" }

// MCPToolPolicy stores governable execution rules for one MCP server/tool pair.
type MCPToolPolicy struct {
	ID                     int64     `gorm:"primaryKey"`
	ServerID               int64     `gorm:"column:server_id;not null;uniqueIndex:uk_mcp_tool_policy_server_tool,priority:1"`
	ToolName               string    `gorm:"column:tool_name;size:128;not null;uniqueIndex:uk_mcp_tool_policy_server_tool,priority:2"`
	Effect                 string    `gorm:"column:effect;size:32;not null;default:allow"`
	RiskLevel              string    `gorm:"column:risk_level;size:32;default:medium"`
	RequireConfirmation    int32     `gorm:"column:require_confirmation;default:0"`
	AllowedRolesJSON       *string   `gorm:"column:allowed_roles_json;type:json"`
	AllowedScopesJSON      *string   `gorm:"column:allowed_scopes_json;type:json"`
	RequiredArgsJSON       *string   `gorm:"column:required_args_json;type:json"`
	DeniedArgsJSON         *string   `gorm:"column:denied_args_json;type:json"`
	ArgRulesJSON           *string   `gorm:"column:arg_rules_json;type:json"`
	RedactFieldsJSON       *string   `gorm:"column:redact_fields_json;type:json"`
	RateLimitWindowSeconds int32     `gorm:"column:rate_limit_window_seconds;default:0"`
	RateLimitMaxCalls      int32     `gorm:"column:rate_limit_max_calls;default:0"`
	IsEnabled              int32     `gorm:"column:is_enabled;default:1"`
	CreatedByHRID          *int64    `gorm:"column:created_by_hr_id"`
	UpdatedByHRID          *int64    `gorm:"column:updated_by_hr_id"`
	CreatedAt              time.Time `gorm:"column:created_at"`
	UpdatedAt              time.Time `gorm:"column:updated_at"`
}

func (MCPToolPolicy) TableName() string { return "mcp_tool_policies" }
