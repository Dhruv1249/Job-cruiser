package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware provides cross-origin resource sharing configuration for web clients.
func CORSMiddleware() gin.HandlerFunc {
	allowedOriginsEnvironmentVariable := os.Getenv("ALLOWED_ORIGINS")

	return func(ginContext *gin.Context) {
		requestOriginHeader := ginContext.Request.Header.Get("Origin")
		resolvedAllowedOrigin := "*"

		if allowedOriginsEnvironmentVariable != "" && requestOriginHeader != "" {
			configuredOriginsList := strings.Split(allowedOriginsEnvironmentVariable, ",")
			originMatched := false
			for _, singleConfiguredOrigin := range configuredOriginsList {
				trimmedOrigin := strings.TrimSpace(singleConfiguredOrigin)
				if trimmedOrigin == requestOriginHeader {
					resolvedAllowedOrigin = requestOriginHeader
					originMatched = true
					break
				}
			}
			if !originMatched && len(configuredOriginsList) > 0 {
				resolvedAllowedOrigin = strings.TrimSpace(configuredOriginsList[0])
			}
		} else if requestOriginHeader != "" {
			resolvedAllowedOrigin = requestOriginHeader
		}

		ginContext.Writer.Header().Set("Access-Control-Allow-Origin", resolvedAllowedOrigin)
		ginContext.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		ginContext.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		ginContext.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		ginContext.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if ginContext.Request.Method == http.MethodOptions {
			ginContext.AbortWithStatus(http.StatusNoContent)
			return
		}

		ginContext.Next()
	}
}
