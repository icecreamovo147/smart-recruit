package interfaces

import (
	"testing"

	"logic-grpc-service/service"
)

func TestCurrentNotificationServiceSatisfiesNotificationAPI(t *testing.T) {
	t.Parallel()

	var _ NotificationAPI = (*service.NotificationService)(nil)
}
