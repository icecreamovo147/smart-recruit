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
