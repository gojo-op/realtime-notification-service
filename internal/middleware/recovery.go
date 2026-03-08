package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			log.Printf("Panic recovered: %s", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"message": "Internal server error",
					"code":    "INTERNAL_ERROR",
				},
			})
		} else if err, ok := recovered.(error); ok {
			log.Printf("Panic recovered: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"message": "Internal server error",
					"code":    "INTERNAL_ERROR",
				},
			})
		} else {
			log.Printf("Panic recovered: %v", recovered)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"message": "Internal server error",
					"code":    "INTERNAL_ERROR",
				},
			})
		}
		c.Abort()
	})
}
