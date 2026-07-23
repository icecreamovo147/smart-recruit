package model

import (
	"errors"
	"time"
)

var (
	ErrInterviewNotFound         = errors.New("面试记录不存在")
	ErrInterviewAlreadyCancelled = errors.New("该面试已取消")
)

type Interview struct {
	ID              int64
	TenantID        int64
	ApplicationID   int64
	InterviewerID   int64
	RoundNo         int32
	Title           string
	Mode            string
	MeetingURL      string
	Location        string
	DurationMinutes int32
	CandidateNote   string
	InternalNote    string
	CancelReason    string
	ScheduledAt     *time.Time
	Status          InterviewStatus
	CreatedBy       *int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ScheduleDetails struct {
	ApplicationID   int64
	InterviewerID   int64
	RoundNo         int32
	Title           string
	Mode            string
	MeetingURL      string
	Location        string
	DurationMinutes int32
	CandidateNote   string
	InternalNote    string
	ScheduledAt     *time.Time
	CreatedBy       int64
}

type InterviewPatch struct {
	Title           string
	Mode            string
	MeetingURL      string
	Location        string
	DurationMinutes int32
	CandidateNote   string
	InternalNote    string
	ScheduledAt     *time.Time
}

func NewScheduledInterview(details ScheduleDetails) (*Interview, error) {
	if details.ApplicationID == 0 {
		return nil, errors.New("application id is required")
	}
	if details.InterviewerID == 0 {
		return nil, errors.New("interviewer id is required")
	}
	if details.CreatedBy == 0 {
		return nil, errors.New("created by is required")
	}
	roundNo := details.RoundNo
	if roundNo <= 0 {
		roundNo = 1
	}
	title := details.Title
	if title == "" {
		title = "第 " + itoa32(roundNo) + " 轮面试"
	}
	mode := details.Mode
	if mode == "" {
		mode = "video"
	}
	createdBy := details.CreatedBy
	return &Interview{
		ApplicationID:   details.ApplicationID,
		InterviewerID:   details.InterviewerID,
		RoundNo:         roundNo,
		Title:           title,
		Mode:            mode,
		MeetingURL:      details.MeetingURL,
		Location:        details.Location,
		DurationMinutes: details.DurationMinutes,
		CandidateNote:   details.CandidateNote,
		InternalNote:    details.InternalNote,
		ScheduledAt:     details.ScheduledAt,
		Status:          InterviewStatusScheduled,
		CreatedBy:       &createdBy,
	}, nil
}

func (i *Interview) ApplyPatch(patch InterviewPatch) {
	if patch.Title != "" {
		i.Title = patch.Title
	}
	if patch.Mode != "" {
		i.Mode = patch.Mode
	}
	if patch.MeetingURL != "" {
		i.MeetingURL = patch.MeetingURL
	}
	if patch.Location != "" {
		i.Location = patch.Location
	}
	if patch.DurationMinutes > 0 {
		i.DurationMinutes = patch.DurationMinutes
	}
	if patch.CandidateNote != "" {
		i.CandidateNote = patch.CandidateNote
	}
	if patch.InternalNote != "" {
		i.InternalNote = patch.InternalNote
	}
	if patch.ScheduledAt != nil {
		i.ScheduledAt = patch.ScheduledAt
	}
}

func (i *Interview) Cancel(reason string) error {
	if i.Status == InterviewStatusCancelled {
		return ErrInterviewAlreadyCancelled
	}
	i.Status = InterviewStatusCancelled
	i.CancelReason = reason
	return nil
}

func (i *Interview) CompleteIfScheduled() bool {
	if i.Status != InterviewStatusScheduled {
		return false
	}
	i.Status = InterviewStatusCompleted
	return true
}

func (i *Interview) IsActive() bool {
	return i.Status == InterviewStatusPending || i.Status == InterviewStatusScheduled
}

func itoa32(v int32) string {
	if v == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	n := v
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
