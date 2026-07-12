package interfaces

import "github.com/gin-gonic/gin"

// NotificationHTTPAdapter is the gateway-owned transport adapter for notification HTTP and SSE routes.
type NotificationHTTPAdapter interface {
	List(*gin.Context)
	UnreadCount(*gin.Context)
	Summary(*gin.Context)
	Stream(*gin.Context)
	MarkRead(*gin.Context)
	MarkAllRead(*gin.Context)
}
