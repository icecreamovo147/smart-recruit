package infrastructure

import (
	"reflect"
	"testing"

	"logic-grpc-service/internal/analytics/application"
	"logic-grpc-service/repository"
)

func TestAnalyticsAdapterUsesCurrentRepositoryType(t *testing.T) {
	t.Parallel()

	if reflect.TypeOf((*ReportingReadModelRepository)(nil)) != reflect.TypeOf((*repository.AnalyticsRepo)(nil)) {
		t.Fatal("analytics repository adapter drifted from current repository")
	}
}

func TestProjectionRepositorySatisfiesProjectionPorts(t *testing.T) {
	t.Parallel()

	var _ application.ProjectionEventStore = (*ProjectionRepository)(nil)
	var _ application.ProjectionCheckpointStore = (*ProjectionRepository)(nil)
}
