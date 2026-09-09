package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const fcmSendEndpointTemplate = "https://fcm.googleapis.com/v1/projects/%s/messages:send"

/*
FCMService delivers push notifications to Android devices via the Firebase Cloud
Messaging HTTP v1 API. It authenticates using a service account whose credentials
are read from the FIREBASE_PROJECT_ID, FIREBASE_CLIENT_EMAIL, and
FIREBASE_PRIVATE_KEY environment variables.
*/
type FCMService struct {
	projectID   string
	tokenSource oauth2.TokenSource
	httpClient  *http.Client
}

type fcmMessage struct {
	Message fcmMessagePayload `json:"message"`
}

type fcmMessagePayload struct {
	Token        string            `json:"token"`
	Notification fcmNotification   `json:"notification"`
	Data         map[string]string `json:"data,omitempty"`
}

type fcmNotification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

/*
NewFCMService constructs an FCMService by reading Firebase service account
credentials from environment variables. Returns nil if any required variable is
missing so callers can safely skip FCM delivery in environments without credentials.
*/
func NewFCMService() *FCMService {
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	clientEmail := os.Getenv("FIREBASE_CLIENT_EMAIL")
	privateKey := os.Getenv("FIREBASE_PRIVATE_KEY")

	if projectID == "" || clientEmail == "" || privateKey == "" {
		return nil
	}

	privateKey = strings.ReplaceAll(privateKey, `\n`, "\n")

	serviceAccountJSON, marshalErr := json.Marshal(map[string]string{
		"type":                        "service_account",
		"project_id":                  projectID,
		"client_email":                clientEmail,
		"private_key":                 privateKey,
		"token_uri":                   "https://oauth2.googleapis.com/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
	})
	if marshalErr != nil {
		return nil
	}

	credentials, credErr := google.CredentialsFromJSON(
		context.Background(),
		serviceAccountJSON,
		"https://www.googleapis.com/auth/firebase.messaging",
	)
	if credErr != nil {
		return nil
	}

	return &FCMService{
		projectID:   projectID,
		tokenSource: credentials.TokenSource,
		httpClient:  &http.Client{},
	}
}

/*
SendPushNotification delivers a push notification to a single device identified
by its FCM registration token. The data map is forwarded as-is to the device for
deep-linking purposes. A nil or empty deviceToken is silently ignored.
*/
func (service *FCMService) SendPushNotification(ctx context.Context, deviceToken, title, body string, data map[string]string) error {
	if deviceToken == "" {
		return nil
	}

	oauthToken, tokenErr := service.tokenSource.Token()
	if tokenErr != nil {
		return fmt.Errorf("fcm: failed to obtain oauth token: %w", tokenErr)
	}

	payload := fcmMessage{
		Message: fcmMessagePayload{
			Token:        deviceToken,
			Notification: fcmNotification{Title: title, Body: body},
			Data:         data,
		},
	}

	payloadBytes, _ := json.Marshal(payload)
	endpoint := fmt.Sprintf(fcmSendEndpointTemplate, service.projectID)

	request, requestErr := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payloadBytes))
	if requestErr != nil {
		return fmt.Errorf("fcm: failed to build request: %w", requestErr)
	}
	request.Header.Set("Authorization", "Bearer "+oauthToken.AccessToken)
	request.Header.Set("Content-Type", "application/json")

	response, responseErr := service.httpClient.Do(request)
	if responseErr != nil {
		return fmt.Errorf("fcm: http request failed: %w", responseErr)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(response.Body)
		return fmt.Errorf("fcm: non-200 response %d: %s", response.StatusCode, string(responseBody))
	}

	return nil
}
