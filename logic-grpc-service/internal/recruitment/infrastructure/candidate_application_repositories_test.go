package infrastructure

import (
	"reflect"
	"testing"

	"logic-grpc-service/repository"
)

func TestCandidateApplicationAdaptersUseCurrentRepositoryTypes(t *testing.T) {
	t.Parallel()

	if reflect.TypeOf((*CandidateProfileRepository)(nil)) != reflect.TypeOf((*repository.ProfileRepo)(nil)) {
		t.Fatal("recruitment candidate profile repository adapter drifted from current repository")
	}
	if reflect.TypeOf((*ResumeRepository)(nil)) != reflect.TypeOf((*repository.ResumeRepo)(nil)) {
		t.Fatal("recruitment resume repository adapter drifted from current repository")
	}
	if reflect.TypeOf((*ApplicationRepository)(nil)) != reflect.TypeOf((*repository.ApplicationRepo)(nil)) {
		t.Fatal("recruitment application repository adapter drifted from current repository")
	}
}
