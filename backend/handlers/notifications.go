package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

/*
NotificationsHandler manages user notifications for background operations like tailoring and scraping.
*/
type NotificationsHandler struct {
	DB *pgxpool.Pool
}

/*
NotificationItem represents a single notification record.
*/
type NotificationItem struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

/*
NewNotificationsHandler initializes a new NotificationsHandler.
*/
func NewNotificationsHandler(dbPool *pgxpool.Pool) *NotificationsHandler {
	return &NotificationsHandler{
		DB: dbPool,
	}
}

/*
GetNotifications returns the most recent notifications for the authenticated user.
*/
func (handler *NotificationsHandler) GetNotifications(ginContext *gin.Context) {
	userIDValue, exists := ginContext.Get("user_id")
	if !exists {
		ginContext.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := fmt.Sprintf("%v", userIDValue)

	if handler.DB == nil {
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "Database unavailable"})
		return
	}

	rows, queryError := handler.DB.Query(
		ginContext.Request.Context(),
		`SELECT id, user_id, title, message, is_read, created_at
		 FROM notifications
		 WHERE user_id = $1
		 ORDER BY created_at DESC
		 LIMIT 50`,
		userID,
	)
	if queryError != nil {
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "Failed fetching notifications: " + queryError.Error()})
		return
	}
	defer rows.Close()

	notificationsList := make([]NotificationItem, 0)
	for rows.Next() {
		var item NotificationItem
		if scanError := rows.Scan(&item.ID, &item.UserID, &item.Title, &item.Message, &item.IsRead, &item.CreatedAt); scanError == nil {
			notificationsList = append(notificationsList, item)
		}
	}

	ginContext.JSON(http.StatusOK, gin.H{
		"data": notificationsList,
	})
}

/*
MarkNotificationAsRead marks a specific notification as read.
*/
func (handler *NotificationsHandler) MarkNotificationAsRead(ginContext *gin.Context) {
	userIDValue, exists := ginContext.Get("user_id")
	if !exists {
		ginContext.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := fmt.Sprintf("%v", userIDValue)
	notificationID := ginContext.Param("id")

	if handler.DB == nil {
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "Database unavailable"})
		return
	}

	_, updateError := handler.DB.Exec(
		ginContext.Request.Context(),
		`UPDATE notifications SET is_read = true WHERE id = $1 AND user_id = $2`,
		notificationID, userID,
	)
	if updateError != nil {
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "Failed updating notification: " + updateError.Error()})
		return
	}

	ginContext.JSON(http.StatusOK, gin.H{"message": "Notification marked as read"})
}

/*
MarkAllNotificationsAsRead marks all unread notifications for the user as read.
*/
func (handler *NotificationsHandler) MarkAllNotificationsAsRead(ginContext *gin.Context) {
	userIDValue, exists := ginContext.Get("user_id")
	if !exists {
		ginContext.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := fmt.Sprintf("%v", userIDValue)

	if handler.DB == nil {
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "Database unavailable"})
		return
	}

	_, updateError := handler.DB.Exec(
		ginContext.Request.Context(),
		`UPDATE notifications SET is_read = true WHERE user_id = $1`,
		userID,
	)
	if updateError != nil {
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "Failed updating notifications: " + updateError.Error()})
		return
	}

	ginContext.JSON(http.StatusOK, gin.H{"message": "All notifications marked as read"})
}

/*
GetUnreadNotificationsCount returns the number of unread notifications for the user.
*/
func (handler *NotificationsHandler) GetUnreadNotificationsCount(ginContext *gin.Context) {
	userIDValue, exists := ginContext.Get("user_id")
	if !exists {
		ginContext.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := fmt.Sprintf("%v", userIDValue)

	if handler.DB == nil {
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "Database unavailable"})
		return
	}

	var unreadCount int
	queryError := handler.DB.QueryRow(
		ginContext.Request.Context(),
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false`,
		userID,
	).Scan(&unreadCount)
	if queryError != nil {
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "Failed counting unread notifications: " + queryError.Error()})
		return
	}

	ginContext.JSON(http.StatusOK, gin.H{"unread_count": unreadCount})
}
