package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DiagnosticError represents an individual LaTeX compilation error or warning snippet.
type DiagnosticError struct {
	Line    int    `json:"line,omitempty"`
	Message string `json:"message"`
	Snippet string `json:"snippet,omitempty"`
}

// TeXDiagnostics holds structured parsing results for LaTeX compilation stdout/stderr logs.
type TeXDiagnostics struct {
	HasErrors       bool              `json:"hasErrors"`
	Errors          []DiagnosticError `json:"errors"`
	MissingPackages []string          `json:"missingPackages"`
}

// CompileResult captures LaTeX compilation diagnostics, page counts, and output paths returned by open-overleaf.
type CompileResult struct {
	Status      string          `json:"status"`
	Engine      string          `json:"engine"`
	PageCount   int             `json:"pageCount"`
	Diagnostics *TeXDiagnostics `json:"diagnostics"`
	OutputLog   string          `json:"outputLog"`
	Errors      string          `json:"errors"`
	PDFPath     string          `json:"pdfPath"`
}

// PDFResult holds binary base64 metadata and total page counts for compiled LaTeX document artifacts.
type PDFResult struct {
	FileName   string `json:"fileName"`
	MimeType   string `json:"mimeType"`
	PageCount  int    `json:"pageCount"`
	Base64Data string `json:"base64Data"`
	SizeBytes  int64  `json:"sizeBytes"`
}

// PreviewImageResult holds base64 encoded PNG rendering of a single PDF page.
type PreviewImageResult struct {
	FileName   string `json:"fileName"`
	MimeType   string `json:"mimeType"`
	PageNumber int    `json:"pageNumber"`
	Base64Data string `json:"base64Data"`
	SizeBytes  int64  `json:"sizeBytes"`
}

// MCPToolResponse defines standard JSON RPC envelope returned by open-overleaf tool HTTP endpoints.
type MCPToolResponse struct {
	Success bool                   `json:"success"`
	Error   string                 `json:"error,omitempty"`
	Result  map[string]interface{} `json:"result,omitempty"`
}

