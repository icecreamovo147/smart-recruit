package interfaces

import (
	"testing"

	"web-gin-service/handler"
)

func TestCurrentNotificationHandlerSatisfiesHTTPAdapter(t *testing.T) {
	t.Parallel()

	var _ NotificationHTTPAdapter = (*handler.NotificationHandler)(nil)
}
