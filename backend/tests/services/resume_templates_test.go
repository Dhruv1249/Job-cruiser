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

func TestGetDefaultResumeTemplateReturnsValidLatex(t *testing.T) {
	template := services.GetDefaultResumeTemplate()
	if !strings.Contains(template, "\\documentclass") {
		t.Fatalf("expected template to contain \\documentclass, got: %s", template)
	}
	if !strings.Contains(template, "\\end{document}") {
		t.Fatalf("expected template to contain \\end{document}, got: %s", template)
	}
	if !strings.Contains(template, "\\newcommand{\\resumeItem}") {
		t.Fatalf("expected template to define \\resumeItem macro, got: %s", template)
	}
	if !strings.Contains(template, "\\newcommand{\\resumeSubheading}") {
		t.Fatalf("expected template to define \\resumeSubheading macro, got: %s", template)
	}
}

func TestGetDefaultCoverLetterTemplateReturnsValidLatex(t *testing.T) {
	template := services.GetDefaultCoverLetterTemplate()
	if !strings.Contains(template, "\\documentclass") {
		t.Fatalf("expected template to contain \\documentclass, got: %s", template)
	}
	if !strings.Contains(template, "\\end{document}") {
		t.Fatalf("expected template to contain \\end{document}, got: %s", template)
	}
	if !strings.Contains(template, "Dear Hiring Team") && !strings.Contains(template, "Dear Hiring Manager") {
		t.Fatalf("expected template to have formal salutation, got: %s", template)
	}
}

func TestEnsureDefaultTemplatesExistWritesMissingTemplates(t *testing.T) {
	writtenFiles := make(map[string]string)
	mcpServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var requestBody map[string]interface{}
		json.NewDecoder(request.Body).Decode(&requestBody)
		tool, _ := requestBody["tool"].(string)
		arguments, _ := requestBody["arguments"].(map[string]interface{})

		switch tool {
		case "read_project_file":
			filePath := arguments["filePath"].(string)
			if content, exists := writtenFiles[filePath]; exists {
				json.NewEncoder(writer).Encode(map[string]interface{}{
					"success": true,
					"result":  map[string]interface{}{"content": content},
				})
				return
			}
			json.NewEncoder(writer).Encode(map[string]interface{}{
				"success": false,
				"error":   "File not found",
			})
		case "write_project_file":
			filePath := arguments["filePath"].(string)
			content := arguments["content"].(string)
			writtenFiles[filePath] = content
			json.NewEncoder(writer).Encode(map[string]interface{}{
				"success": true,
				"result":  map[string]interface{}{"message": "written"},
			})
		}
	}))
	defer mcpServer.Close()

	mcpClient := services.NewMCPClient(mcpServer.URL, "test-token")
	err := services.EnsureDefaultTemplatesExist(context.Background(), mcpClient, "job_applications")
	if err != nil {
		t.Fatalf("expected no error from EnsureDefaultTemplatesExist, got: %v", err)
	}

	if _, exists := writtenFiles["templates/resume.tex"]; !exists {
		t.Fatalf("expected templates/resume.tex to be created")
	}
	if _, exists := writtenFiles["templates/cover_letter.tex"]; !exists {
		t.Fatalf("expected templates/cover_letter.tex to be created")
	}
}

