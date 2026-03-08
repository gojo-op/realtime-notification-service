package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/realtime-notification-system/notification-service/internal/config"
	"github.com/realtime-notification-system/notification-service/internal/database"
	"github.com/realtime-notification-system/notification-service/internal/handler"
	"github.com/realtime-notification-system/notification-service/internal/kafka"
	"github.com/realtime-notification-system/notification-service/internal/middleware"
	"github.com/realtime-notification-system/notification-service/internal/repository"
	"github.com/realtime-notification-system/notification-service/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.NewConnection(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := database.RunMigrations(db.DB, "migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	producer, err := kafka.NewProducer(&cfg.Kafka)
	if err != nil {
		log.Fatalf("Failed to create Kafka producer: %v", err)
	}
	defer producer.Close()

	userRepo := repository.NewUserRepository(db.DB)
	notificationRepo := repository.NewNotificationRepository(db.DB)
	deliveryLogRepo := repository.NewDeliveryLogRepository(db.DB)

	userService := service.NewUserService(userRepo)
	notificationService := service.NewNotificationService(notificationRepo, userRepo, producer)
	analyticsHandler := handler.NewAnalyticsHandler(notificationRepo, deliveryLogRepo)

	userHandler := handler.NewUserHandler(userService)
	notificationHandler := handler.NewNotificationHandler(notificationService, userService)

	router := setupRouter(cfg, userHandler, notificationHandler, analyticsHandler)

	srv := &http.Server{
		Addr:         ":" + cfg.API.Port,
		Handler:      router,
		ReadTimeout:  cfg.API.ReadTimeout,
		WriteTimeout: cfg.API.WriteTimeout,
		IdleTimeout:  cfg.API.IdleTimeout,
	}

	go func() {
		log.Printf("Starting API server on port %s", cfg.API.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func setupRouter(cfg *config.Config, userHandler *handler.UserHandler, notificationHandler *handler.NotificationHandler, analyticsHandler *handler.AnalyticsHandler) *gin.Engine {
	if cfg.Log.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.RequestID())

	api := router.Group("/api/v1")
	{
		users := api.Group("/users")
		{
			users.POST("", userHandler.CreateUser)
			users.GET("/:id", userHandler.GetUser)
			users.PUT("/:id/preferences", userHandler.UpdateUserPreferences)
		}

		notifications := api.Group("/notifications")
		{
			notifications.POST("", notificationHandler.CreateNotification)
			notifications.POST("/batch", notificationHandler.CreateBatchNotifications)
			notifications.GET("/:id", notificationHandler.GetNotification)
			notifications.PUT("/:id/read", notificationHandler.MarkAsRead)
		}

		usersNotifications := api.Group("/users/:user_id/notifications")
		{
			usersNotifications.GET("", notificationHandler.GetUserNotifications)
			usersNotifications.GET("/unread-count", notificationHandler.GetUnreadCount)
			usersNotifications.PUT("/read-all", notificationHandler.MarkAllAsRead)
		}

		analytics := api.Group("/analytics")
		{
			analytics.GET("/delivery-stats", analyticsHandler.GetDeliveryStats)
			analytics.GET("/users/:user_id/stats", analyticsHandler.GetUserStats)
		}
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	return router
}
