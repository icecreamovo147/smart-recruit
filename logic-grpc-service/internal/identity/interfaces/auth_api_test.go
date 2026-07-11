package interfaces

import (
	"testing"

	"logic-grpc-service/service"
)

func TestCurrentAuthServiceSatisfiesIdentityAPI(t *testing.T) {
	t.Parallel()

	var _ AuthAPI = (*service.AuthService)(nil)
}
