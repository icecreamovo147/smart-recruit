package infrastructure

import (
	"reflect"
	"testing"

	"logic-grpc-service/repository"
)

func TestInterviewAdapterUsesCurrentRepositoryType(t *testing.T) {
	t.Parallel()

	if reflect.TypeOf((*InterviewRepository)(nil)) != reflect.TypeOf((*repository.InterviewRepo)(nil)) {
		t.Fatal("interview repository adapter drifted from current repository")
	}
}
