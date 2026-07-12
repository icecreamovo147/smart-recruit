package infrastructure

import (
	"reflect"
	"testing"

	"logic-grpc-service/repository"
)

func TestJobRepositoryAdaptersUseCurrentRepositoryTypes(t *testing.T) {
	t.Parallel()

	if reflect.TypeOf((*JobRepository)(nil)) != reflect.TypeOf((*repository.JobRepo)(nil)) {
		t.Fatal("recruitment job repository adapter drifted from current repository")
	}
	if reflect.TypeOf((*DepartmentRepository)(nil)) != reflect.TypeOf((*repository.DepartmentRepo)(nil)) {
		t.Fatal("recruitment department repository adapter drifted from current repository")
	}
	if reflect.TypeOf((*LocationRepository)(nil)) != reflect.TypeOf((*repository.JobLocationRepo)(nil)) {
		t.Fatal("recruitment location repository adapter drifted from current repository")
	}
}
