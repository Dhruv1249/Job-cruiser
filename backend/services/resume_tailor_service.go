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

// ResumeTailorResult contains the generated LaTeX content along with MCP compilation metadata and page counts.
type ResumeTailorResult struct {
	ProjectName   string         `json:"projectName"`
	FilePath      string         `json:"filePath"`
	TargetPages   int            `json:"targetPages"`
	TailoredTeX   string         `json:"tailoredTeX"`
	CompileResult *CompileResult `json:"compileResult"`
	PDFResult     *PDFResult     `json:"pdfResult"`
}

// ResumeTailorService orchestrates Gemini AI prompt tailoring and MCP compilation in open-overleaf.
type ResumeTailorService struct {
	GeminiBaseURL string
	GeminiAPIKey  string
	MCPClient     *MCPClient
	HTTPClient    *http.Client
}

// NewResumeTailorService constructs a ResumeTailorService instance.
func NewResumeTailorService(geminiBaseURL string, geminiAPIKey string, mcpClient *MCPClient) *ResumeTailorService {
	if geminiBaseURL == "" {
		geminiBaseURL = "https://generativelanguage.googleapis.com"
	}
	return &ResumeTailorService{
		GeminiBaseURL: geminiBaseURL,
		GeminiAPIKey:  geminiAPIKey,
		MCPClient:     mcpClient,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// TailorResumeDirect generates tailored LaTeX content via Gemini and writes/compiles it in open-overleaf via MCP with target page enforcement.
func (service *ResumeTailorService) TailorResumeDirect(ctx context.Context, userBio string, jobDescription string, projectName string, filePath string, targetPages int) (*ResumeTailorResult, error) {
	if filePath == "" {
		filePath = "main.tex"
	}
	if targetPages <= 0 {
		targetPages = 1
	}

	initialPromptText := fmt.Sprintf(
		"You are an expert LaTeX resume tailoring engine.\n"+
			"User Experience Bank:\n%s\n\n"+
			"Target Job Description:\n%s\n\n"+
			"STRICT PAGE BUDGET: The generated resume MUST fit on exactly %d page(s). Keep spacing concise, prune lower-impact bullets, and use compact LaTeX formatting.\n"+
			"Generate a complete, syntactically valid LaTeX (.tex) resume document tailored to the job description.\n"+
			"Do not include markdown code block backticks. Output plain raw LaTeX code directly.",
		userBio,
		jobDescription,
		targetPages,
	)

	tailoredTeX, err := service.generateContentWithGemini(ctx, initialPromptText)
	if err != nil {
		return nil, fmt.Errorf("failed generating LaTeX with Gemini: %w", err)
	}

	if err := service.MCPClient.WriteProjectFile(ctx, projectName, filePath, tailoredTeX); err != nil {
		return nil, fmt.Errorf("failed writing project file to open-overleaf: %w", err)
	}

	compileResult, err := service.MCPClient.CompileProject(ctx, projectName, "xelatex", filePath)
	if err != nil {
		return nil, fmt.Errorf("failed compiling project in open-overleaf: %w", err)
	}

	if compileResult.PageCount > targetPages {
		tighteningPromptText := fmt.Sprintf(
			"The compiled LaTeX resume currently spans %d pages, which exceeds the user's strict target budget of %d page(s).\n"+
				"Current LaTeX Content:\n%s\n\n"+
				"Please prune lower-priority bullet points and adjust vertical spacing to fit the document onto exactly %d page(s).\n"+
				"Do not include markdown code block backticks. Output plain raw LaTeX code directly.",
			compileResult.PageCount,
			targetPages,
			tailoredTeX,
			targetPages,
		)

		refinedTeX, refinementError := service.generateContentWithGemini(ctx, tighteningPromptText)
		if refinementError == nil && refinedTeX != "" {
			tailoredTeX = refinedTeX
			_ = service.MCPClient.WriteProjectFile(ctx, projectName, filePath, tailoredTeX)
			secondCompileResult, secondCompileError := service.MCPClient.CompileProject(ctx, projectName, "xelatex", filePath)
			if secondCompileError == nil {
				compileResult = secondCompileResult
			}
		}
	}

	pdfResult, err := service.MCPClient.GetProjectPDF(ctx, projectName, "main.pdf")
	if err != nil {
		pdfResult = &PDFResult{
			FileName:  "main.pdf",
			MimeType:  "application/pdf",
			PageCount: compileResult.PageCount,
			SizeBytes: 0,
		}
	}

	return &ResumeTailorResult{
		ProjectName:   projectName,
		FilePath:      filePath,
		TargetPages:   targetPages,
		TailoredTeX:   tailoredTeX,
		CompileResult: compileResult,
		PDFResult:     pdfResult,
	}, nil
}

// GenerateCoverLetterDirect produces a personalized LaTeX cover letter using Gemini and compiles it via MCP.
func (service *ResumeTailorService) GenerateCoverLetterDirect(ctx context.Context, userBio string, jobDescription string, projectName string, filePath string) (*ResumeTailorResult, error) {
	if filePath == "" {
		filePath = "cover_letter.tex"
	}

	promptText := fmt.Sprintf(
		"You are an expert LaTeX cover letter tailoring engine.\n"+
			"User Profile & Bio:\n%s\n\n"+
			"Target Job Description:\n%s\n\n"+
			"STRICT PAGE BUDGET: The cover letter MUST fit on exactly 1 page.\n"+
			"Draft a compelling, highly professional LaTeX cover letter.\n"+
			"Do not include markdown code block backticks. Output plain raw LaTeX code directly.",
		userBio,
		jobDescription,
	)

	tailoredTeX, err := service.generateContentWithGemini(ctx, promptText)
	if err != nil {
		return nil, fmt.Errorf("failed generating cover letter with Gemini: %w", err)
	}

	if err := service.MCPClient.WriteProjectFile(ctx, projectName, filePath, tailoredTeX); err != nil {
		return nil, fmt.Errorf("failed writing cover letter file to open-overleaf: %w", err)
	}

	compileResult, err := service.MCPClient.CompileProject(ctx, projectName, "xelatex", filePath)
	if err != nil {
		return nil, fmt.Errorf("failed compiling cover letter in open-overleaf: %w", err)
	}

	pdfResult, err := service.MCPClient.GetProjectPDF(ctx, projectName, "cover_letter.pdf")
	if err != nil {
		pdfResult = &PDFResult{
			FileName:  "cover_letter.pdf",
			MimeType:  "application/pdf",
			PageCount: compileResult.PageCount,
			SizeBytes: 0,
		}
	}

	return &ResumeTailorResult{
		ProjectName:   projectName,
		FilePath:      filePath,
		TargetPages:   1,
		TailoredTeX:   tailoredTeX,
		CompileResult: compileResult,
		PDFResult:     pdfResult,
	}, nil
}

func (service *ResumeTailorService) generateContentWithGemini(ctx context.Context, promptText string) (string, error) {
	targetEndpoint := fmt.Sprintf("%s/v1beta/models/gemini-3.6-flash-lite:generateContent?key=%s", service.GeminiBaseURL, service.GeminiAPIKey)

	payloadMap := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{
						"text": promptText,
					},
				},
			},
		},
	}

	jsonBytes, err := json.Marshal(payloadMap)
	if err != nil {
		return "", fmt.Errorf("failed encoding gemini request: %w", err)
	}

	maxRetriesLimit := 3
	for attemptIndex := 1; attemptIndex <= maxRetriesLimit; attemptIndex++ {
		httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, targetEndpoint, bytes.NewBuffer(jsonBytes))
		if err != nil {
			return "", fmt.Errorf("failed creating gemini HTTP request: %w", err)
		}

		httpRequest.Header.Set("Content-Type", "application/json")

		httpResponse, err := service.HTTPClient.Do(httpRequest)
		if err != nil {
			return "", fmt.Errorf("gemini HTTP request failed: %w", err)
		}

		responseBytes, err := io.ReadAll(httpResponse.Body)
		httpResponse.Body.Close()
		if err != nil {
			return "", fmt.Errorf("failed reading gemini response: %w", err)
		}

		if httpResponse.StatusCode == http.StatusTooManyRequests {
			if attemptIndex < maxRetriesLimit {
				time.Sleep(30 * time.Second)
				continue
			}
			return "", fmt.Errorf("gemini rate limit exceeded after %d retries", maxRetriesLimit)
		}

		if httpResponse.StatusCode != http.StatusOK {
			return "", fmt.Errorf("gemini returned non-200 status %d: %s", httpResponse.StatusCode, string(responseBytes))
		}

		var responseEnvelope struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}

		if err := json.Unmarshal(responseBytes, &responseEnvelope); err != nil {
			return "", fmt.Errorf("failed unmarshaling gemini response envelope: %w", err)
		}

		if len(responseEnvelope.Candidates) == 0 || len(responseEnvelope.Candidates[0].Content.Parts) == 0 {
			return "", fmt.Errorf("gemini returned empty response candidate text")
		}

		return responseEnvelope.Candidates[0].Content.Parts[0].Text, nil
	}

	return "", fmt.Errorf("gemini generation failed after max retries")
}
