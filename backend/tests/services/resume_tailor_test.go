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
		"job_applications",
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
		"job_applications",
		1,
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

func TestGetGeminiModelCascade(t *testing.T) {
	t.Run("default cascade when env is empty", func(t *testing.T) {
		t.Setenv("GEMINI_MODELS", "")
		t.Setenv("GEMINI_MODEL", "")
		cascade := services.GetGeminiModelCascade()
		if len(cascade) != 3 {
			t.Fatalf("expected 3 default models in cascade, got %d", len(cascade))
		}
		if cascade[0] != "gemini-3.5-flash-lite" || cascade[1] != "gemini-3.1-flash-lite" || cascade[2] != "gemini-2.5-flash-lite" {
			t.Fatalf("unexpected default cascade slice: %v", cascade)
		}
	})

	t.Run("parses GEMINI_MODELS comma separated list", func(t *testing.T) {
		t.Setenv("GEMINI_MODELS", "custom-model-1, custom-model-2 ,custom-model-3")
		t.Setenv("GEMINI_MODEL", "")
		cascade := services.GetGeminiModelCascade()
		if len(cascade) != 3 {
			t.Fatalf("expected 3 models, got %d", len(cascade))
		}
		if cascade[0] != "custom-model-1" || cascade[1] != "custom-model-2" || cascade[2] != "custom-model-3" {
			t.Fatalf("unexpected parsed cascade: %v", cascade)
		}
	})

	t.Run("falls back to GEMINI_MODEL when GEMINI_MODELS is unset", func(t *testing.T) {
		t.Setenv("GEMINI_MODELS", "")
		t.Setenv("GEMINI_MODEL", "single-fallback-model")
		cascade := services.GetGeminiModelCascade()
		if len(cascade) != 1 || cascade[0] != "single-fallback-model" {
			t.Fatalf("expected single fallback model, got %v", cascade)
		}
	})
}

func TestResumeTailorServiceCascadesOnRateLimit(t *testing.T) {
	calledModels := make([]string, 0)
	geminiServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		path := request.URL.Path
		if strings.Contains(path, "gemini-3.5-flash-lite") {
			calledModels = append(calledModels, "gemini-3.5-flash-lite")
			writer.WriteHeader(http.StatusTooManyRequests)
			writer.Write([]byte(`{"error":{"code":429,"message":"Resource has been exhausted"}}`))
			return
		}
		if strings.Contains(path, "gemini-3.1-flash-lite") {
			calledModels = append(calledModels, "gemini-3.1-flash-lite")
			responsePayload := map[string]interface{}{
				"candidates": []map[string]interface{}{
					{
						"content": map[string]interface{}{
							"parts": []map[string]string{
								{"text": "\\documentclass{article}\n\\begin{document}\nCascaded resume\n\\end{document}"},
							},
						},
					},
				},
			}
			writer.Header().Set("Content-Type", "application/json")
			json.NewEncoder(writer).Encode(responsePayload)
			return
		}
		writer.WriteHeader(http.StatusBadRequest)
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
			json.NewEncoder(writer).Encode(map[string]interface{}{"success": true, "result": map[string]interface{}{"status": "compiled", "engine": "xelatex", "pageCount": 1, "outputLog": "OK"}})
		case "get_project_pdf":
			json.NewEncoder(writer).Encode(map[string]interface{}{"success": true, "result": map[string]interface{}{"fileName": "main.pdf", "mimeType": "application/pdf", "pageCount": 1, "base64Data": "AAAA", "sizeBytes": 512}})
		}
	}))
	defer mcpServer.Close()

	mcpClient := services.NewMCPClient(mcpServer.URL, "token")
	tailorService := services.NewResumeTailorService(geminiServer.URL, "key", mcpClient)
	tailorService.GeminiModels = []string{"gemini-3.5-flash-lite", "gemini-3.1-flash-lite"}

	result, err := tailorService.TailorResumeDirect(context.Background(), "user bio", "job description", "job_proj", "main.tex", 1)
	if err != nil {
		t.Fatalf("expected successful cascade fallback, got error: %v", err)
	}
	if result.CompileResult.Status != "compiled" {
		t.Fatalf("expected compiled status, got: %s", result.CompileResult.Status)
	}
	if len(calledModels) != 2 || calledModels[0] != "gemini-3.5-flash-lite" || calledModels[1] != "gemini-3.1-flash-lite" {
		t.Fatalf("expected cascade order [gemini-3.5-flash-lite, gemini-3.1-flash-lite], got: %v", calledModels)
	}
}

