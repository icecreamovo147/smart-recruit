package handler

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"web-gin-service/middleware"
	"web-gin-service/recruitment/pb"
	"web-gin-service/rpc"
)

type NotificationHandler struct {
	clients *rpc.Clients
	rdb     *redis.Client
}

func NewNotificationHandler(clients *rpc.Clients, rdb *redis.Client) *NotificationHandler {
	return &NotificationHandler{clients: clients, rdb: rdb}
}

// accountType extracts the user's account_type from the Gin context.
func accountType(c *gin.Context) string {
	return middleware.AccountType(c)
}

func (h *NotificationHandler) List(c *gin.Context) {
	userID := middleware.UserID(c)
	acctType := accountType(c)
	page, pageSize := int32(1), int32(20)
	if v, err := strconv.Atoi(c.Query("page")); err == nil && v > 0 {
		page = int32(v)
	}
	if v, err := strconv.Atoi(c.Query("page_size")); err == nil && v > 0 && v <= 50 {
		pageSize = int32(v)
	}
	cursor, hasCursor := c.GetQuery("cursor")
	if hasCursor {
		page = 0
	}
	resp, err := h.clients.Notification.ListNotifications(c.Request.Context(), &pb.ListNotificationsRequest{
		UserId:      userID,
		AccountType: acctType,
		Page:        page,
		PageSize:    pageSize,
		Cursor:      cursor,
	})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	resp, err := h.clients.Notification.UnreadNotificationCount(c.Request.Context(), &pb.UnreadNotificationCountRequest{
		UserId:      middleware.UserID(c),
		AccountType: accountType(c),
	})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func (h *NotificationHandler) Summary(c *gin.Context) {
	resp, err := h.clients.Notification.NotificationSummary(c.Request.Context(), &pb.NotificationSummaryRequest{
		UserId:      middleware.UserID(c),
		AccountType: accountType(c),
	})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func (h *NotificationHandler) Stream(c *gin.Context) {
	if h.rdb == nil {
		Internal(c, redis.Nil)
		return
	}
	userID := middleware.UserID(c)
	acctType := accountType(c)
	pubsub := h.rdb.Subscribe(c.Request.Context(), notificationEventChannel(userID, acctType))
	defer pubsub.Close()

	if _, err := pubsub.Receive(c.Request.Context()); err != nil {
		Internal(c, err)
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(200)
	flushWriter(c.Writer)

	ch := pubsub.Channel()
	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			pubsub.Close()
			return
		case msg := <-ch:
			if msg == nil {
				pubsub.Close()
				return
			}
			line := fmt.Sprintf("event: notification\ndata: %s\n\n", msg.Payload)
			if n, err := c.Writer.Write([]byte(line)); err != nil || n == 0 {
				pubsub.Close()
				return
			}
			if !flushWriter(c.Writer) {
				pubsub.Close()
				return
			}
		case <-heartbeat.C:
			line := []byte(": heartbeat\n\n")
			if n, err := c.Writer.Write(line); err != nil || n == 0 {
				pubsub.Close()
				return
			}
			if !flushWriter(c.Writer) {
				pubsub.Close()
				return
			}
		}
	}
}

// flushWriter is an alias for FlushSSE provided by response.go for internal use.
func flushWriter(w gin.ResponseWriter) bool {
	return FlushSSE(w)
}

func (h *NotificationHandler) MarkRead(c *gin.Context) {
	notificationID, err := strconv.ParseInt(c.Param("notification_id"), 10, 64)
	if err != nil {
		BadRequest(c, "通知 ID 不合法")
		return
	}
	resp, err := h.clients.Notification.MarkNotificationRead(c.Request.Context(), &pb.MarkNotificationReadRequest{
		UserId:         middleware.UserID(c),
		AccountType:    accountType(c),
		NotificationId: notificationID,
	})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func notificationEventChannel(userID int64, acctType string) string {
	return fmt.Sprintf("notif:event:%s:%d", acctType, userID)
}

func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	resp, err := h.clients.Notification.MarkAllNotificationsRead(c.Request.Context(), &pb.MarkAllNotificationsReadRequest{
		UserId:      middleware.UserID(c),
		AccountType: accountType(c),
	})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}
