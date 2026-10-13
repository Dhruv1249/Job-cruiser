package middleware

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

/*
RequireAuth validates the Bearer JWT authorization header and populates user context.
*/
func RequireAuth() gin.HandlerFunc {
	return func(ginContext *gin.Context) {
		authorizationHeader := ginContext.GetHeader("Authorization")
		if authorizationHeader == "" {
			ginContext.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is missing"})
			return
		}

		authorizationHeaderParts := strings.Split(authorizationHeader, " ")
		if len(authorizationHeaderParts) != 2 || authorizationHeaderParts[0] != "Bearer" {
			ginContext.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format. Expected 'Bearer <token>'"})
			return
		}
		tokenString := authorizationHeaderParts[1]

		secretKey := os.Getenv("JWT_SECRET")
		if secretKey == "" {
			log.Fatalf("CRITICAL ERROR: JWT_SECRET environment variable is missing.")
		}

		parsedToken, parseError := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, isHMAC := token.Method.(*jwt.SigningMethodHMAC); !isHMAC {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secretKey), nil
		})

		if parseError != nil || !parsedToken.Valid {
			ginContext.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		tokenClaims, claimsValid := parsedToken.Claims.(jwt.MapClaims)
		if !claimsValid {
			ginContext.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Failed to parse token claims"})
			return
		}

		userID, userIDValid := tokenClaims["user_id"].(string)
		if !userIDValid || userID == "" {
			ginContext.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid user_id in token"})
			return
		}

		ginContext.Set("user_id", userID)
		ginContext.Next()
	}
}
