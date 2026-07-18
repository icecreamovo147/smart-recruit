package interfaces

import (
	"testing"

	"smart-recruit-gateway/handler"
)

func TestCurrentNotificationHandlerSatisfiesHTTPAdapter(t *testing.T) {
	t.Parallel()

	var _ NotificationHTTPAdapter = (*handler.NotificationHandler)(nil)
}
