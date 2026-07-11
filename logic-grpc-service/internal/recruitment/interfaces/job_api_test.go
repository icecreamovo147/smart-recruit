package interfaces

import (
	"testing"

	"logic-grpc-service/service"
)

func TestCurrentJobServiceSatisfiesRecruitmentJobAPI(t *testing.T) {
	t.Parallel()

	var _ JobAPI = (*service.JobService)(nil)
	var _ JobTaxonomyAPI = (*service.JobTaxonomyService)(nil)
}
