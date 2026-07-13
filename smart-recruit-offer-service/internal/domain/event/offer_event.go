package event

import "time"

type Type string

const (
	TypeCreated   Type = "created"
	TypeUpdated   Type = "updated"
	TypeSent      Type = "sent"
	TypeWithdrawn Type = "withdrawn"
	TypeAccepted  Type = "accepted"
	TypeRejected  Type = "rejected"
)

type OfferEvent struct {
	ID               uint64
	OfferID          int64
	EventType        Type
	ActorUserID      int64
	ActorAccountType string
	Reason           string
	MetadataJSON     string
	CreatedAt        time.Time
}

func NewOfferEvent(offerID int64, eventType Type, actorUserID int64, actorAccountType string, reason string, createdAt time.Time) OfferEvent {
	return OfferEvent{
		OfferID:          offerID,
		EventType:        eventType,
		ActorUserID:      actorUserID,
		ActorAccountType: actorAccountType,
		Reason:           reason,
		CreatedAt:        createdAt,
	}
}
