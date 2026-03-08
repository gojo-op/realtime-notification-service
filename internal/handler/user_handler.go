package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/realtime-notification-system/notification-service/internal/service"
)

type UserHandler struct {
	service service.UserService
}

func NewUserHandler(service service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

type CreateUserRequest struct {
	Email        string `json:"email" binding:"required,email"`
	Phone        string `json:"phone,omitempty"`
	FullName     string `json:"full_name" binding:"required"`
	EmailEnabled bool   `json:"email_enabled"`
	SMSEnabled   bool   `json:"sms_enabled"`
	PushEnabled  bool   `json:"push_enabled"`
}

type UpdateUserPreferencesRequest struct {
	EmailEnabled bool `json:"email_enabled"`
	SMSEnabled   bool `json:"sms_enabled"`
	PushEnabled  bool `json:"push_enabled"`
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error(), "code": "VALIDATION_ERROR"}})
		return
	}

	user, err := h.service.CreateUser(c.Request.Context(), req.Email, req.Phone, req.FullName, req.EmailEnabled, req.SMSEnabled, req.PushEnabled)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Failed to create user", "code": "INTERNAL_ERROR"}})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (h *UserHandler) GetUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Invalid user ID", "code": "VALIDATION_ERROR"}})
		return
	}

	user, err := h.service.GetUser(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "User not found", "code": "NOT_FOUND"}})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) UpdateUserPreferences(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Invalid user ID", "code": "VALIDATION_ERROR"}})
		return
	}

	var req UpdateUserPreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error(), "code": "VALIDATION_ERROR"}})
		return
	}

	if err := h.service.UpdateUserPreferences(c.Request.Context(), id, req.EmailEnabled, req.SMSEnabled, req.PushEnabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Failed to update user preferences", "code": "INTERNAL_ERROR"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