// MCPClient handles programmatic RPC communication with the self-hosted open-overleaf MCP server.
type MCPClient struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// NewMCPClient constructs a thread-safe MCP HTTP client targeting open-overleaf.
func NewMCPClient(baseURL string, token string) *MCPClient {
	return &MCPClient{
		BaseURL: baseURL,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ListProjects queries open-overleaf for all active LaTeX project directory names.
func (client *MCPClient) ListProjects(ctx context.Context) ([]string, error) {
	responsePayload, err := client.executeTool(ctx, "list_projects", map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	rawProjectsList, exists := responsePayload["projects"]
	if !exists {
		return nil, fmt.Errorf("response missing projects array")
	}

	interfaceSlice, ok := rawProjectsList.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid projects data format")
	}

	projectNames := make([]string, 0, len(interfaceSlice))
	for _, rawItem := range interfaceSlice {
		if nameString, isString := rawItem.(string); isString {
			projectNames = append(projectNames, nameString)
		}
	}

	return projectNames, nil
}

// ReadProjectFile reads text content of a file inside an open-overleaf LaTeX project.
func (client *MCPClient) ReadProjectFile(ctx context.Context, projectName string, filePath string) (string, error) {
	arguments := map[string]interface{}{
		"projectName": projectName,
		"filePath":    filePath,
	}

	responsePayload, err := client.executeTool(ctx, "read_project_file", arguments)
	if err != nil {
		return "", err
	}

	contentString, ok := responsePayload["content"].(string)
	if !ok {
		return "", fmt.Errorf("invalid or missing file content in response")
	}

	return contentString, nil
}

// WriteProjectFile updates or creates a text file inside an open-overleaf LaTeX project.
func (client *MCPClient) WriteProjectFile(ctx context.Context, projectName string, filePath string, content string) error {
	arguments := map[string]interface{}{
		"projectName": projectName,
		"filePath":    filePath,
		"content":     content,
	}

	_, err := client.executeTool(ctx, "write_project_file", arguments)
	return err
}

// CompileProject triggers xelatex/pdflatex compilation for an open-overleaf LaTeX project.
func (client *MCPClient) CompileProject(ctx context.Context, projectName string, engine string, entryFile string) (*CompileResult, error) {
	if engine == "" {
		engine = "xelatex"
	}
	if entryFile == "" {
		entryFile = "main.tex"
	}

	arguments := map[string]interface{}{
		"projectName": projectName,
		"engine":      engine,
		"entryFile":   entryFile,
	}

	responsePayload, err := client.executeTool(ctx, "compile_project", arguments)
	if err != nil {
		return nil, err
	}

	marshaledBytes, err := json.Marshal(responsePayload)
	if err != nil {
		return nil, fmt.Errorf("failed marshaling compile payload: %w", err)
	}

	var compileResult CompileResult
	if err := json.Unmarshal(marshaledBytes, &compileResult); err != nil {
		return nil, fmt.Errorf("failed unmarshaling compile result: %w", err)
	}

	return &compileResult, nil
}

// GetProjectPDF retrieves base64 encoded binary data for a compiled PDF artifact in open-overleaf.
func (client *MCPClient) GetProjectPDF(ctx context.Context, projectName string, pdfName string) (*PDFResult, error) {
	if pdfName == "" {
		pdfName = "main.pdf"
	}

	arguments := map[string]interface{}{
		"projectName": projectName,
		"pdfName":     pdfName,
	}

	responsePayload, err := client.executeTool(ctx, "get_project_pdf", arguments)
	if err != nil {
		return nil, err
	}

	marshaledBytes, err := json.Marshal(responsePayload)
	if err != nil {
		return nil, fmt.Errorf("failed marshaling pdf payload: %w", err)
	}

	var pdfResult PDFResult
	if err := json.Unmarshal(marshaledBytes, &pdfResult); err != nil {
		return nil, fmt.Errorf("failed unmarshaling pdf result: %w", err)
	}

	return &pdfResult, nil
}

// GetProjectPreviewImage renders page N of a compiled PDF into a PNG preview image base64 string.
func (client *MCPClient) GetProjectPreviewImage(ctx context.Context, projectName string, pdfName string, pageNumber int, dpi int) (*PreviewImageResult, error) {
	if pdfName == "" {
		pdfName = "main.pdf"
	}
	if pageNumber <= 0 {
		pageNumber = 1
	}
	if dpi <= 0 {
		dpi = 150
	}

	arguments := map[string]interface{}{
		"projectName": projectName,
		"pdfName":     pdfName,
		"pageNumber":  pageNumber,
		"dpi":         dpi,
	}

	responsePayload, err := client.executeTool(ctx, "get_project_preview_image", arguments)
	if err != nil {
		return nil, err
	}

	marshaledBytes, err := json.Marshal(responsePayload)
	if err != nil {
		return nil, fmt.Errorf("failed marshaling preview payload: %w", err)
	}

	var previewResult PreviewImageResult
	if err := json.Unmarshal(marshaledBytes, &previewResult); err != nil {
		return nil, fmt.Errorf("failed unmarshaling preview result: %w", err)
	}

	return &previewResult, nil
}

func (client *MCPClient) executeTool(ctx context.Context, toolName string, arguments map[string]interface{}) (map[string]interface{}, error) {
	requestPayload := map[string]interface{}{
		"tool":      toolName,
		"arguments": arguments,
	}

	jsonBytes, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("failed encoding MCP tool request: %w", err)
	}

	targetURL := fmt.Sprintf("%s/api/mcp/tool", client.BaseURL)
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("failed creating HTTP request: %w", err)
	}

	httpRequest.Header.Set("Content-Type", "application/json")
	if client.Token != "" {
		httpRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.Token))
	}

	httpResponse, err := client.HTTPClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("MCP request to %s failed: %w", targetURL, err)
	}
	defer httpResponse.Body.Close()

	bodyBytes, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading MCP response body: %w", err)
	}

	if httpResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MCP server returned status %d: %s", httpResponse.StatusCode, string(bodyBytes))
	}

	var toolEnvelope MCPToolResponse
	if err := json.Unmarshal(bodyBytes, &toolEnvelope); err != nil {
		return nil, fmt.Errorf("failed parsing MCP envelope JSON: %w", err)
	}

	if !toolEnvelope.Success {
		return nil, fmt.Errorf("MCP tool error: %s", toolEnvelope.Error)
	}

	return toolEnvelope.Result, nil
}
