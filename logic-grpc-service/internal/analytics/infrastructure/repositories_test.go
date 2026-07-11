package infrastructure

import (
	"reflect"
	"testing"

	"logic-grpc-service/repository"
)

func TestAnalyticsAdapterUsesCurrentRepositoryType(t *testing.T) {
	t.Parallel()

	if reflect.TypeOf((*ReportingReadModelRepository)(nil)) != reflect.TypeOf((*repository.AnalyticsRepo)(nil)) {
		t.Fatal("analytics repository adapter drifted from current repository")
	}
}
