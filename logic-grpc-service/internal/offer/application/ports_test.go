package application

import (
	"testing"

	"logic-grpc-service/repository"
)

func TestCurrentRepositorySatisfiesOfferPorts(t *testing.T) {
	t.Parallel()

	var _ OfferRepository = (*repository.OfferRepo)(nil)
}
