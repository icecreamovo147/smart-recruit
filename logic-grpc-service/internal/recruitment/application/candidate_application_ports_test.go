package application

import (
	"testing"

	"logic-grpc-service/repository"
)

func TestCurrentRepositoriesSatisfyRecruitmentCandidateApplicationPorts(t *testing.T) {
	t.Parallel()

	var _ CandidateProfileRepository = (*repository.ProfileRepo)(nil)
	var _ ResumeRepository = (*repository.ResumeRepo)(nil)
	var _ ApplicationRepository = (*repository.ApplicationRepo)(nil)
}