func TestEnsureDefaultTemplatesExistPreservesExistingTemplates(t *testing.T) {
	customResumeContent := "\\documentclass{article}\n\\begin{document}\nCustom Resume Template\n\\end{document}"
	customCoverContent := "\\documentclass{article}\n\\begin{document}\nCustom Cover Template\n\\end{document}"
	filesMap := map[string]string{
		"templates/resume.tex":       customResumeContent,
		"templates/cover_letter.tex": customCoverContent,
	}

	mcpServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var requestBody map[string]interface{}
		json.NewDecoder(request.Body).Decode(&requestBody)
		tool, _ := requestBody["tool"].(string)
		arguments, _ := requestBody["arguments"].(map[string]interface{})

		switch tool {
		case "read_project_file":
			filePath := arguments["filePath"].(string)
			if content, exists := filesMap[filePath]; exists {
				json.NewEncoder(writer).Encode(map[string]interface{}{
					"success": true,
					"result":  map[string]interface{}{"content": content},
				})
				return
			}
			json.NewEncoder(writer).Encode(map[string]interface{}{
				"success": false,
				"error":   "File not found",
			})
		case "write_project_file":
			filePath := arguments["filePath"].(string)
			content := arguments["content"].(string)
			filesMap[filePath] = content
			json.NewEncoder(writer).Encode(map[string]interface{}{
				"success": true,
				"result":  map[string]interface{}{"message": "written"},
			})
		}
	}))
	defer mcpServer.Close()

	mcpClient := services.NewMCPClient(mcpServer.URL, "test-token")
	err := services.EnsureDefaultTemplatesExist(context.Background(), mcpClient, "job_applications")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if filesMap["templates/resume.tex"] != customResumeContent {
		t.Fatalf("expected custom resume template to be preserved, but was overwritten")
	}
	if filesMap["templates/cover_letter.tex"] != customCoverContent {
		t.Fatalf("expected custom cover letter template to be preserved, but was overwritten")
	}
}

func TestSyncAuxiliaryTemplateFilesCopiesClassesAndStyles(t *testing.T) {
	templateFiles := []map[string]interface{}{
		{"name": "resume.tex", "path": "templates/resume.tex", "isDirectory": false, "sizeBytes": 100},
		{"name": "awesome-cv.cls", "path": "templates/awesome-cv.cls", "isDirectory": false, "sizeBytes": 500},
		{"name": "fontawesome.sty", "path": "templates/fontawesome.sty", "isDirectory": false, "sizeBytes": 300},
	}
	fileContents := map[string]string{
		"templates/awesome-cv.cls":  "\\ProvidesClass{awesome-cv}",
		"templates/fontawesome.sty": "\\ProvidesPackage{fontawesome}",
	}
	destinationFiles := make(map[string]string)

	mcpServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var requestBody map[string]interface{}
		json.NewDecoder(request.Body).Decode(&requestBody)
		tool, _ := requestBody["tool"].(string)
		arguments, _ := requestBody["arguments"].(map[string]interface{})

		switch tool {
		case "list_files":
			json.NewEncoder(writer).Encode(map[string]interface{}{
				"success": true,
				"result":  map[string]interface{}{"files": templateFiles},
			})
		case "read_project_file":
			filePath := arguments["filePath"].(string)
			if content, exists := fileContents[filePath]; exists {
				json.NewEncoder(writer).Encode(map[string]interface{}{
					"success": true,
					"result":  map[string]interface{}{"content": content},
				})
				return
			}
			json.NewEncoder(writer).Encode(map[string]interface{}{
				"success": false,
				"error":   "File not found",
			})
		case "write_project_file":
			filePath := arguments["filePath"].(string)
			content := arguments["content"].(string)
			destinationFiles[filePath] = content
			json.NewEncoder(writer).Encode(map[string]interface{}{
				"success": true,
				"result":  map[string]interface{}{"message": "written"},
			})
		}
	}))
	defer mcpServer.Close()

	mcpClient := services.NewMCPClient(mcpServer.URL, "test-token")
	err := services.SyncAuxiliaryTemplateFiles(context.Background(), mcpClient, "job_applications", "templates", "target_job_folder")
	if err != nil {
		t.Fatalf("expected no error syncing auxiliary files, got: %v", err)
	}

	if destinationFiles["target_job_folder/awesome-cv.cls"] != "\\ProvidesClass{awesome-cv}" {
		t.Fatalf("expected awesome-cv.cls to be copied to destination folder")
	}
	if destinationFiles["target_job_folder/fontawesome.sty"] != "\\ProvidesPackage{fontawesome}" {
		t.Fatalf("expected fontawesome.sty to be copied to destination folder")
	}
	if _, exists := destinationFiles["target_job_folder/resume.tex"]; exists {
		t.Fatalf("did not expect resume.tex template to be copied as an auxiliary file")
	}
}
