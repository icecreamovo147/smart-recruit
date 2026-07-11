package application

import (
	"testing"

	"logic-grpc-service/repository"
)

func TestCurrentRepositorySatisfiesInterviewPorts(t *testing.T) {
	t.Parallel()

	var _ InterviewRepository = (*repository.InterviewRepo)(nil)
}
