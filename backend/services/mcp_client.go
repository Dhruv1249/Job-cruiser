package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GenerateMCPToken derives SHA-256 authorization token from secret, github token hash, and repository name.
func GenerateMCPToken(secret string, ghTokenHash string, repoName string) string {
	if secret == "" {
		secret = "open_overleaf_mcp_secret"
	}
	if ghTokenHash == "" {
		sum := sha256.Sum256([]byte("default_gh_token"))
		ghTokenHash = hex.EncodeToString(sum[:])
	}
	if repoName == "" {
		repoName = "overleaf-projects"
	}
	rawCombined := fmt.Sprintf("%s:%s:%s", secret, ghTokenHash, repoName)
	hashBytes := sha256.Sum256([]byte(rawCombined))
	return hex.EncodeToString(hashBytes[:])
}

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

// FileEntryItem represents metadata for a file or sub-directory inside an open-overleaf project.
type FileEntryItem struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	IsDirectory bool   `json:"isDirectory"`
	SizeBytes   int64  `json:"sizeBytes"`
}

// ReadLinesResult holds sliced content of specific file line ranges.
type ReadLinesResult struct {
	FilePath     string `json:"filePath"`
	StartLine    int    `json:"startLine"`
	EndLine      int    `json:"endLine"`
	TotalLines   int    `json:"totalLines"`
	LinesContent string `json:"linesContent"`
}

// MCPToolResponse defines standard JSON RPC envelope returned by open-overleaf tool HTTP endpoints.
type MCPToolResponse struct {
	Success bool                   `json:"success"`
	Error   string                 `json:"error,omitempty"`
	Result  map[string]interface{} `json:"result,omitempty"`
}

// MCPClient handles programmatic RPC communication with the self-hosted open-overleaf MCP server.
type MCPClient struct {
	BaseURL     string
	Token       string
	GitHubToken string
	HTTPClient  *http.Client
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

// ListFiles retrieves recursive file tree entries inside an open-overleaf LaTeX project.
func (client *MCPClient) ListFiles(ctx context.Context, projectName string, subDir string) ([]FileEntryItem, error) {
	arguments := map[string]interface{}{
		"projectName": projectName,
		"subDir":      subDir,
	}

	responsePayload, err := client.executeTool(ctx, "list_files", arguments)
	if err != nil {
		return nil, err
	}

	marshaledBytes, err := json.Marshal(responsePayload["files"])
	if err != nil {
		return nil, fmt.Errorf("failed marshaling files list: %w", err)
	}

	var filesList []FileEntryItem
	if err := json.Unmarshal(marshaledBytes, &filesList); err != nil {
		return nil, fmt.Errorf("failed unmarshaling file entries: %w", err)
	}

	return filesList, nil
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

// ReadFileLines reads specific line ranges [startLine, endLine] from a file inside an open-overleaf project.
func (client *MCPClient) ReadFileLines(ctx context.Context, projectName string, filePath string, startLine int, endLine int) (*ReadLinesResult, error) {
	arguments := map[string]interface{}{
		"projectName": projectName,
		"filePath":    filePath,
		"startLine":   startLine,
		"endLine":     endLine,
	}

	responsePayload, err := client.executeTool(ctx, "read_file_lines", arguments)
	if err != nil {
		return nil, err
	}

	marshaledBytes, err := json.Marshal(responsePayload)
	if err != nil {
		return nil, fmt.Errorf("failed marshaling read_file_lines payload: %w", err)
	}

	var linesResult ReadLinesResult
	if err := json.Unmarshal(marshaledBytes, &linesResult); err != nil {
		return nil, fmt.Errorf("failed unmarshaling read_file_lines result: %w", err)
	}

	return &linesResult, nil
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

// DeleteFile deletes a file or directory inside an open-overleaf LaTeX project.
func (client *MCPClient) DeleteFile(ctx context.Context, projectName string, filePath string) error {
	arguments := map[string]interface{}{
		"projectName": projectName,
		"filePath":    filePath,
	}

	_, err := client.executeTool(ctx, "delete_file", arguments)
	return err
}

// SyncProject triggers local git commit and status synchronization for a project.
func (client *MCPClient) SyncProject(ctx context.Context, projectName string, commitMessage string) error {
	arguments := map[string]interface{}{
		"projectName":   projectName,
		"commitMessage": commitMessage,
	}

	_, err := client.executeTool(ctx, "sync_project", arguments)
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
	if client.GitHubToken != "" {
		if _, exists := arguments["githubToken"]; !exists {
			arguments["githubToken"] = client.GitHubToken
		}
	}

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
