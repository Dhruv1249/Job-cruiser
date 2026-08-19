package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const defaultGeminiBaseURL = "https://generativelanguage.googleapis.com"
const defaultGeminiModel = "gemini-2.5-flash-lite"
const defaultTargetPages = 1
const defaultMaxTighteningPasses = 3
const jobApplicationsProjectName = "job_applications"

var markdownLatexFenceRegexp = regexp.MustCompile("(?s)^\\s*`{3,}(latex|tex)?\\s*")
var markdownClosingFenceRegexp = regexp.MustCompile("(?s)\\s*`{3,}\\s*$")

// JobTailoringContext carries structured job metadata used to build richer Gemini prompts.
type JobTailoringContext struct {
	Title     string
	Company   string
	Seniority string
	TechStack []string
	RawDesc   string
}

// ResumeTailorResult contains the compiled PDF artifact, MCP path references, and page count metadata.
type ResumeTailorResult struct {
	ProjectName   string         `json:"projectName"`
	FolderPath    string         `json:"folderPath"`
	FilePath      string         `json:"filePath"`
	TargetPages   int            `json:"targetPages"`
	PDFWebURL     string         `json:"pdfWebURL"`
	CompileResult *CompileResult `json:"compileResult"`
	PDFResult     *PDFResult     `json:"pdfResult"`
}

// ResumeTailorService orchestrates Gemini AI prompt tailoring and MCP compilation in open-overleaf.
type ResumeTailorService struct {
	GeminiBaseURL       string
	GeminiAPIKey        string
	GeminiModel         string
	MaxTighteningPasses int
	HTTPClient          *http.Client
	MCPClient           *MCPClient
}

