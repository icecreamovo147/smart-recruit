package interfaces

import (
	"testing"

	"logic-grpc-service/service"
)

func TestCurrentInterviewServiceSatisfiesInterviewAPI(t *testing.T) {
	t.Parallel()

	var _ InterviewAPI = (*service.InterviewService)(nil)
}
