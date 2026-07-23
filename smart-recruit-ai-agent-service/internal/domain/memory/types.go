package memory

import (
	"errors"
	"time"
)

type OwnerRole int32

const (
	OwnerRoleCandidate OwnerRole = 1
	OwnerRoleHR        OwnerRole = 2
)

type Status string

const (
	StatusActive   Status = "active"
	StatusArchived Status = "archived"
	StatusRevoked  Status = "revoked"
)

type PIILevel string

const (
	PIILevelNone PIILevel = "none"
	PIILevelLow  PIILevel = "low"
	PIILevelHigh PIILevel = "high"
)

type Scope struct {
	Type string
	ID   uint64
}

type OwnerKey struct {
	TenantID *uint64
	Role     OwnerRole
	ID       uint64
}

func (o OwnerKey) Validate() error {
	if o.ID == 0 {
		return errors.New("memory owner id is required")
	}
	switch o.Role {
	case OwnerRoleHR:
		if o.TenantID == nil || *o.TenantID == 0 {
			return errors.New("HR memory tenant id is required")
		}
	case OwnerRoleCandidate:
		if o.TenantID != nil {
			return errors.New("candidate memory must not have a tenant id")
		}
	default:
		return errors.New("invalid memory owner role")
	}
	return nil
}

func (m Memory) OwnerKey() OwnerKey {
	return OwnerKey{TenantID: m.TenantID, Role: m.OwnerRole, ID: m.OwnerID}
}

type Memory struct {
	ID              uint64
	TenantID        *uint64
	OwnerRole       OwnerRole
	OwnerID         uint64
	HRID            uint64
	Scope           Scope
	MemoryType      string
	Content         string
	Source          string
	Confidence      float64
	Importance      float64
	Status          Status
	PIILevel        PIILevel
	ContentHash     string
	SourceSessionID *uint64
	SourceMessageID *uint64
	SourceRunID     *uint64
	CreatedBy       *uint64
	RevokedBy       *uint64
	RevokeReason    string
	ExpiresAt       *time.Time
	DeletedAt       *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
	AllowHighPII    bool
}

func CompatibilityHRID(ownerRole OwnerRole, ownerID uint64) uint64 {
	if ownerRole == OwnerRoleHR {
		return ownerID
	}
	return 0
}
