package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dhruv1249/Job-cruiser/backend/handlers"
	"github.com/gin-gonic/gin"
)

func TestNotificationsRequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	notificationsHandler := handlers.NewNotificationsHandler(nil)

	t.Run("get notifications unauthorized without user_id", func(t *testing.T) {
		router := gin.New()
		router.GET("/api/notifications", notificationsHandler.GetNotifications)

		httpRequest, _ := http.NewRequest(http.MethodGet, "/api/notifications", nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httpRequest)

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", recorder.Code)
		}
	})

	t.Run("mark notification read unauthorized without user_id", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/notifications/:id/read", notificationsHandler.MarkNotificationAsRead)

		httpRequest, _ := http.NewRequest(http.MethodPost, "/api/notifications/test-id/read", nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httpRequest)

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", recorder.Code)
		}
	})

	t.Run("mark all read unauthorized without user_id", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/notifications/read-all", notificationsHandler.MarkAllNotificationsAsRead)

		httpRequest, _ := http.NewRequest(http.MethodPost, "/api/notifications/read-all", nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httpRequest)

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", recorder.Code)
		}
	})

	t.Run("unread count unauthorized without user_id", func(t *testing.T) {
		router := gin.New()
		router.GET("/api/notifications/unread-count", notificationsHandler.GetUnreadNotificationsCount)

		httpRequest, _ := http.NewRequest(http.MethodGet, "/api/notifications/unread-count", nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httpRequest)

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", recorder.Code)
		}
	})
}
