package application

import (
	"testing"

	"logic-grpc-service/repository"
)

func TestCurrentRepositoriesSatisfyRecruitmentJobPorts(t *testing.T) {
	t.Parallel()

	var _ JobRepository = (*repository.JobRepo)(nil)
	var _ JobReadModelRepository = (*repository.JobRepo)(nil)
	var _ DepartmentRepository = (*repository.DepartmentRepo)(nil)
	var _ LocationRepository = (*repository.JobLocationRepo)(nil)
}
