package infrastructure

import (
	"gorm.io/gorm"

	"logic-grpc-service/repository"
)

// ReportingReadModelRepository adapts the current analytics query repository to Analytics application ports.
type ReportingReadModelRepository = repository.AnalyticsRepo

// NewReportingReadModelRepository creates an Analytics-owned reporting read-model adapter.
func NewReportingReadModelRepository(db *gorm.DB) *ReportingReadModelRepository {
	return repository.NewAnalyticsRepo(db)
}
