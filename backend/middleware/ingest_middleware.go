package middleware

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

/*
RequireIngestKey validates the shared API secret key for the serverless scraper ingestion endpoints.
*/
func RequireIngestKey() gin.HandlerFunc {
	return func(ginContext *gin.Context) {
		ingestKeyHeader := ginContext.GetHeader("X-Ingest-Key")
		expectedIngestKey := os.Getenv("INGEST_API_KEY")

		if expectedIngestKey == "" || ingestKeyHeader == "" || ingestKeyHeader != expectedIngestKey {
			ginContext.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing X-Ingest-Key"})
			return
		}

		ginContext.Next()
	}
}
