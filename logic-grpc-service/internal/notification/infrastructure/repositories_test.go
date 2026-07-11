package infrastructure

import (
	"reflect"
	"testing"

	"logic-grpc-service/pkg/cache"
	"logic-grpc-service/repository"
)

func TestNotificationAdaptersUseCurrentInfrastructureTypes(t *testing.T) {
	t.Parallel()

	if reflect.TypeOf((*NotificationRepository)(nil)) != reflect.TypeOf((*repository.NotificationRepo)(nil)) {
		t.Fatal("notification repository adapter drifted from current repository")
	}
	if reflect.TypeOf((*NotificationCache)(nil)) != reflect.TypeOf((*cache.NotificationCache)(nil)) {
		t.Fatal("notification cache adapter drifted from current cache")
	}
}