func TestResumeTailorServiceFailsWhenAllModelsRateLimited(t *testing.T) {
	geminiServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusTooManyRequests)
		writer.Write([]byte(`{"error":{"code":429,"message":"Rate limit reached"}}`))
	}))
	defer geminiServer.Close()

	mcpClient := services.NewMCPClient("http://localhost:3202", "token")
	tailorService := services.NewResumeTailorService(geminiServer.URL, "key", mcpClient)
	tailorService.GeminiModels = []string{"model-1", "model-2"}

	_, err := tailorService.TailorResumeDirect(context.Background(), "user bio", "job description", "job_proj", "main.tex", 1)
	if err == nil {
		t.Fatalf("expected error when all models are rate limited, got nil")
	}
}

func TestTailorResumeSelfHealingOnCompileFailure(t *testing.T) {
	var geminiCallCount int
	geminiServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		geminiCallCount++
		text := "\\documentclass{article}\n\\begin{document}\nInitial LaTeX\n\\end{document}"
		if geminiCallCount > 1 {
			text = "\\documentclass{article}\n\\begin{document}\nHealed LaTeX\n\\end{document}"
		}
		responsePayload := map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]string{
							{"text": text},
						},
					},
				},
			},
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(responsePayload)
	}))
	defer geminiServer.Close()

	var compileCount int
	mcpServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(request.Body).Decode(&body)
		tool, _ := body["tool"].(string)
		switch tool {
		case "write_project_file":
			json.NewEncoder(writer).Encode(map[string]interface{}{"success": true, "result": map[string]interface{}{"message": "written"}})
		case "compile_project":
			compileCount++
			if compileCount == 1 {
				json.NewEncoder(writer).Encode(map[string]interface{}{
					"success": true,
					"result": map[string]interface{}{
						"status":    "failed",
						"engine":    "xelatex",
						"pageCount": 0,
						"outputLog": "! Undefined control sequence \\badmacro",
						"errors":    "Compilation failed",
					},
				})
			} else {
				json.NewEncoder(writer).Encode(map[string]interface{}{
					"success": true,
					"result": map[string]interface{}{
						"status":    "compiled",
						"engine":    "xelatex",
						"pageCount": 1,
						"outputLog": "Output written on resume.pdf (1 page).",
					},
				})
			}
		case "get_project_pdf":
			json.NewEncoder(writer).Encode(map[string]interface{}{
				"success": true,
				"result": map[string]interface{}{
					"fileName":   "resume.pdf",
					"mimeType":   "application/pdf",
					"pageCount":  1,
					"base64Data": "AAAA",
					"sizeBytes":  1024,
				},
			})
		}
	}))
	defer mcpServer.Close()

	mcpClient := services.NewMCPClient(mcpServer.URL, "test-token")
	tailorService := services.NewResumeTailorService(geminiServer.URL, "test-key", mcpClient)
	jobContext := services.JobTailoringContext{
		Title:     "Software Engineer",
		Company:   "Conga",
		Seniority: "Junior",
		TechStack: []string{"Go", "LaTeX"},
		RawDesc:   "Write robust code",
	}

	result, err := tailorService.TailorResumeToFolder(context.Background(), mcpClient, "Experience bank", jobContext, "conga_software_engineer", "job_applications", 1)
	if err != nil {
		t.Fatalf("expected successful recovery via self-healing, got error: %v", err)
	}
	if result.CompileResult.Status != "compiled" {
		t.Fatalf("expected compiled status after healing, got: %s", result.CompileResult.Status)
	}
	if compileCount != 2 {
		t.Fatalf("expected 2 compile attempts (1 failed + 1 healed), got %d", compileCount)
	}
	if geminiCallCount != 2 {
		t.Fatalf("expected 2 Gemini calls (initial + healing pass), got %d", geminiCallCount)
	}
}


