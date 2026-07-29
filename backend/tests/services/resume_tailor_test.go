package services_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dhruv1249/Job-cruiser/backend/services"
)

func TestResumeTailorServiceGenerateTailoredResumeSuccess(t *testing.T) {
	geminiServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		responsePayload := map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]string{
							{
								"text": "\\documentclass{article}\n\\begin{document}\nTailored Resume Content for Senior Software Engineer\n\\end{document}",
							},
						},
					},
				},
			},
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(responsePayload)
	}))
	defer geminiServer.Close()

	mcpServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(request.Body).Decode(&body)

		tool := body["tool"].(string)
		if tool == "write_project_file" {
			response := map[string]interface{}{
				"success": true,
				"result":  map[string]interface{}{"message": "wrote file"},
			}
			json.NewEncoder(writer).Encode(response)
			return
		}
		if tool == "compile_project" {
			response := map[string]interface{}{
				"success": true,
				"result": map[string]interface{}{
					"status":    "compiled",
					"engine":    "xelatex",
					"pageCount": 1,
					"outputLog": "Output written on main.pdf",
					"errors":    "",
					"pdfPath":   "/projects/resume_job1/main.pdf",
				},
			}
			json.NewEncoder(writer).Encode(response)
			return
		}
		if tool == "get_project_pdf" {
			response := map[string]interface{}{
				"success": true,
				"result": map[string]interface{}{
					"fileName":   "main.pdf",
					"mimeType":   "application/pdf",
					"pageCount":  1,
					"base64Data": "JVBERi0xLjQK...",
					"sizeBytes":  2048,
				},
			}
			json.NewEncoder(writer).Encode(response)
			return
		}
	}))
	defer mcpServer.Close()

	mcpClient := services.NewMCPClient(mcpServer.URL, "test-mcp-token")
	tailorService := services.NewResumeTailorService(geminiServer.URL, "test-gemini-key", mcpClient)

	jobDescription := "Senior Go Developer required with expertise in cloud native microservices."
	userBio := "Experienced backend software engineer with 5 years in Go and Kubernetes."

	result, err := tailorService.TailorResumeDirect(context.Background(), userBio, jobDescription, "resume_job1", "main.tex", 1)
	if err != nil {
		t.Fatalf("expected no error tailoring resume, got: %v", err)
	}

	if result.CompileResult.Status != "compiled" {
		t.Fatalf("expected compiled status, got: %s", result.CompileResult.Status)
	}

	if result.TargetPages != 1 {
		t.Fatalf("expected target pages 1, got %d", result.TargetPages)
	}

	if result.PDFResult.SizeBytes != 2048 {
		t.Fatalf("expected 2048 bytes pdf, got %d", result.PDFResult.SizeBytes)
	}
}
