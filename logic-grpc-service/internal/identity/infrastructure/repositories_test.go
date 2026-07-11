package infrastructure

import (
	"reflect"
	"testing"

	"logic-grpc-service/repository"
)

func TestRepositoryAdaptersUseCurrentRepositoryTypes(t *testing.T) {
	t.Parallel()

	if reflect.TypeOf((*UserRepository)(nil)) != reflect.TypeOf((*repository.UserRepo)(nil)) {
		t.Fatal("identity user repository adapter drifted from current repository")
	}
	if reflect.TypeOf((*RefreshTokenRepository)(nil)) != reflect.TypeOf((*repository.RefreshTokenRepo)(nil)) {
		t.Fatal("identity refresh-token repository adapter drifted from current repository")
	}
	if reflect.TypeOf((*AuthorizationRepository)(nil)) != reflect.TypeOf((*repository.AuthzRepo)(nil)) {
		t.Fatal("identity authorization repository adapter drifted from current repository")
	}
}