func TestTailorResumeToFolderWithTemplateIncludesBaselineInPrompt(t *testing.T) {
	customTemplate := "\\documentclass{article}\n\\newcommand{\\myCustomMacro}[1]{\\textbf{#1}}\n\\begin{document}\nTemplate\n\\end{document}"
	var capturedGeminiPrompt string

	geminiServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var requestBody map[string]interface{}
		json.NewDecoder(request.Body).Decode(&requestBody)
		contents := requestBody["contents"].([]interface{})
		firstContent := contents[0].(map[string]interface{})
		parts := firstContent["parts"].([]interface{})
		firstPart := parts[0].(map[string]interface{})
		capturedGeminiPrompt = firstPart["text"].(string)

		responsePayload := map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]string{
							{"text": "\\documentclass{article}\n\\begin{document}\nTailored with custom template\n\\end{document}"},
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
		arguments, _ := body["arguments"].(map[string]interface{})

		switch tool {
		case "read_project_file":
			filePath := arguments["filePath"].(string)
			if filePath == "templates/custom_resume.tex" {
				json.NewEncoder(writer).Encode(map[string]interface{}{
					"success": true,
					"result":  map[string]interface{}{"content": customTemplate},
				})
				return
			}
			json.NewEncoder(writer).Encode(map[string]interface{}{
				"success": false,
				"error":   "not found",
			})
		case "list_files":
			json.NewEncoder(writer).Encode(map[string]interface{}{
				"success": true,
				"result":  map[string]interface{}{"files": []interface{}{}},
			})
		case "write_project_file":
			json.NewEncoder(writer).Encode(map[string]interface{}{"success": true, "result": map[string]interface{}{"message": "written"}})
		case "compile_project":
			json.NewEncoder(writer).Encode(map[string]interface{}{
				"success": true,
				"result":  map[string]interface{}{"status": "compiled", "pageCount": 1, "outputLog": "OK"},
			})
		case "get_project_pdf":
			json.NewEncoder(writer).Encode(map[string]interface{}{
				"success": true,
				"result":  map[string]interface{}{"fileName": "resume.pdf", "mimeType": "application/pdf", "pageCount": 1, "base64Data": "AAAA", "sizeBytes": 1024},
			})
		}
	}))
	defer mcpServer.Close()

	mcpClient := services.NewMCPClient(mcpServer.URL, "test-token")
	tailorService := services.NewResumeTailorService(geminiServer.URL, "test-key", mcpClient)
	jobContext := services.JobTailoringContext{
		Title:     "DevOps Engineer",
		Company:   "CloudCorp",
		Seniority: "Senior",
		TechStack: []string{"Kubernetes", "Terraform"},
		RawDesc:   "Manage cloud infrastructure",
	}

	result, err := tailorService.TailorResumeToFolderWithTemplate(
		context.Background(),
		mcpClient,
		"Experience with Kubernetes and Terraform",
		jobContext,
		"cloudcorp_devops",
		"job_applications",
		1,
		"templates/custom_resume.tex",
	)

	if err != nil {
		t.Fatalf("expected no error tailoring with custom template, got: %v", err)
	}
	if result.CompileResult.Status != "compiled" {
		t.Fatalf("expected compiled status, got: %s", result.CompileResult.Status)
	}
	if !strings.Contains(capturedGeminiPrompt, "myCustomMacro") {
		t.Fatalf("expected Gemini prompt to include custom macro from baseline template, prompt was: %s", capturedGeminiPrompt)
	}
}
