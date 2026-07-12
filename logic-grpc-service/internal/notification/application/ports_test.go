package application

import (
	"testing"

	"logic-grpc-service/pkg/cache"
	"logic-grpc-service/repository"
	"logic-grpc-service/service"
)

func TestCurrentNotificationInfrastructureSatisfiesPorts(t *testing.T) {
	t.Parallel()

	var _ NotificationRepository = (*repository.NotificationRepo)(nil)
	var _ UnreadAndRealtimeCache = (*cache.NotificationCache)(nil)
	var _ AsyncNotificationWriter = (*service.NotificationWorkerPool)(nil)
	var _ OutboxEventPublisher = (*service.OutboxPublisher)(nil)
}
