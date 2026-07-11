package infrastructure

import (
	"reflect"
	"testing"

	"logic-grpc-service/repository"
)

func TestOfferAdapterUsesCurrentRepositoryType(t *testing.T) {
	t.Parallel()

	if reflect.TypeOf((*OfferRepository)(nil)) != reflect.TypeOf((*repository.OfferRepo)(nil)) {
		t.Fatal("offer repository adapter drifted from current repository")
	}
}
