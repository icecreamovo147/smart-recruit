package interfaces

import (
	"testing"

	"logic-grpc-service/service"
)

func TestCurrentAnalyticsServiceSatisfiesAnalyticsQueryAPI(t *testing.T) {
	t.Parallel()

	var _ AnalyticsQueryAPI = (*service.AnalyticsService)(nil)
}
