package model

import "time"

const (
	JobStatusOffline int32 = 0
	JobStatusOnline  int32 = 1

	CandidateLegacyRole int32 = 1

	MaxResumeSizeBytes = 20 * 1024 * 1024
)

type Job struct {
	ID           int64
	HRID         int64
	Title        string
	Department   string
	DepartmentID *int64
	Location     string
	LocationID   *int64
	SalaryRange  string
	Description  string
	Requirements string
	Status       int32
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Department struct {
	ID       int64
	FullName string
}

type JobLocation struct {
	ID   int64
	Name string
}

type CandidateProfile struct {
	ID             int64
	UserID         int64
	RealName       string
	Phone          string
	Education      string
	School         string
	WorkExperience string
	Skills         string
	IsComplete     int32
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Resume struct {
	ID         int64
	UserID     int64
	OSSKey     string
	FileName   string
	FileType   string
	FileSize   int64
	ParsedText string
	ParsedAt   *time.Time
	IsValid    int32
	UploadedAt time.Time
}

type PresignSession struct {
	UserID      int64
	OSSKey      string
	FileName    string
	FileType    string
	ContentType string
	MaxSize     int64
	Status      string
}

type UsageLogEntry struct {
	UserID      int64
	Role        int32
	ServiceType string
	Endpoint    string
	Provider    string
	ObjectKey   string
	ObjectSize  int64
	Status      string
}

type OutboxEvent struct {
	Topic         string
	AggregateType string
	AggregateID   uint64
	EventType     string
	Payload       any
}

type ResumeParsePayload struct {
	ResumeID int64  `json:"resume_id"`
	FileType string `json:"file_type"`
	OSSKey   string `json:"oss_key"`
}

const (
	StatusKeyApplied            = "applied"
	StatusKeyViewed             = "viewed"
	StatusKeyScreening          = "screening"
	StatusKeyScreenPassed       = "screen_passed"
	StatusKeyInterviewPending   = "interview_pending"
	StatusKeyInterviewing       = "interviewing"
	StatusKeyInterviewPassed    = "interview_passed"
	StatusKeyInterviewCancelled = "interview_cancelled"
	StatusKeyOfferPending       = "offer_pending"
	StatusKeyOfferSent          = "offer_sent"
	StatusKeyOfferAccepted      = "offer_accepted"
	StatusKeyOfferRejected      = "offer_rejected"
	StatusKeyHired              = "hired"
	StatusKeyRejected           = "rejected"
	StatusKeyWithdrawn          = "withdrawn"

	PermissionApplicationRead         = "application.read"
	PermissionCollaborationNoteCreate = "collaboration.note.create"
	PermissionCollaborationNoteRead   = "collaboration.note.read"
	PermissionCollaborationTagManage  = "collaboration.tag.manage"
	PermissionCollaborationTaskManage = "collaboration.task.manage"
	PermissionAuditUsageRead          = "audit.usage.read"
)

var LegacyStatusToKey = map[int32]string{
	0: StatusKeyApplied,
	1: StatusKeyViewed,
	2: StatusKeyScreenPassed,
	3: StatusKeyRejected,
}

var StatusKeyToLegacy = map[string]int32{
	StatusKeyApplied:            0,
	StatusKeyViewed:             1,
	StatusKeyScreening:          1,
	StatusKeyScreenPassed:       2,
	StatusKeyInterviewPending:   2,
	StatusKeyInterviewing:       2,
	StatusKeyInterviewPassed:    2,
	StatusKeyInterviewCancelled: 2,
	StatusKeyOfferPending:       2,
	StatusKeyOfferSent:          2,
	StatusKeyOfferAccepted:      2,
	StatusKeyHired:              2,
	StatusKeyOfferRejected:      3,
	StatusKeyRejected:           3,
	StatusKeyWithdrawn:          3,
}

var HRStatusLabels = map[string]string{
	StatusKeyApplied:            "待查看",
	StatusKeyViewed:             "已查看",
	StatusKeyScreening:          "筛选中",
	StatusKeyScreenPassed:       "筛选通过",
	StatusKeyInterviewPending:   "待安排面试",
	StatusKeyInterviewing:       "面试中",
	StatusKeyInterviewPassed:    "面试通过",
	StatusKeyInterviewCancelled: "面试已取消",
	StatusKeyOfferPending:       "待发Offer",
	StatusKeyOfferSent:          "Offer已发",
	StatusKeyOfferAccepted:      "Offer已接受",
	StatusKeyOfferRejected:      "Offer被拒",
	StatusKeyHired:              "已入职",
	StatusKeyRejected:           "淘汰",
	StatusKeyWithdrawn:          "候选人撤回",
}

var TerminalStatusKeys = map[string]bool{
	StatusKeyRejected:      true,
	StatusKeyWithdrawn:     true,
	StatusKeyOfferRejected: true,
	StatusKeyHired:         true,
}

type Application struct {
	ID        int64
	UserID    int64
	JobID     int64
	ResumeID  int64
	Status    int32
	StatusKey string
	RoundNo   int32
	IsCurrent int32
	AppliedAt time.Time
}

type ApplicationDetail struct {
	ApplicationID int64
	UserID        int64
	JobID         int64
	JobTitle      string
	RealName      string
	ResumeID      int64
	Status        int32
	StatusKey     string
	RoundNo       int32
	IsCurrent     int32
}

type ApplicationStatusTransition struct {
	ID               uint64
	ApplicationID    int64
	FromStatus       string
	ToStatus         string
	ActorUserID      int64
	ActorAccountType string
	Reason           string
	CreatedAt        time.Time
}

type NotificationPayload struct {
	ReceiverID          int64
	ReceiverAccountType string
	Type                string
	Title               string
	Content             string
	Link                string
	BizType             string
	BizID               int64
	JobTitle            string
	RecipientName       string
}

type CandidateNote struct {
	ID              uint64
	CandidateUserID uint64
	ApplicationID   *uint64
	AuthorUserID    uint64
	Content         string
	Visibility      string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type CandidateTag struct {
	ID        uint64
	Name      string
	Color     string
	CreatedBy *uint64
}

type CandidateTagAssignment struct {
	TagID           uint64
	CandidateUserID uint64
	CreatedBy       *uint64
}

type FollowUpTask struct {
	ID              uint64
	CandidateUserID uint64
	ApplicationID   *uint64
	AssigneeUserID  uint64
	CreatedBy       uint64
	Title           string
	Description     string
	DueAt           *time.Time
	Status          string
	CompletedAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type DepartmentNode struct {
	ID               int64
	ParentID         int64
	Name             string
	FullName         string
	Path             string
	Depth            int
	SortOrder        int
	IsActive         int32
	InheritLocations int32
	CreatedBy        *int64
}

type UsageStatsRow struct {
	Name          string
	TotalTokens   int64
	CallCount     int64
	AvgCostMs     float64
	EstimatedCost float64
}

type UsageTrendRow struct {
	Date          string
	TotalTokens   int64
	CallCount     int64
	AvgCostMs     float64
	EstimatedCost float64
}
