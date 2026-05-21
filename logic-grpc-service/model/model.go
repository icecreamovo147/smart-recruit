package model

import "time"

type User struct {
	ID        int64 `gorm:"primaryKey"`
	Username  string
	Password  string
	Role      int32
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

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
	ID        int64 `gorm:"primaryKey"`
	JobID     int64
	UserID    int64
	ResumeID  int64
	Status    int32
	RoundNo   int32
	IsCurrent int32
	AppliedAt time.Time
	UpdatedAt time.Time
}

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
	ID           int64      `gorm:"primaryKey"`
	EventID      *string    `gorm:"column:event_id;uniqueIndex:uk_notification_event_id"`
	ReceiverID   int64      `gorm:"column:receiver_id;uniqueIndex:uk_notification_once,priority:1"`
	ReceiverRole int32      `gorm:"column:receiver_role;uniqueIndex:uk_notification_once,priority:2"`
	Type         string     `gorm:"column:type;uniqueIndex:uk_notification_once,priority:5"`
	Title        string     `gorm:"column:title"`
	Content      string     `gorm:"column:content"`
	Link         string     `gorm:"column:link"`
	BizType      string     `gorm:"column:biz_type;uniqueIndex:uk_notification_once,priority:3"`
	BizID        int64      `gorm:"column:biz_id;uniqueIndex:uk_notification_once,priority:4"`
	IsRead       int32      `gorm:"column:is_read"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	ReadAt       *time.Time `gorm:"column:read_at"`
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
