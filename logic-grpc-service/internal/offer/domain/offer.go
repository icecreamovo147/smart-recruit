package domain

// OfferID identifies an Offer-owned lifecycle aggregate.
type OfferID int64

// ApplicationID identifies the recruitment application linked by an offer.
type ApplicationID int64

// CandidateUserID identifies the candidate user that receives an offer.
type CandidateUserID int64

// JobID identifies the recruitment job linked by an offer.
type JobID int64

// OfferStatus is the current persisted offer lifecycle state.
type OfferStatus string

const (
	OfferStatusDraft     OfferStatus = "draft"
	OfferStatusSent      OfferStatus = "sent"
	OfferStatusWithdrawn OfferStatus = "withdrawn"
	OfferStatusAccepted  OfferStatus = "accepted"
	OfferStatusRejected  OfferStatus = "rejected"
)

// OfferEventType is the persisted audit event type for offer lifecycle changes.
type OfferEventType string

const (
	OfferEventCreated   OfferEventType = "created"
	OfferEventUpdated   OfferEventType = "updated"
	OfferEventSent      OfferEventType = "sent"
	OfferEventWithdrawn OfferEventType = "withdrawn"
	OfferEventAccepted  OfferEventType = "accepted"
	OfferEventRejected  OfferEventType = "rejected"
)
