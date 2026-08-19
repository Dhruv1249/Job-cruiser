package services_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestBuildJobFolderPathSlugifiesCorrectly(t *testing.T) {
	folderPath := services.BuildJobFolderPath("Stripe, Inc.", "Senior Software Engineer")
	if !strings.Contains(folderPath, "stripe") {
		t.Fatalf("expected folder path to contain 'stripe', got: %s", folderPath)
	}
	if !strings.Contains(folderPath, "senior_software_engineer") {
		t.Fatalf("expected folder path to contain 'senior_software_engineer', got: %s", folderPath)
	}
}

func TestSanitizeGeminiLatexStripsMarkdownFences(t *testing.T) {
	input := "```latex\n\\documentclass{article}\n\\begin{document}\nHello\n\\end{document}\n```"
	result := services.ExposedSanitizeGeminiLatex(input)
	if strings.Contains(result, "```") {
		t.Fatalf("sanitized output must not contain markdown fences, got: %s", result)
	}
	if !strings.Contains(result, "\\documentclass") {
		t.Fatalf("sanitized output must retain LaTeX content, got: %s", result)
	}
}

func TestTailorResumeToFolderSuccess(t *testing.T) {
	geminiServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		responsePayload := map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]string{
							{"text": "\\documentclass{article}\n\\begin{document}\nTailored resume\n\\end{document}"},
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
		tool, _ := body["tool"].(string)
		switch tool {
		case "write_project_file":
			json.NewEncoder(writer).Encode(map[string]interface{}{"success": true, "result": map[string]interface{}{"message": "written"}})
		case "compile_project":
			json.NewEncoder(writer).Encode(map[string]interface{}{
				"success": true,
				"result": map[string]interface{}{"status": "compiled", "engine": "xelatex", "pageCount": 1, "outputLog": "OK"},
			})
		case "get_project_pdf":
			json.NewEncoder(writer).Encode(map[string]interface{}{
				"success": true,
				"result": map[string]interface{}{"fileName": "resume.pdf", "mimeType": "application/pdf", "pageCount": 1, "base64Data": "AAAA", "sizeBytes": 1024},
			})
		}
	}))
	defer mcpServer.Close()

	mcpClient := services.NewMCPClient(mcpServer.URL, "test-token")
	tailorService := services.NewResumeTailorService(geminiServer.URL, "test-key", nil)

	jobContext := services.JobTailoringContext{
		Title:     "Backend Engineer",
		Company:   "Acme Corp",
		Seniority: "Senior",
		TechStack: []string{"Go", "PostgreSQL"},
		RawDesc:   "We need a senior backend engineer.",
	}

	result, err := tailorService.TailorResumeToFolder(
		context.Background(),
		mcpClient,
		"5 years Go experience",
		jobContext,
		"acme_corp_backend_engineer_20260820",
		1,
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.CompileResult.Status != "compiled" {
		t.Fatalf("expected compiled status, got: %s", result.CompileResult.Status)
	}
	if result.FolderPath != "acme_corp_backend_engineer_20260820" {
		t.Fatalf("unexpected folder path: %s", result.FolderPath)
	}
	if result.PDFResult.SizeBytes != 1024 {
		t.Fatalf("expected 1024 byte PDF, got %d", result.PDFResult.SizeBytes)
	}
}

func TestGenerateCoverLetterToFolderSuccess(t *testing.T) {
	geminiServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		respPayload := map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]string{
							{"text": "\\documentclass{letter}\n\\begin{document}\nDear Hiring Manager\n\\end{document}"},
						},
					},
				},
			},
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(respPayload)
	}))
	defer geminiServer.Close()

	mcpServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(request.Body).Decode(&body)
		tool, _ := body["tool"].(string)
		switch tool {
		case "write_project_file":
			json.NewEncoder(writer).Encode(map[string]interface{}{"success": true, "result": map[string]interface{}{"message": "written"}})
		case "compile_project":
			json.NewEncoder(writer).Encode(map[string]interface{}{
				"success": true,
				"result": map[string]interface{}{"status": "compiled", "pageCount": 1, "outputLog": "OK"},
			})
		case "get_project_pdf":
			json.NewEncoder(writer).Encode(map[string]interface{}{
				"success": true,
				"result": map[string]interface{}{"fileName": "cover_letter.pdf", "mimeType": "application/pdf", "pageCount": 1, "base64Data": "BBBB", "sizeBytes": 512},
			})
		}
	}))
	defer mcpServer.Close()

	mcpClient := services.NewMCPClient(mcpServer.URL, "test-token")
	tailorService := services.NewResumeTailorService(geminiServer.URL, "test-key", nil)

	jobContext := services.JobTailoringContext{
		Title:     "Product Manager",
		Company:   "TechCo",
		RawDesc:   "We want a PM for our B2B product.",
		TechStack: []string{},
	}

	result, err := tailorService.GenerateCoverLetterToFolder(
		context.Background(),
		mcpClient,
		"Experienced PM with B2B background",
		jobContext,
		"techco_product_manager_20260820",
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.FilePath != "techco_product_manager_20260820/cover_letter.tex" {
		t.Fatalf("unexpected file path: %s", result.FilePath)
	}
	if result.TargetPages != 1 {
		t.Fatalf("expected target pages 1, got %d", result.TargetPages)
	}
}