// NewResumeTailorService constructs a ResumeTailorService with configurable Gemini endpoint and model.
func NewResumeTailorService(geminiBaseURL string, geminiAPIKey string, mcpClient *MCPClient) *ResumeTailorService {
	if geminiBaseURL == "" {
		geminiBaseURL = defaultGeminiBaseURL
	}
	geminiModel := os.Getenv("GEMINI_MODEL")
	if geminiModel == "" {
		geminiModel = defaultGeminiModel
	}
	maxPassesEnv := os.Getenv("TAILOR_MAX_TIGHTENING_PASSES")
	maxPasses, parseError := strconv.Atoi(maxPassesEnv)
	if parseError != nil || maxPasses <= 0 {
		maxPasses = defaultMaxTighteningPasses
	}
	return &ResumeTailorService{
		GeminiBaseURL:       geminiBaseURL,
		GeminiAPIKey:        geminiAPIKey,
		GeminiModel:         geminiModel,
		MaxTighteningPasses: maxPasses,
		MCPClient:           mcpClient,
		HTTPClient: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

// BuildJobFolderPath creates a URL-safe folder slug combining company name, job title, and current date.
// Example: "Stripe" + "Software Engineer" → "stripe_software_engineer_20260820"
func BuildJobFolderPath(company string, title string) string {
	slugify := func(input string) string {
		lowered := strings.ToLower(input)
		var builder strings.Builder
		for _, character := range lowered {
			if unicode.IsLetter(character) || unicode.IsDigit(character) {
				builder.WriteRune(character)
			} else {
				builder.WriteRune('_')
			}
		}
		result := builder.String()
		for strings.Contains(result, "__") {
			result = strings.ReplaceAll(result, "__", "_")
		}
		return strings.Trim(result, "_")
	}
	dateSuffix := time.Now().UTC().Format("20060102")
	return fmt.Sprintf("%s_%s_%s", slugify(company), slugify(title), dateSuffix)
}

// sanitizeGeminiLatex strips markdown code fence markers that Gemini may prepend or append
// to generated LaTeX content despite explicit instructions to avoid them.
func sanitizeGeminiLatex(rawOutput string) string {
	cleaned := markdownLatexFenceRegexp.ReplaceAllString(rawOutput, "")
	cleaned = markdownClosingFenceRegexp.ReplaceAllString(cleaned, "")
	return strings.TrimSpace(cleaned)
}

// TailorResumeToFolder generates tailored LaTeX using Gemini based on the job context and user bio,
// writes it to job_applications/{folderPath}/resume.tex via the provided mcpClient,
// compiles with xelatex (up to MaxTighteningPasses retry loops to hit the page budget),
// and returns the result including base64 PDF bytes.
func (service *ResumeTailorService) TailorResumeToFolder(
	ctx context.Context,
	mcpClient *MCPClient,
	userBio string,
	jobContext JobTailoringContext,
	folderPath string,
	targetPages int,
) (*ResumeTailorResult, error) {
	if targetPages <= 0 {
		targetPages = defaultTargetPages
	}
	effectiveMCPClient := mcpClient
	if effectiveMCPClient == nil {
		effectiveMCPClient = service.MCPClient
	}

	techStackList := strings.Join(jobContext.TechStack, ", ")
	initialPrompt := fmt.Sprintf(
		"You are an expert LaTeX resume tailoring engine. Output ONLY valid compilable LaTeX — no markdown fences, no explanations, no commentary.\n\n"+
			"TARGET ROLE\n"+
			"  Title:     %s\n"+
			"  Company:   %s\n"+
			"  Seniority: %s\n"+
			"  Tech:      %s\n\n"+
			"JOB DESCRIPTION\n%s\n\n"+
			"CANDIDATE EXPERIENCE BANK\n%s\n\n"+
			"STRICT RULES\n"+
			"- Page budget: exactly %d page(s). Prune aggressively if needed.\n"+
			"- Never invent experience. Reorder, reword, and emphasize real facts only.\n"+
			"- Emphasize Tech keywords in bullets where factually accurate.\n"+
			"- Output MUST begin with \\documentclass and end with \\end{document}.",
		jobContext.Title,
		jobContext.Company,
		jobContext.Seniority,
		techStackList,
		jobContext.RawDesc,
		userBio,
		targetPages,
	)

	tailoredTeX, generateError := service.generateContentWithGemini(ctx, initialPrompt)
	if generateError != nil {
		return nil, fmt.Errorf("failed generating LaTeX with Gemini: %w", generateError)
	}
	tailoredTeX = sanitizeGeminiLatex(tailoredTeX)

	texFilePath := fmt.Sprintf("%s/resume.tex", folderPath)
	pdfFileName := fmt.Sprintf("%s/resume.pdf", folderPath)

	if writeError := effectiveMCPClient.WriteProjectFile(ctx, jobApplicationsProjectName, texFilePath, tailoredTeX); writeError != nil {
		return nil, fmt.Errorf("failed writing resume.tex to open-overleaf: %w", writeError)
	}

	compileResult, compileError := effectiveMCPClient.CompileProject(ctx, jobApplicationsProjectName, "xelatex", texFilePath)
	if compileError != nil {
		return nil, fmt.Errorf("failed compiling resume in open-overleaf: %w", compileError)
	}

	for passIndex := 1; passIndex <= service.MaxTighteningPasses && compileResult.PageCount > targetPages; passIndex++ {
		tighteningPrompt := fmt.Sprintf(
			"The compiled LaTeX resume spans %d pages, exceeding the strict target of %d page(s).\n"+
				"Tightening pass %d of %d — be progressively more aggressive: prune lower-impact bullets, reduce spacing, condense sections.\n\n"+
				"Current LaTeX:\n%s\n\n"+
				"Output ONLY the revised LaTeX — no markdown fences, no commentary.",
			compileResult.PageCount,
			targetPages,
			passIndex,
			service.MaxTighteningPasses,
			tailoredTeX,
		)
		refinedTeX, refinementError := service.generateContentWithGemini(ctx, tighteningPrompt)
		if refinementError != nil {
			break
		}
		refinedTeX = sanitizeGeminiLatex(refinedTeX)
		if writeError := effectiveMCPClient.WriteProjectFile(ctx, jobApplicationsProjectName, texFilePath, refinedTeX); writeError != nil {
			break
		}
		newCompileResult, newCompileError := effectiveMCPClient.CompileProject(ctx, jobApplicationsProjectName, "xelatex", texFilePath)
		if newCompileError != nil {
			break
		}
		tailoredTeX = refinedTeX
		compileResult = newCompileResult
	}

	pdfResult, pdfError := effectiveMCPClient.GetProjectPDF(ctx, jobApplicationsProjectName, pdfFileName)
	if pdfError != nil {
		pdfResult = &PDFResult{
			FileName:  pdfFileName,
			MimeType:  "application/pdf",
			PageCount: compileResult.PageCount,
			SizeBytes: 0,
		}
	}

	return &ResumeTailorResult{
		ProjectName:   jobApplicationsProjectName,
		FolderPath:    folderPath,
		FilePath:      texFilePath,
		TargetPages:   targetPages,
		CompileResult: compileResult,
		PDFResult:     pdfResult,
	}, nil
}

// GenerateCoverLetterToFolder generates a personalized LaTeX cover letter using Gemini based on job context
// and user bio, writes it to job_applications/{folderPath}/cover_letter.tex via mcpClient,
// compiles with xelatex, and returns the result including base64 PDF bytes.
func (service *ResumeTailorService) GenerateCoverLetterToFolder(
	ctx context.Context,
	mcpClient *MCPClient,
	userBio string,
	jobContext JobTailoringContext,
	folderPath string,
) (*ResumeTailorResult, error) {
	effectiveMCPClient := mcpClient
	if effectiveMCPClient == nil {
		effectiveMCPClient = service.MCPClient
	}

	techStackList := strings.Join(jobContext.TechStack, ", ")
	coverLetterPrompt := fmt.Sprintf(
		"You are an expert LaTeX cover letter writer. Output ONLY valid compilable LaTeX — no markdown fences, no explanations.\n\n"+
			"TARGET ROLE\n"+
			"  Title:     %s\n"+
			"  Company:   %s\n"+
			"  Seniority: %s\n"+
			"  Tech:      %s\n\n"+
			"JOB DESCRIPTION\n%s\n\n"+
			"CANDIDATE BIO\n%s\n\n"+
			"STRICT RULES\n"+
			"- Exactly 1 page. Professional tone. Address the hiring team directly.\n"+
			"- Reference specific company details and tech stack from the JD.\n"+
			"- Output MUST begin with \\documentclass and end with \\end{document}.",
		jobContext.Title,
		jobContext.Company,
		jobContext.Seniority,
		techStackList,
		jobContext.RawDesc,
		userBio,
	)

	coverLetterTeX, generateError := service.generateContentWithGemini(ctx, coverLetterPrompt)
	if generateError != nil {
		return nil, fmt.Errorf("failed generating cover letter LaTeX with Gemini: %w", generateError)
	}
	coverLetterTeX = sanitizeGeminiLatex(coverLetterTeX)

	texFilePath := fmt.Sprintf("%s/cover_letter.tex", folderPath)
	pdfFileName := fmt.Sprintf("%s/cover_letter.pdf", folderPath)

	if writeError := effectiveMCPClient.WriteProjectFile(ctx, jobApplicationsProjectName, texFilePath, coverLetterTeX); writeError != nil {
		return nil, fmt.Errorf("failed writing cover_letter.tex to open-overleaf: %w", writeError)
	}

	compileResult, compileError := effectiveMCPClient.CompileProject(ctx, jobApplicationsProjectName, "xelatex", texFilePath)
	if compileError != nil {
		return nil, fmt.Errorf("failed compiling cover letter in open-overleaf: %w", compileError)
	}

	pdfResult, pdfError := effectiveMCPClient.GetProjectPDF(ctx, jobApplicationsProjectName, pdfFileName)
	if pdfError != nil {
		pdfResult = &PDFResult{
			FileName:  pdfFileName,
			MimeType:  "application/pdf",
			PageCount: compileResult.PageCount,
			SizeBytes: 0,
		}
	}

	return &ResumeTailorResult{
		ProjectName:   jobApplicationsProjectName,
		FolderPath:    folderPath,
		FilePath:      texFilePath,
		TargetPages:   1,
		CompileResult: compileResult,
		PDFResult:     pdfResult,
	}, nil
}

// TailorResumeDirect is kept for backward compatibility with existing tests.
// New code should use TailorResumeToFolder with per-user MCPClient.
func (service *ResumeTailorService) TailorResumeDirect(
	ctx context.Context,
	userBio string,
	jobDescription string,
	projectName string,
	filePath string,
	targetPages int,
) (*ResumeTailorResult, error) {
	if filePath == "" {
		filePath = "main.tex"
	}
	if targetPages <= 0 {
		targetPages = defaultTargetPages
	}
	legacyJobContext := JobTailoringContext{
		Title:     "Software Engineer",
		Company:   "Target Company",
		Seniority: "",
		TechStack: []string{},
		RawDesc:   jobDescription,
	}
	if writeError := service.MCPClient.WriteProjectFile(ctx, projectName, filePath, ""); writeError == nil {
	}

	initialPrompt := fmt.Sprintf(
		"You are an expert LaTeX resume tailoring engine. Output ONLY valid LaTeX without markdown fences.\n\n"+
			"Candidate Bio:\n%s\n\nJob Description:\n%s\n\nPage budget: %d page(s). Output MUST begin with \\documentclass.",
		userBio, legacyJobContext.RawDesc, targetPages,
	)
	tailoredTeX, generateError := service.generateContentWithGemini(ctx, initialPrompt)
	if generateError != nil {
		return nil, fmt.Errorf("failed generating LaTeX with Gemini: %w", generateError)
	}
	tailoredTeX = sanitizeGeminiLatex(tailoredTeX)

	if writeError := service.MCPClient.WriteProjectFile(ctx, projectName, filePath, tailoredTeX); writeError != nil {
		return nil, fmt.Errorf("failed writing project file to open-overleaf: %w", writeError)
	}

	compileResult, compileError := service.MCPClient.CompileProject(ctx, projectName, "xelatex", filePath)
	if compileError != nil {
		return nil, fmt.Errorf("failed compiling project in open-overleaf: %w", compileError)
	}

	for passIndex := 1; passIndex <= service.MaxTighteningPasses && compileResult.PageCount > targetPages; passIndex++ {
		tighteningPrompt := fmt.Sprintf(
			"The resume is %d pages. Target: %d page(s). Tighten pass %d. Prune bullets.\nCurrent LaTeX:\n%s\nOutput ONLY LaTeX.",
			compileResult.PageCount, targetPages, passIndex, tailoredTeX,
		)
		refinedTeX, refinementError := service.generateContentWithGemini(ctx, tighteningPrompt)
		if refinementError == nil && refinedTeX != "" {
			refinedTeX = sanitizeGeminiLatex(refinedTeX)
			tailoredTeX = refinedTeX
			_ = service.MCPClient.WriteProjectFile(ctx, projectName, filePath, tailoredTeX)
			newCompileResult, newCompileError := service.MCPClient.CompileProject(ctx, projectName, "xelatex", filePath)
			if newCompileError == nil {
				compileResult = newCompileResult
			}
		}
	}

	pdfResult, pdfError := service.MCPClient.GetProjectPDF(ctx, projectName, "main.pdf")
	if pdfError != nil {
		pdfResult = &PDFResult{
			FileName:  "main.pdf",
			MimeType:  "application/pdf",
			PageCount: compileResult.PageCount,
			SizeBytes: 0,
		}
	}

	return &ResumeTailorResult{
		ProjectName:   projectName,
		FolderPath:    "",
		FilePath:      filePath,
		TargetPages:   targetPages,
		CompileResult: compileResult,
		PDFResult:     pdfResult,
	}, nil
}

// GenerateCoverLetterDirect is kept for backward compatibility.
func (service *ResumeTailorService) GenerateCoverLetterDirect(
	ctx context.Context,
	userBio string,
	jobDescription string,
	projectName string,
	filePath string,
) (*ResumeTailorResult, error) {
	if filePath == "" {
		filePath = "cover_letter.tex"
	}
	promptText := fmt.Sprintf(
		"You are an expert LaTeX cover letter writer. Output ONLY valid LaTeX without markdown fences.\n"+
			"User Bio:\n%s\n\nJob Description:\n%s\n\nExactly 1 page. Output MUST begin with \\documentclass.",
		userBio,
		jobDescription,
	)
	coverLetterTeX, generateError := service.generateContentWithGemini(ctx, promptText)
	if generateError != nil {
		return nil, fmt.Errorf("failed generating cover letter with Gemini: %w", generateError)
	}
	coverLetterTeX = sanitizeGeminiLatex(coverLetterTeX)

	if writeError := service.MCPClient.WriteProjectFile(ctx, projectName, filePath, coverLetterTeX); writeError != nil {
		return nil, fmt.Errorf("failed writing cover letter file to open-overleaf: %w", writeError)
	}

	compileResult, compileError := service.MCPClient.CompileProject(ctx, projectName, "xelatex", filePath)
	if compileError != nil {
		return nil, fmt.Errorf("failed compiling cover letter in open-overleaf: %w", compileError)
	}

	pdfResult, pdfError := service.MCPClient.GetProjectPDF(ctx, projectName, "cover_letter.pdf")
	if pdfError != nil {
		pdfResult = &PDFResult{
			FileName:  "cover_letter.pdf",
			MimeType:  "application/pdf",
			PageCount: compileResult.PageCount,
			SizeBytes: 0,
		}
	}

	return &ResumeTailorResult{
		ProjectName:   projectName,
		FolderPath:    "",
		FilePath:      filePath,
		TargetPages:   1,
		CompileResult: compileResult,
		PDFResult:     pdfResult,
	}, nil
}

func (service *ResumeTailorService) generateContentWithGemini(ctx context.Context, promptText string) (string, error) {
	targetEndpoint := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s",
		service.GeminiBaseURL,
		service.GeminiModel,
		service.GeminiAPIKey,
	)

	payloadMap := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": promptText},
				},
			},
		},
	}

	jsonBytes, marshalError := json.Marshal(payloadMap)
	if marshalError != nil {
		return "", fmt.Errorf("failed encoding gemini request: %w", marshalError)
	}

	const maxRetries = 3
	for attemptIndex := 1; attemptIndex <= maxRetries; attemptIndex++ {
		httpRequest, requestError := http.NewRequestWithContext(ctx, http.MethodPost, targetEndpoint, bytes.NewBuffer(jsonBytes))
		if requestError != nil {
			return "", fmt.Errorf("failed creating gemini HTTP request: %w", requestError)
		}
		httpRequest.Header.Set("Content-Type", "application/json")

		httpResponse, responseError := service.HTTPClient.Do(httpRequest)
		if responseError != nil {
			return "", fmt.Errorf("gemini HTTP request failed: %w", responseError)
		}

		responseBytes, readError := io.ReadAll(httpResponse.Body)
		httpResponse.Body.Close()
		if readError != nil {
			return "", fmt.Errorf("failed reading gemini response: %w", readError)
		}

		if httpResponse.StatusCode == http.StatusTooManyRequests {
			if attemptIndex < maxRetries {
				time.Sleep(30 * time.Second)
				continue
			}
			return "", fmt.Errorf("gemini rate limit exceeded after %d retries", maxRetries)
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

		if unmarshalError := json.Unmarshal(responseBytes, &responseEnvelope); unmarshalError != nil {
			return "", fmt.Errorf("failed unmarshaling gemini response: %w", unmarshalError)
		}

		if len(responseEnvelope.Candidates) == 0 || len(responseEnvelope.Candidates[0].Content.Parts) == 0 {
			return "", fmt.Errorf("gemini returned empty response candidate")
		}

		return responseEnvelope.Candidates[0].Content.Parts[0].Text, nil
	}
	return "", fmt.Errorf("gemini generation failed after max retries")
}
