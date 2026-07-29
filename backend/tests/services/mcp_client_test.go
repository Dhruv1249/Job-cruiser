package services_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dhruv1249/Job-cruiser/backend/services"
)

func TestMCPClientListProjectsSuccess(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer test-mcp-token" {
			http.Error(writer, "unauthorized", http.StatusUnauthorized)
			return
		}
		if request.Method != http.MethodPost || request.URL.Path != "/api/mcp/tool" {
			http.Error(writer, "not found", http.StatusNotFound)
			return
		}

		responsePayload := map[string]interface{}{
			"success": true,
			"result": map[string]interface{}{
				"projects": []string{"resume_master", "cover_letter_google"},
			},
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(responsePayload)
	}))
	defer mockServer.Close()

	client := services.NewMCPClient(mockServer.URL, "test-mcp-token")
	projects, err := client.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}
	if projects[0] != "resume_master" || projects[1] != "cover_letter_google" {
		t.Fatalf("unexpected project names: %v", projects)
	}
}

func TestMCPClientReadProjectFileSuccess(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var requestBody map[string]interface{}
		json.NewDecoder(request.Body).Decode(&requestBody)

		if requestBody["tool"] != "read_project_file" {
			http.Error(writer, "invalid tool", http.StatusBadRequest)
			return
		}

		args := requestBody["arguments"].(map[string]interface{})
		if args["projectName"] != "resume_master" || args["filePath"] != "main.tex" {
			http.Error(writer, "invalid args", http.StatusBadRequest)
			return
		}

		responsePayload := map[string]interface{}{
			"success": true,
			"result": map[string]interface{}{
				"content": "\\documentclass{article}\n\\begin{document}\nHello World\n\\end{document}",
			},
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(responsePayload)
	}))
	defer mockServer.Close()

	client := services.NewMCPClient(mockServer.URL, "test-token")
	content, err := client.ReadProjectFile(context.Background(), "resume_master", "main.tex")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if content != "\\documentclass{article}\n\\begin{document}\nHello World\n\\end{document}" {
		t.Fatalf("unexpected file content: %s", content)
	}
}

func TestMCPClientWriteProjectFileSuccess(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var requestBody map[string]interface{}
		json.NewDecoder(request.Body).Decode(&requestBody)

		if requestBody["tool"] != "write_project_file" {
			http.Error(writer, "invalid tool", http.StatusBadRequest)
			return
		}

		responsePayload := map[string]interface{}{
			"success": true,
			"result": map[string]interface{}{
				"message": "Successfully wrote main.tex",
			},
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(responsePayload)
	}))
	defer mockServer.Close()

	client := services.NewMCPClient(mockServer.URL, "test-token")
	err := client.WriteProjectFile(context.Background(), "resume_master", "main.tex", "\\documentclass{article}")
	if err != nil {
		t.Fatalf("expected no error writing file, got: %v", err)
	}
}

func TestMCPClientCompileProjectSuccess(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		responsePayload := map[string]interface{}{
			"success": true,
			"result": map[string]interface{}{
				"status":    "compiled",
				"engine":    "xelatex",
				"outputLog": "Output written on main.pdf (1 page).",
				"errors":    "",
				"pdfPath":   "/projects/resume_master/main.pdf",
			},
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(responsePayload)
	}))
	defer mockServer.Close()

	client := services.NewMCPClient(mockServer.URL, "test-token")
	result, err := client.CompileProject(context.Background(), "resume_master", "xelatex", "main.tex")
	if err != nil {
		t.Fatalf("expected no compilation error, got: %v", err)
	}
	if result.Status != "compiled" || result.Engine != "xelatex" {
		t.Fatalf("unexpected compilation result: %+v", result)
	}
}

func TestMCPClientGetProjectPDFSuccess(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		responsePayload := map[string]interface{}{
			"success": true,
			"result": map[string]interface{}{
				"fileName":   "main.pdf",
				"mimeType":   "application/pdf",
				"base64Data": "JVBERi0xLjQK...",
				"sizeBytes":  1024,
			},
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(responsePayload)
	}))
	defer mockServer.Close()

	client := services.NewMCPClient(mockServer.URL, "test-token")
	pdfResult, err := client.GetProjectPDF(context.Background(), "resume_master", "main.pdf")
	if err != nil {
		t.Fatalf("expected no pdf error, got: %v", err)
	}
	if pdfResult.FileName != "main.pdf" || pdfResult.MimeType != "application/pdf" {
		t.Fatalf("unexpected pdf metadata: %+v", pdfResult)
	}
}
