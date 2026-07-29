package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dhruv1249/Job-cruiser/backend/handlers"
	"github.com/Dhruv1249/Job-cruiser/backend/services"
	"github.com/gin-gonic/gin"
)

func TestTailorHandlerTailorResumeSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	geminiServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		responsePayload := map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]string{
							{"text": "\\documentclass{article}\n\\begin{document}\nTailored CV\n\\end{document}"},
						},
					},
				},
			},
		}
		json.NewEncoder(writer).Encode(responsePayload)
	}))
	defer geminiServer.Close()

	mcpServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(request.Body).Decode(&body)
		tool := body["tool"].(string)

		if tool == "write_project_file" {
			json.NewEncoder(writer).Encode(map[string]interface{}{
				"success": true,
				"result":  map[string]interface{}{"message": "wrote file"},
			})
			return
		}
		if tool == "compile_project" {
			json.NewEncoder(writer).Encode(map[string]interface{}{
				"success": true,
				"result": map[string]interface{}{
					"status":    "compiled",
					"engine":    "xelatex",
					"outputLog": "PDF written",
					"errors":    "",
					"pdfPath":   "/projects/my_project/main.pdf",
				},
			})
			return
		}
		if tool == "get_project_pdf" {
			json.NewEncoder(writer).Encode(map[string]interface{}{
				"success": true,
				"result": map[string]interface{}{
					"fileName":   "main.pdf",
					"mimeType":   "application/pdf",
					"base64Data": "JVBERi0x...",
					"sizeBytes":  1024,
				},
			})
			return
		}
	}))
	defer mcpServer.Close()

	mcpClient := services.NewMCPClient(mcpServer.URL, "test-mcp-token")
	tailorService := services.NewResumeTailorService(geminiServer.URL, "test-gemini-key", mcpClient)
	tailorHandler := handlers.NewTailorHandler(tailorService, nil)

	router := gin.New()
	router.POST("/api/tailor/resume", tailorHandler.TailorResume)

	requestBody := map[string]string{
		"user_bio":        "Senior Engineer with Go and AWS experience.",
		"job_description": "We are seeking a Go Engineer for backend microservices.",
		"project_name":    "my_project",
		"file_path":       "main.tex",
	}
	jsonBytes, _ := json.Marshal(requestBody)

	httpRequest, _ := http.NewRequest(http.MethodPost, "/api/tailor/resume", bytes.NewBuffer(jsonBytes))
	httpRequest.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, httpRequest)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", recorder.Code, recorder.Body.String())
	}

	var responseMap map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &responseMap)

	if responseMap["project_name"] != "my_project" {
		t.Fatalf("unexpected project name in response: %v", responseMap)
	}
}
