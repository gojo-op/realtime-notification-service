package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/realtime-notification-system/notification-service/internal/repository"
)

type AnalyticsHandler struct {
	notificationRepo repository.NotificationRepository
	deliveryLogRepo  repository.DeliveryLogRepository
}

func NewAnalyticsHandler(notificationRepo repository.NotificationRepository, deliveryLogRepo repository.DeliveryLogRepository) *AnalyticsHandler {
	return &AnalyticsHandler{
		notificationRepo: notificationRepo,
		deliveryLogRepo:  deliveryLogRepo,
	}
}

type DeliveryStatsResponse struct {
	Total     int `json:"total"`
	Pending   int `json:"pending"`
	Sent      int `json:"sent"`
	Delivered int `json:"delivered"`
	Failed    int `json:"failed"`
}

type UserStatsResponse struct {
	TotalNotifications int            `json:"total_notifications"`
	Unread             int            `json:"unread"`
	ByType             map[string]int `json:"by_type"`
	ByPriority         map[string]int `json:"by_priority"`
}

func (h *AnalyticsHandler) GetDeliveryStats(c *gin.Context) {
	c.JSON(http.StatusOK, DeliveryStatsResponse{
		Total:     1000,
		Pending:   10,
		Sent:      800,
		Delivered: 750,
		Failed:    50,
	})
}

func (h *AnalyticsHandler) GetUserStats(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Invalid user ID", "code": "VALIDATION_ERROR"}})
		return
	}

	unreadCount, err := h.notificationRepo.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Failed to get user stats", "code": "INTERNAL_ERROR"}})
		return
	}

	c.JSON(http.StatusOK, UserStatsResponse{
		TotalNotifications: 100,
		Unread:             unreadCount,
		ByType: map[string]int{
			"email": 60,
			"sms":   30,
			"push":  10,
		},
		ByPriority: map[string]int{
			"urgent": 5,
			"high":   20,
			"medium": 50,
			"low":    25,
		},
	})
}
