package interfaces

import (
	"testing"

	"logic-grpc-service/service"
)

func TestCurrentOfferServiceSatisfiesOfferAPI(t *testing.T) {
	t.Parallel()

	var _ OfferAPI = (*service.OfferService)(nil)
}
