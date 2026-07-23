package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	ErrOfferNotDraft        = errors.New("仅可编辑草稿状态的 Offer")
	ErrOfferNotSendable     = errors.New("仅可发送草稿状态的 Offer")
	ErrOfferNotWithdrawable = errors.New("该状态下的 Offer 无法撤回")
	ErrOfferNotDecidable    = errors.New("仅可操作已发送状态的 Offer")
	ErrOfferExpired         = errors.New("Offer 已过期，无法接受")
	ErrCandidateMismatch    = errors.New("候选人与 Offer 不匹配")
)

type Offer struct {
	ID               int64
	TenantID         int64
	ApplicationID    int64
	CandidateUserID  int64
	JobID            int64
	Status           OfferStatus
	Title            string
	SalaryRange      string
	Level            string
	WorkLocation     string
	StartDate        string
	ExpiresAt        *time.Time
	TermsJSON        string
	SentSnapshotJSON string
	CreatedBy        int64
	SentBy           *int64
	DecidedAt        *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type DraftDetails struct {
	ApplicationID   int64
	CandidateUserID int64
	JobID           int64
	Title           string
	SalaryRange     string
	Level           string
	WorkLocation    string
	StartDate       string
	ExpiresAt       *time.Time
	TermsJSON       string
	CreatedBy       int64
}

type DraftPatch struct {
	Title        string
	SalaryRange  string
	Level        string
	WorkLocation string
	StartDate    string
	ExpiresAt    *time.Time
	TermsJSON    string
}

type SentSnapshot struct {
	Title        string `json:"title"`
	SalaryRange  string `json:"salary_range"`
	Level        string `json:"level"`
	WorkLocation string `json:"work_location"`
	StartDate    string `json:"start_date"`
	TermsJSON    string `json:"terms_json"`
}

func NewDraft(details DraftDetails) (*Offer, error) {
	if details.ApplicationID == 0 {
		return nil, errors.New("application id is required")
	}
	if details.CandidateUserID == 0 {
		return nil, errors.New("candidate user id is required")
	}
	if details.JobID == 0 {
		return nil, errors.New("job id is required")
	}
	if details.CreatedBy == 0 {
		return nil, errors.New("created by is required")
	}
	return &Offer{
		ApplicationID:   details.ApplicationID,
		CandidateUserID: details.CandidateUserID,
		JobID:           details.JobID,
		Status:          OfferStatusDraft,
		Title:           details.Title,
		SalaryRange:     details.SalaryRange,
		Level:           details.Level,
		WorkLocation:    details.WorkLocation,
		StartDate:       details.StartDate,
		ExpiresAt:       details.ExpiresAt,
		TermsJSON:       details.TermsJSON,
		CreatedBy:       details.CreatedBy,
	}, nil
}

func (o *Offer) ApplyDraftPatch(patch DraftPatch) error {
	if o.Status != OfferStatusDraft {
		return ErrOfferNotDraft
	}
	if patch.Title != "" {
		o.Title = patch.Title
	}
	if patch.SalaryRange != "" {
		o.SalaryRange = patch.SalaryRange
	}
	if patch.Level != "" {
		o.Level = patch.Level
	}
	if patch.WorkLocation != "" {
		o.WorkLocation = patch.WorkLocation
	}
	if patch.StartDate != "" {
		o.StartDate = patch.StartDate
	}
	if patch.ExpiresAt != nil {
		o.ExpiresAt = patch.ExpiresAt
	}
	if patch.TermsJSON != "" {
		o.TermsJSON = patch.TermsJSON
	}
	return nil
}

func (o *Offer) MarkSent(actorID int64) error {
	if o.Status != OfferStatusDraft {
		return ErrOfferNotSendable
	}
	if actorID == 0 {
		return errors.New("sent actor is required")
	}
	snapshot, err := json.Marshal(SentSnapshot{
		Title:        o.Title,
		SalaryRange:  o.SalaryRange,
		Level:        o.Level,
		WorkLocation: o.WorkLocation,
		StartDate:    o.StartDate,
		TermsJSON:    o.TermsJSON,
	})
	if err != nil {
		return fmt.Errorf("marshal offer sent snapshot: %w", err)
	}
	o.Status = OfferStatusSent
	o.SentSnapshotJSON = string(snapshot)
	o.SentBy = &actorID
	return nil
}

func (o *Offer) MarkWithdrawn() (bool, error) {
	switch o.Status {
	case OfferStatusDraft:
		o.Status = OfferStatusWithdrawn
		return false, nil
	case OfferStatusSent:
		o.Status = OfferStatusWithdrawn
		return true, nil
	default:
		return false, ErrOfferNotWithdrawable
	}
}

func (o *Offer) MarkAccepted(candidateUserID int64, now time.Time) error {
	if o.CandidateUserID != candidateUserID {
		return ErrCandidateMismatch
	}
	if o.Status != OfferStatusSent {
		return ErrOfferNotDecidable
	}
	if o.ExpiresAt != nil && now.After(*o.ExpiresAt) {
		return ErrOfferExpired
	}
	o.Status = OfferStatusAccepted
	o.DecidedAt = &now
	return nil
}

func (o *Offer) MarkRejected(candidateUserID int64, now time.Time) error {
	if o.CandidateUserID != candidateUserID {
		return ErrCandidateMismatch
	}
	if o.Status != OfferStatusSent {
		return ErrOfferNotDecidable
	}
	o.Status = OfferStatusRejected
	o.DecidedAt = &now
	return nil
}
