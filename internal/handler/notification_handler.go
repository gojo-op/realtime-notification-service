package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/realtime-notification-system/notification-service/internal/models"
	"github.com/realtime-notification-system/notification-service/internal/service"
)

type NotificationHandler struct {
	notificationService service.NotificationService
	userService         service.UserService
}

func NewNotificationHandler(notificationService service.NotificationService, userService service.UserService) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
		userService:         userService,
	}
}

func (h *NotificationHandler) CreateNotification(c *gin.Context) {
	var req struct {
		UserID   string                 `json:"user_id" binding:"required"`
		Type     string                 `json:"type" binding:"required,oneof=email sms push"`
		Title    string                 `json:"title" binding:"required"`
		Message  string                 `json:"message" binding:"required"`
		Priority string                 `json:"priority" binding:"required,oneof=low medium high urgent"`
		Metadata map[string]interface{} `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID format"})
		return
	}

	_, err = h.userService.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	notification, err := h.notificationService.CreateNotification(
		c.Request.Context(),
		userID,
		req.Title,
		req.Message,
		models.NotificationType(req.Type),
		models.NotificationPriority(req.Priority),
		req.Metadata,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create notification"})
		return
	}

	c.JSON(http.StatusCreated, notification)
}

func (h *NotificationHandler) CreateBatchNotifications(c *gin.Context) {
	var req struct {
		UserIDs  []string               `json:"user_ids" binding:"required"`
		Type     string                 `json:"type" binding:"required,oneof=email sms push"`
		Title    string                 `json:"title" binding:"required"`
		Message  string                 `json:"message" binding:"required"`
		Priority string                 `json:"priority" binding:"required,oneof=low medium high urgent"`
		Metadata map[string]interface{} `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.UserIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_ids cannot be empty"})
		return
	}

	if len(req.UserIDs) > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "batch size cannot exceed 1000"})
		return
	}

	userIDs := make([]uuid.UUID, 0, len(req.UserIDs))
	for _, userIDStr := range req.UserIDs {
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID format: " + userIDStr})
			return
		}
		userIDs = append(userIDs, userID)
	}

	created, err := h.notificationService.CreateBatchNotifications(
		c.Request.Context(),
		userIDs,
		req.Title,
		req.Message,
		models.NotificationType(req.Type),
		models.NotificationPriority(req.Priority),
		req.Metadata,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create batch notifications"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"created": created,
		"total":   len(req.UserIDs),
	})
}

func (h *NotificationHandler) GetNotification(c *gin.Context) {
	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid notification ID format"})
		return
	}

	notification, err := h.notificationService.GetNotification(c.Request.Context(), notificationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
		return
	}

	c.JSON(http.StatusOK, notification)
}

func (h *NotificationHandler) GetUserNotifications(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID format"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	status := c.Query("status")
	typeFilter := c.Query("type")

	if limit > 100 {
		limit = 100
	}

	filters := map[string]interface{}{
		"status": status,
		"type":   typeFilter,
	}
	page := offset / limit
	if page < 1 {
		page = 1
	}
	notifications, total, err := h.notificationService.GetUserNotifications(c.Request.Context(), userID, page, limit, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"total":         total,
		"limit":         limit,
		"offset":        offset,
	})
}

func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid notification ID format"})
		return
	}

	if err := h.notificationService.MarkAsRead(c.Request.Context(), notificationID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark notification as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "notification marked as read"})
}

func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID format"})
		return
	}

	count, err := h.notificationService.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get unread count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"unread_count": count})
}

func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID format"})
		return
	}

	count, err := h.notificationService.MarkAllAsRead(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark all as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"marked_count": count})
}
