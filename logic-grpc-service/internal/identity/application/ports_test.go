package application

import (
	"testing"

	"logic-grpc-service/repository"
)

func TestCurrentRepositoriesSatisfyIdentityPorts(t *testing.T) {
	t.Parallel()

	var _ UserRepository = (*repository.UserRepo)(nil)
	var _ RefreshTokenRepository = (*repository.RefreshTokenRepo)(nil)
	var _ AuthorizationRepository = (*repository.AuthzRepo)(nil)
}
