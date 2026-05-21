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

func (h *NotificationHandler) List(c *gin.Context) {
	userID := middleware.UserID(c)
	role := middleware.Role(c)
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
		UserId:   userID,
		Role:     notificationReceiverRole(role),
		Page:     page,
		PageSize: pageSize,
		Cursor:   cursor,
	})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	resp, err := h.clients.Notification.UnreadNotificationCount(c.Request.Context(), &pb.UnreadNotificationCountRequest{
		UserId: middleware.UserID(c),
		Role:   notificationReceiverRole(middleware.Role(c)),
	})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func (h *NotificationHandler) Summary(c *gin.Context) {
	resp, err := h.clients.Notification.NotificationSummary(c.Request.Context(), &pb.NotificationSummaryRequest{
		UserId: middleware.UserID(c),
		Role:   notificationReceiverRole(middleware.Role(c)),
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
	role := notificationReceiverRole(middleware.Role(c))
	pubsub := h.rdb.Subscribe(c.Request.Context(), notificationEventChannel(userID, role))
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
	c.Writer.Flush()

	ch := pubsub.Channel()
	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case msg := <-ch:
			if msg == nil {
				return
			}
			_, _ = fmt.Fprintf(c.Writer, "event: notification\ndata: %s\n\n", msg.Payload)
			c.Writer.Flush()
		case <-heartbeat.C:
			_, _ = fmt.Fprint(c.Writer, ": heartbeat\n\n")
			c.Writer.Flush()
		}
	}
}

func (h *NotificationHandler) MarkRead(c *gin.Context) {
	notificationID, err := strconv.ParseInt(c.Param("notification_id"), 10, 64)
	if err != nil {
		BadRequest(c, "通知 ID 不合法")
		return
	}
	resp, err := h.clients.Notification.MarkNotificationRead(c.Request.Context(), &pb.MarkNotificationReadRequest{
		UserId:         middleware.UserID(c),
		Role:           notificationReceiverRole(middleware.Role(c)),
		NotificationId: notificationID,
	})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func notificationEventChannel(userID int64, role int32) string {
	return fmt.Sprintf("notif:event:%d:%d", role, userID)
}

func notificationReceiverRole(role int32) int32 {
	if role >= 2 {
		return 2
	}
	return role
}

func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	resp, err := h.clients.Notification.MarkAllNotificationsRead(c.Request.Context(), &pb.MarkAllNotificationsReadRequest{
		UserId: middleware.UserID(c),
		Role:   notificationReceiverRole(middleware.Role(c)),
	})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}
