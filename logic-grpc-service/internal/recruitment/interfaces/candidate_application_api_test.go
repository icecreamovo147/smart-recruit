package interfaces

import (
	"testing"

	"logic-grpc-service/service"
)

func TestCurrentCandidateAndApplicationServicesSatisfyRecruitmentAPIs(t *testing.T) {
	t.Parallel()

	var _ CandidateProfileAPI = (*service.CandidateService)(nil)
	var _ ApplicationAPI = (*service.ApplicationService)(nil)
}
