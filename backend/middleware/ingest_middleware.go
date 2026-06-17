package middleware

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// RequireIngestKey validates the shared API secret key for the serverless scraper ingestion endpoints
func RequireIngestKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		ingestKey := c.GetHeader("X-Ingest-Key")
		expectedKey := os.Getenv("INGEST_API_KEY")
		
		// Fallback for development if not specified
		if expectedKey == "" {
			expectedKey = "dev-ingest-key-12345"
		}

		if ingestKey == "" || ingestKey != expectedKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing X-Ingest-Key"})
			return
		}

		c.Next()
	}
}
