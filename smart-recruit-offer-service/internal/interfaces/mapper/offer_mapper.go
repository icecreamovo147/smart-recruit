package mapper

import (
	"time"

	"smart-recruit-offer-service/internal/domain/event"
	"smart-recruit-offer-service/internal/domain/repository"
	"smart-recruit-proto/recruitment/pb"
)

func ToPBOffer(details repository.OfferDetails) *pb.Offer {
	var expiresAt string
	if details.ExpiresAt != nil {
		expiresAt = details.ExpiresAt.Format(time.RFC3339)
	}
	var decidedAt string
	if details.DecidedAt != nil {
		decidedAt = details.DecidedAt.Format(time.RFC3339)
	}
	var sentBy int64
	if details.SentBy != nil {
		sentBy = *details.SentBy
	}
	return &pb.Offer{
		Id:                   details.ID,
		ApplicationId:        details.ApplicationID,
		CandidateUserId:      details.CandidateUserID,
		JobId:                details.JobID,
		Status:               string(details.Status),
		Title:                details.Title,
		SalaryRange:          details.SalaryRange,
		Level:                details.Level,
		WorkLocation:         details.WorkLocation,
		StartDate:            details.StartDate,
		ExpiresAt:            expiresAt,
		TermsJson:            details.TermsJSON,
		SentSnapshotJson:     details.SentSnapshotJSON,
		CreatedBy:            details.CreatedBy,
		SentBy:               sentBy,
		DecidedAt:            decidedAt,
		CreatedAt:            FormatTime(details.CreatedAt),
		UpdatedAt:            FormatTime(details.UpdatedAt),
		JobTitle:             details.JobTitle,
		CandidateName:        details.CandidateName,
		ApplicationStatusKey: string(details.ApplicationStatusKey),
		CreatedByName:        details.CreatedByName,
		SentByName:           details.SentByName,
	}
}

func ToPBOffers(rows []repository.OfferDetails) []*pb.Offer {
	result := make([]*pb.Offer, 0, len(rows))
	for _, row := range rows {
		result = append(result, ToPBOffer(row))
	}
	return result
}

func ToPBOfferEvent(row event.OfferEvent) *pb.OfferEventInfo {
	return &pb.OfferEventInfo{
		Id:               int64(row.ID),
		OfferId:          row.OfferID,
		EventType:        string(row.EventType),
		ActorUserId:      row.ActorUserID,
		ActorAccountType: row.ActorAccountType,
		Reason:           row.Reason,
		MetadataJson:     row.MetadataJSON,
		CreatedAt:        FormatTime(row.CreatedAt),
	}
}

func ToPBOfferEvents(rows []event.OfferEvent) []*pb.OfferEventInfo {
	result := make([]*pb.OfferEventInfo, 0, len(rows))
	for _, row := range rows {
		result = append(result, ToPBOfferEvent(row))
	}
	return result
}

func FormatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}

func ParseOptionalRFC3339(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
