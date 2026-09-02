package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const defaultGeminiBaseURL = "https://generativelanguage.googleapis.com"
const defaultTargetPages = 1
const defaultMaxTighteningPasses = 3
const jobApplicationsProjectName = "job_applications"

var defaultGeminiModelsCascade = []string{
	"gemini-3.5-flash-lite",
	"gemini-3.1-flash-lite",
	"gemini-2.5-flash-lite",
}

/*
GetGeminiModelCascade parses GEMINI_MODELS or GEMINI_MODEL environment variables,
falling back to the default ordered cascade list.
*/
func GetGeminiModelCascade() []string {
	environmentModels := os.Getenv("GEMINI_MODELS")
	if environmentModels == "" {
		environmentModels = os.Getenv("GEMINI_MODEL")
	}
	if environmentModels != "" {
		var parsedModels []string
		for _, rawModel := range strings.Split(environmentModels, ",") {
			cleanedModel := strings.TrimSpace(rawModel)
			if cleanedModel != "" {
				parsedModels = append(parsedModels, cleanedModel)
			}
		}
		if len(parsedModels) > 0 {
			return parsedModels
		}
	}
	return defaultGeminiModelsCascade
}

var markdownLatexFenceRegexp = regexp.MustCompile("(?s)^\\s*`{3,}(latex|tex)?\\s*")
var markdownClosingFenceRegexp = regexp.MustCompile("(?s)\\s*`{3,}\\s*$")

/*
JobTailoringContext carries structured job metadata used to build richer Gemini prompts.
*/
type JobTailoringContext struct {
	Title     string
	Company   string
	Seniority string
	TechStack []string
	RawDesc   string
}

/*
ResumeTailorResult contains the compiled PDF artifact, MCP path references, and page count metadata.
*/
type ResumeTailorResult struct {
	ProjectName   string         `json:"projectName"`
	FolderPath    string         `json:"folderPath"`
	FilePath      string         `json:"filePath"`
	TargetPages   int            `json:"targetPages"`
	PDFWebURL     string         `json:"pdfWebURL"`
	CompileResult *CompileResult `json:"compileResult"`
	PDFResult     *PDFResult     `json:"pdfResult"`
}

/*
ResumeTailorService orchestrates Gemini AI prompt tailoring and MCP compilation in open-overleaf.
*/
type ResumeTailorService struct {
	GeminiBaseURL       string
	GeminiAPIKey        string
	GeminiModels        []string
	MaxTighteningPasses int
	HTTPClient          *http.Client
	MCPClient           *MCPClient
}

/*
NewResumeTailorService constructs a ResumeTailorService with configurable Gemini endpoint and model cascade.
*/
func NewResumeTailorService(geminiBaseURL string, geminiAPIKey string, mcpClient *MCPClient) *ResumeTailorService {
	if geminiBaseURL == "" {
		geminiBaseURL = defaultGeminiBaseURL
	}
	geminiModels := GetGeminiModelCascade()
	maxPassesEnv := os.Getenv("TAILOR_MAX_TIGHTENING_PASSES")
	maxPasses, parseError := strconv.Atoi(maxPassesEnv)
	if parseError != nil || maxPasses <= 0 {
		maxPasses = defaultMaxTighteningPasses
	}
	return &ResumeTailorService{
		GeminiBaseURL:       geminiBaseURL,
		GeminiAPIKey:        geminiAPIKey,
		GeminiModels:        geminiModels,
		MaxTighteningPasses: maxPasses,
		MCPClient:           mcpClient,
		HTTPClient: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

/*
BuildJobFolderPath creates a URL-safe folder slug combining company name, job title, and current date.
*/
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

func sanitizeGeminiLatex(rawOutput string) string {
	cleaned := markdownLatexFenceRegexp.ReplaceAllString(rawOutput, "")
	cleaned = markdownClosingFenceRegexp.ReplaceAllString(cleaned, "")
	return strings.TrimSpace(cleaned)
}

/*
TailorResumeToFolder generates tailored LaTeX using Gemini based on the job context and user bio,
saving to the specified folder path with default template fallback.
*/
func (service *ResumeTailorService) TailorResumeToFolder(
	ctx context.Context,
	mcpClient *MCPClient,
	userBio string,
	jobContext JobTailoringContext,
	folderPath string,
	projectName string,
	targetPages int,
) (*ResumeTailorResult, error) {
	return service.TailorResumeToFolderWithTemplate(ctx, mcpClient, userBio, jobContext, folderPath, projectName, targetPages, defaultResumeTemplatePath)
}

/*
TailorResumeToFolderWithTemplate reads the specified baseline resume template from Open-Overleaf,
prompts Gemini to preserve the template's layout and custom macros while tailoring content to the JD,
syncs auxiliary files, writes to {projectName}/{folderPath}/resume.tex, compiles with xelatex,
and returns the compiled PDF result.
*/
func (service *ResumeTailorService) TailorResumeToFolderWithTemplate(
	ctx context.Context,
	mcpClient *MCPClient,
	userBio string,
	jobContext JobTailoringContext,
	folderPath string,
	projectName string,
	targetPages int,
	templatePath string,
) (*ResumeTailorResult, error) {
	if targetPages <= 0 {
		targetPages = defaultTargetPages
	}
	effectiveMCPClient := mcpClient
	if effectiveMCPClient == nil {
		effectiveMCPClient = service.MCPClient
	}
	effectiveProjectName := strings.TrimSpace(projectName)
	if effectiveProjectName == "" {
		effectiveProjectName = jobApplicationsProjectName
	}

	effectiveTemplatePath := strings.TrimSpace(templatePath)
	if effectiveTemplatePath == "" {
		effectiveTemplatePath = defaultResumeTemplatePath
	}

	_ = EnsureDefaultTemplatesExist(ctx, effectiveMCPClient, effectiveProjectName)

	templateContent, templateReadError := effectiveMCPClient.ReadProjectFile(ctx, effectiveProjectName, effectiveTemplatePath)
	if templateReadError != nil || strings.TrimSpace(templateContent) == "" {
		templateContent = GetDefaultResumeTemplate()
		_ = effectiveMCPClient.WriteProjectFile(ctx, effectiveProjectName, effectiveTemplatePath, templateContent)
	}

	templateDirectory := path.Dir(effectiveTemplatePath)
	if templateDirectory != "." && templateDirectory != "/" && templateDirectory != "" {
		_ = SyncAuxiliaryTemplateFiles(ctx, effectiveMCPClient, effectiveProjectName, templateDirectory, folderPath)
	}

	techStackList := strings.Join(jobContext.TechStack, ", ")

	systemInstruction := fmt.Sprintf(
		"You are an expert LaTeX resume tailoring engine. Your task is to craft a dense, single-page, ATS-compliant LaTeX resume tailored specifically to the target job description, derived exclusively from the candidate's authentic background.\n\n"+
			"CORE DIRECTIVES:\n"+
			"1. OUTPUT FORMAT: Output ONLY valid compilable LaTeX code. Do NOT output markdown code fences (no ```latex or ```), no explanations, and no commentary. The output must start directly with \\documentclass and end with \\end{document}.\n"+
			"2. BASELINE TEMPLATE USAGE: The baseline LaTeX template provided in user content is a FORMAT AND MACRO SKELETON ONLY. You MUST adopt its preamble, package imports, geometry margins, color definitions, and custom structural macros (\\resumeSubheading, \\resumeItem, \\resumeProjectHeading).\n"+
			"3. PURGE ALL DUMMY TEMPLATE DATA: All text inside \\begin{document}...\\end{document} in the baseline template (such as 'Candidate Name', 'Acme Cloud Technologies', 'Nexus Systems', 'Distributed Task Queue Engine', dummy universities, sample bullet points, and dummy dates) is PLACEHOLDER MOCK DATA. You MUST COMPLETELY PURGE AND REPLACE all dummy template entries with the candidate's authentic information from the CANDIDATE EXPERIENCE BANK. NEVER retain any dummy companies, projects, or placeholder names from the template.\n"+
			"4. CONTACT & PROFILE LINKS: Render the candidate's real name, email, phone, location, LinkedIn, GitHub, portfolio, and any other provided profile links prominently in the header. Never use 'Candidate Name' or placeholder links.\n"+
			"5. PROFESSIONAL SUMMARY: Write an impactful 2-3 sentence summary tailored specifically for the %s role at %s, synthesizing the candidate's genuine technical strengths and domain expertise from their experience bank to address the core requirements in the JOB DESCRIPTION.\n"+
			"6. TECHNICAL SKILLS: Populate the skills section exclusively with the candidate's actual languages, frameworks, cloud technologies, databases, and developer tools extracted from their experience bank. Group them cleanly and prioritize skills that match the target role (%s).\n"+
			"7. WORK EXPERIENCE: Render the candidate's real work history (company names, titles, employment dates, locations) from the experience bank using \\resumeSubheading. Write strong, tailored \\resumeItem bullets starting with assertive action verbs that emphasize technical accomplishments, architecture, scalability, latency, database optimizations, and tooling matching the JD's requirements. NEVER retain placeholder companies or generic template bullets.\n"+
			"8. PROJECTS: Highlight 2-3 of the candidate's real projects from their experience bank that best demonstrate relevant tech stack proficiencies and problem-solving. Render using \\resumeProjectHeading with real project names, technologies used, and GitHub/live links. Detail what the candidate actually built and the technical impact. NEVER copy dummy template projects.\n"+
			"9. EDUCATION & CERTIFICATIONS: Render the candidate's real degree, university/institution, graduation year, and academic achievements using \\resumeSubheading. Do not use dummy universities.\n"+
			"10. NO INVENTED EXPERIENCE: NEVER invent fake employers, fake job titles, or unearned degrees. However, deeply expand upon the technical execution of the candidate's genuine projects and responsibilities (architecture, concurrency, APIs, performance, data pipelines) to create a dense, impressive, fully-filled resume.\n"+
			"11. PAGE BUDGET: Exactly %d page(s). Ensure the resume fills the target page budget completely from top to bottom with zero awkward whitespace gaps at the bottom, without spilling onto an extra page.\n"+
			"12. STANDARD PACKAGES ONLY: Use standard TeX Live packages only: geometry, hyperref, titlesec, enumitem, tabularx, array, xcolor.\n"+
			"13. ESCAPE SPECIAL CHARACTERS: ALWAYS properly escape special characters in text, company names, titles, and links: use \\& for &, \\%% for %%, \\_ for _, \\# for #, \\$ for $.\n"+
			"14. Output MUST begin with \\documentclass and end with \\end{document}.",
		jobContext.Title,
		jobContext.Company,
		techStackList,
		targetPages,
	)

	contentText := fmt.Sprintf(
		"=== BASELINE LATEX TEMPLATE (FORMAT & MACRO BLUEPRINT ONLY) ===\n%s\n=== END BASELINE TEMPLATE ===\n\n"+
			"TARGET ROLE\n"+
			"  Title:     %s\n"+
			"  Company:   %s\n"+
			"  Seniority: %s\n"+
			"  Tech:      %s\n\n"+
			"JOB DESCRIPTION\n%s\n\n"+
			"CANDIDATE EXPERIENCE BANK (PRIMARY SOURCE OF TRUTH FOR ALL CONTENT)\n%s",
		templateContent,
		jobContext.Title,
		jobContext.Company,
		jobContext.Seniority,
		techStackList,
		jobContext.RawDesc,
		userBio,
	)

	tailoredTeX, generateError := service.generateContentWithGeminiAndSystemInstruction(ctx, systemInstruction, contentText)
	if generateError != nil {
		return nil, fmt.Errorf("failed generating LaTeX with Gemini: %w", generateError)
	}
	tailoredTeX = sanitizeGeminiLatex(tailoredTeX)

	texFilePath := fmt.Sprintf("%s/resume.tex", folderPath)
	pdfFileName := fmt.Sprintf("%s/resume.pdf", folderPath)

	if writeError := effectiveMCPClient.WriteProjectFile(ctx, effectiveProjectName, texFilePath, tailoredTeX); writeError != nil {
		return nil, fmt.Errorf("failed writing resume.tex to open-overleaf: %w", writeError)
	}

	compileResult, compileError := effectiveMCPClient.CompileProject(ctx, effectiveProjectName, "xelatex", texFilePath)

	const maxHealingAttempts = 4
	for healPass := 1; healPass <= maxHealingAttempts && (compileError != nil || (compileResult != nil && compileResult.Status == "failed")); healPass++ {
		errorLog := ""
		if compileResult != nil {
			if compileResult.OutputLog != "" {
				errorLog = compileResult.OutputLog
			} else if compileResult.Errors != "" {
				errorLog = compileResult.Errors
			}
		}
		if errorLog == "" && compileError != nil {
			errorLog = compileError.Error()
		}

		healingPrompt := fmt.Sprintf(
			"You are an expert LaTeX debugging engine. The LaTeX resume document below failed to compile with the following compiler error log:\n\n"+
				"--- COMPILER ERROR LOG ---\n%s\n--- END ERROR LOG ---\n\n"+
				"--- FAILED LATEX SOURCE ---\n%s\n--- END LATEX SOURCE ---\n\n"+
				"DEBUGGING & FIXING INSTRUCTIONS:\n"+
				"1. Correct all syntax errors, undefined macros, and environment mismatches shown in the log while strictly preserving the template's design.\n"+
				"2. Escape all special characters in text: use \\& for &, \\%% for %%, \\_ for _, \\# for #, and \\$ for $.\n"+
				"3. Use standard TeX Live packages only: geometry, hyperref, titlesec, enumitem, tabularx, array, xcolor.\n"+
				"4. Output ONLY the complete, corrected, compilable LaTeX code with no markdown fences or commentary.",
			errorLog,
			tailoredTeX,
		)
		healedTeX, healErr := service.generateContentWithGemini(ctx, healingPrompt)
		if healErr != nil {
			break
		}
		healedTeX = sanitizeGeminiLatex(healedTeX)
		if writeErr := effectiveMCPClient.WriteProjectFile(ctx, effectiveProjectName, texFilePath, healedTeX); writeErr != nil {
			break
		}
		newResult, newErr := effectiveMCPClient.CompileProject(ctx, effectiveProjectName, "xelatex", texFilePath)
		tailoredTeX = healedTeX
		compileResult = newResult
		compileError = newErr
	}

	if compileError != nil {
		return nil, fmt.Errorf("failed compiling resume in open-overleaf after healing attempts: %w", compileError)
	}
	if compileResult != nil && compileResult.Status == "failed" {
		return nil, fmt.Errorf("failed compiling resume in open-overleaf: %s", compileResult.Errors)
	}

	for passIndex := 1; passIndex <= service.MaxTighteningPasses && compileResult.PageCount > targetPages; passIndex++ {
		tighteningPrompt := fmt.Sprintf(
			"The compiled LaTeX resume spans %d pages, exceeding the strict target of %d page(s).\n"+
				"Tightening pass %d of %d — be progressively more aggressive: prune lower-impact bullets, reduce spacing, condense sections, while preserving custom macros and structural formatting.\n\n"+
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
		if writeError := effectiveMCPClient.WriteProjectFile(ctx, effectiveProjectName, texFilePath, refinedTeX); writeError != nil {
			break
		}
		newCompileResult, newCompileError := effectiveMCPClient.CompileProject(ctx, effectiveProjectName, "xelatex", texFilePath)
		if newCompileError != nil {
			break
		}
		tailoredTeX = refinedTeX
		compileResult = newCompileResult
	}

	pdfResult, pdfError := effectiveMCPClient.GetProjectPDF(ctx, effectiveProjectName, pdfFileName)
	if pdfError != nil {
		pdfResult = &PDFResult{
			FileName:  pdfFileName,
			MimeType:  "application/pdf",
			PageCount: compileResult.PageCount,
			SizeBytes: 0,
		}
	}

	return &ResumeTailorResult{
		ProjectName:   effectiveProjectName,
		FolderPath:    folderPath,
		FilePath:      texFilePath,
		TargetPages:   targetPages,
		CompileResult: compileResult,
		PDFResult:     pdfResult,
	}, nil
}

/*
GenerateCoverLetterToFolder generates a personalized LaTeX cover letter using default template fallback.
*/
func (service *ResumeTailorService) GenerateCoverLetterToFolder(
	ctx context.Context,
	mcpClient *MCPClient,
	userBio string,
	jobContext JobTailoringContext,
	folderPath string,
	projectName string,
	targetPages int,
) (*ResumeTailorResult, error) {
	return service.GenerateCoverLetterToFolderWithTemplate(ctx, mcpClient, userBio, jobContext, folderPath, projectName, targetPages, defaultCoverLetterTemplatePath)
}

/*
GenerateCoverLetterToFolderWithTemplate reads the specified baseline cover letter template from Open-Overleaf,
prompts Gemini to preserve layout while customizing the letter body to the JD, writes and compiles via MCP.
*/
func (service *ResumeTailorService) GenerateCoverLetterToFolderWithTemplate(
	ctx context.Context,
	mcpClient *MCPClient,
	userBio string,
	jobContext JobTailoringContext,
	folderPath string,
	projectName string,
	targetPages int,
	templatePath string,
) (*ResumeTailorResult, error) {
	if targetPages <= 0 {
		targetPages = defaultTargetPages
	}
	effectiveMCPClient := mcpClient
	if effectiveMCPClient == nil {
		effectiveMCPClient = service.MCPClient
	}
	effectiveProjectName := strings.TrimSpace(projectName)
	if effectiveProjectName == "" {
		effectiveProjectName = jobApplicationsProjectName
	}

	effectiveTemplatePath := strings.TrimSpace(templatePath)
	if effectiveTemplatePath == "" {
		effectiveTemplatePath = defaultCoverLetterTemplatePath
	}

	_ = EnsureDefaultTemplatesExist(ctx, effectiveMCPClient, effectiveProjectName)

	templateContent, templateReadError := effectiveMCPClient.ReadProjectFile(ctx, effectiveProjectName, effectiveTemplatePath)
	if templateReadError != nil || strings.TrimSpace(templateContent) == "" {
		templateContent = GetDefaultCoverLetterTemplate()
		_ = effectiveMCPClient.WriteProjectFile(ctx, effectiveProjectName, effectiveTemplatePath, templateContent)
	}

	templateDirectory := path.Dir(effectiveTemplatePath)
	if templateDirectory != "." && templateDirectory != "/" && templateDirectory != "" {
		_ = SyncAuxiliaryTemplateFiles(ctx, effectiveMCPClient, effectiveProjectName, templateDirectory, folderPath)
	}

	techStackList := strings.Join(jobContext.TechStack, ", ")

	systemInstruction := fmt.Sprintf(
		"You are an expert LaTeX cover letter writer. Output ONLY valid compilable LaTeX code without markdown fences, commentary, or conversational filler. Output MUST begin with \\documentclass and end with \\end{document}.\n\n"+
			"CORE DIRECTIVES:\n"+
			"1. SKELETON BLUEPRINT ONLY: The baseline template provided in user content is ONLY a styling blueprint (preamble, geometry, and layout). The text in the template (such as 'Candidate Name', 'Target Company', 'Software Engineer role at Target Company', and the generic sample paragraphs) is DUMMY PLACEHOLDER TEXT. You MUST COMPLETELY REPLACE all dummy text with a fully customized, professional cover letter tailored specifically to the candidate and the target role.\n"+
			"2. RECIPIENT & COMPANY: Address the letter specifically to '%s' (Company) and reference the '%s' (Title) role. NEVER output 'Target Company' or placeholder names.\n"+
			"3. HEADER & SIGNATURE: Render the candidate's real name, email, phone, location, and links from the candidate profile in both the top header and closing signature. Never use 'Candidate Name' or placeholder links.\n"+
			"4. ORIGINAL PERSUASIVE LETTER BODY: Write 3-4 cohesive, compelling, beautifully phrased paragraphs:\n"+
			"   - Opening: State enthusiastic interest in the %s position at %s. Summarize who the candidate is and why their unique background makes them an exceptional match.\n"+
			"   - Technical Alignment: Detail 2-3 specific real projects, technologies, and achievements from the candidate's background that directly solve the requirements and tech stack (%s) described in the JOB DESCRIPTION. Explain what the candidate built and the tangible technical impact.\n"+
			"   - Mission Alignment: Articulate genuine appreciation for %s's engineering mission and domain based on the JD, explaining how the candidate will hit the ground running.\n"+
			"   - Professional Closing: Reiterate value proposition, express eagerness to discuss technical contributions, and provide a polite call to action.\n"+
			"5. TRUTHFULNESS: Every technical claim and project must be grounded in the candidate's actual background. Do not invent companies or fake credentials.\n"+
			"6. PAGE BUDGET: Exactly %d page(s). Neatly balanced and full without overflowing onto a second page.\n"+
			"7. STANDARD PACKAGES & ESCAPING: Use standard TeX Live packages only. ALWAYS properly escape special characters (\\&, \\%%, \\_, \\#, \\$).",
		jobContext.Company,
		jobContext.Title,
		jobContext.Title,
		jobContext.Company,
		techStackList,
		jobContext.Company,
		targetPages,
	)

	contentText := fmt.Sprintf(
		"=== BASELINE LATEX COVER LETTER TEMPLATE (FORMAT BLUEPRINT ONLY) ===\n%s\n=== END BASELINE TEMPLATE ===\n\n"+
			"TARGET ROLE\n"+
			"  Title:     %s\n"+
			"  Company:   %s\n"+
			"  Seniority: %s\n"+
			"  Tech:      %s\n\n"+
			"JOB DESCRIPTION\n%s\n\n"+
			"CANDIDATE BIO & EXPERIENCE (SOURCE OF TRUTH FOR ALL CONTENT)\n%s",
		templateContent,
		jobContext.Title,
		jobContext.Company,
		jobContext.Seniority,
		techStackList,
		jobContext.RawDesc,
		userBio,
	)

	coverLetterTeX, generateError := service.generateContentWithGeminiAndSystemInstruction(ctx, systemInstruction, contentText)
	if generateError != nil {
		return nil, fmt.Errorf("failed generating cover letter LaTeX with Gemini: %w", generateError)
	}
	coverLetterTeX = sanitizeGeminiLatex(coverLetterTeX)

	texFilePath := fmt.Sprintf("%s/cover_letter.tex", folderPath)
	pdfFileName := fmt.Sprintf("%s/cover_letter.pdf", folderPath)

	if writeError := effectiveMCPClient.WriteProjectFile(ctx, effectiveProjectName, texFilePath, coverLetterTeX); writeError != nil {
		return nil, fmt.Errorf("failed writing cover_letter.tex to open-overleaf: %w", writeError)
	}

	compileResult, compileError := effectiveMCPClient.CompileProject(ctx, effectiveProjectName, "xelatex", texFilePath)

	const maxHealingAttempts = 4
	for healPass := 1; healPass <= maxHealingAttempts && (compileError != nil || (compileResult != nil && compileResult.Status == "failed")); healPass++ {
		errorLog := ""
		if compileResult != nil {
			if compileResult.OutputLog != "" {
				errorLog = compileResult.OutputLog
			} else if compileResult.Errors != "" {
				errorLog = compileResult.Errors
			}
		}
		if errorLog == "" && compileError != nil {
			errorLog = compileError.Error()
		}

		healingPrompt := fmt.Sprintf(
			"You are an expert LaTeX debugging engine. The LaTeX cover letter document below failed to compile with the following compiler error log:\n\n"+
				"--- COMPILER ERROR LOG ---\n%s\n--- END ERROR LOG ---\n\n"+
				"--- FAILED LATEX SOURCE ---\n%s\n--- END LATEX SOURCE ---\n\n"+
				"DEBUGGING & FIXING INSTRUCTIONS:\n"+
				"1. Correct all syntax errors, undefined macros, and environment mismatches shown in the log while preserving the template layout.\n"+
				"2. Escape all special characters in text: use \\& for &, \\%% for %%, \\_ for _, \\# for #, and \\$ for $.\n"+
				"3. Use standard TeX Live packages only: geometry, hyperref, titlesec, enumitem, tabularx, array, xcolor.\n"+
				"4. Output ONLY the complete, corrected, compilable LaTeX code with no markdown fences or commentary.",
			errorLog,
			coverLetterTeX,
		)
		healedTeX, healErr := service.generateContentWithGemini(ctx, healingPrompt)
		if healErr != nil {
			break
		}
		healedTeX = sanitizeGeminiLatex(healedTeX)
		if writeErr := effectiveMCPClient.WriteProjectFile(ctx, effectiveProjectName, texFilePath, healedTeX); writeErr != nil {
			break
		}
		newResult, newErr := effectiveMCPClient.CompileProject(ctx, effectiveProjectName, "xelatex", texFilePath)
		coverLetterTeX = healedTeX
		compileResult = newResult
		compileError = newErr
	}

	if compileError != nil {
		return nil, fmt.Errorf("failed compiling cover letter in open-overleaf after healing attempts: %w", compileError)
	}
	if compileResult != nil && compileResult.Status == "failed" {
		return nil, fmt.Errorf("failed compiling cover letter in open-overleaf: %s", compileResult.Errors)
	}

	for passIndex := 1; passIndex <= service.MaxTighteningPasses && compileResult.PageCount > targetPages; passIndex++ {
		tighteningPrompt := fmt.Sprintf(
			"The compiled LaTeX cover letter spans %d pages, exceeding the strict target of %d page(s).\n"+
				"Tightening pass %d of %d: reduce paragraph lengths, adjust spacing/margins to fit neatly on %d page(s) while preserving template structure.\n\n"+
				"Current LaTeX:\n%s\n\n"+
				"Output ONLY the revised LaTeX — no markdown fences, no commentary.",
			compileResult.PageCount,
			targetPages,
			passIndex,
			service.MaxTighteningPasses,
			targetPages,
			coverLetterTeX,
		)
		refinedTeX, refinementError := service.generateContentWithGemini(ctx, tighteningPrompt)
		if refinementError != nil {
			break
		}
		refinedTeX = sanitizeGeminiLatex(refinedTeX)
		if writeError := effectiveMCPClient.WriteProjectFile(ctx, effectiveProjectName, texFilePath, refinedTeX); writeError != nil {
			break
		}
		newCompileResult, newCompileError := effectiveMCPClient.CompileProject(ctx, effectiveProjectName, "xelatex", texFilePath)
		if newCompileError != nil {
			break
		}
		coverLetterTeX = refinedTeX
		compileResult = newCompileResult
	}

	pdfResult, pdfError := effectiveMCPClient.GetProjectPDF(ctx, effectiveProjectName, pdfFileName)
	if pdfError != nil {
		pdfResult = &PDFResult{
			FileName:  pdfFileName,
			MimeType:  "application/pdf",
			PageCount: compileResult.PageCount,
			SizeBytes: 0,
		}
	}

	return &ResumeTailorResult{
		ProjectName:   effectiveProjectName,
		FolderPath:    folderPath,
		FilePath:      texFilePath,
		TargetPages:   targetPages,
		CompileResult: compileResult,
		PDFResult:     pdfResult,
	}, nil
}

/*
TailorResumeDirect is kept for backward compatibility with existing tests.
New code should use TailorResumeToFolder with per-user MCPClient.
*/
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
		"You are an expert LaTeX resume tailoring engine. Output ONLY valid LaTeX without markdown fences or explanations.\n\n"+
			"Candidate Information (Source of Truth):\n%s\n\nJob Description:\n%s\n\n"+
			"STRICT INSTRUCTIONS: Replace any dummy placeholders with the candidate's real name, authentic projects, real employment history, and actual technical skills from the candidate information. Never invent fake companies or use placeholder names. Fill the target budget of %d page(s) densely with technical depth. Output MUST begin with \\documentclass.",
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

/*
GenerateCoverLetterDirect is kept for backward compatibility with existing tests.
*/
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
		"You are an expert LaTeX cover letter writer. Output ONLY valid LaTeX without markdown fences.\n\n"+
			"Candidate Information (Source of Truth):\n%s\n\nJob Description:\n%s\n\n"+
			"STRICT INSTRUCTIONS: Replace all placeholders with candidate's real name and authentic achievements. Write an original, persuasive letter connecting candidate's real experience to the job requirements. Exactly 1 page. Output MUST begin with \\documentclass.",
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
	return service.generateContentWithGeminiAndSystemInstruction(ctx, "", promptText)
}

func (service *ResumeTailorService) generateContentWithGeminiAndSystemInstruction(ctx context.Context, systemInstruction string, contentText string) (string, error) {
	payloadMap := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": contentText},
				},
			},
		},
	}

	if strings.TrimSpace(systemInstruction) != "" {
		payloadMap["system_instruction"] = map[string]interface{}{
			"parts": []map[string]string{
				{"text": systemInstruction},
			},
		}
	}

	jsonBytes, marshalError := json.Marshal(payloadMap)
	if marshalError != nil {
		return "", fmt.Errorf("failed encoding gemini request: %w", marshalError)
	}

	modelsToTry := service.GeminiModels
	if len(modelsToTry) == 0 {
		modelsToTry = defaultGeminiModelsCascade
	}

	var lastError error

	for modelIndex, modelName := range modelsToTry {
		targetEndpoint := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s",
			service.GeminiBaseURL,
			modelName,
			service.GeminiAPIKey,
		)

		httpRequest, requestError := http.NewRequestWithContext(ctx, http.MethodPost, targetEndpoint, bytes.NewBuffer(jsonBytes))
		if requestError != nil {
			lastError = fmt.Errorf("failed creating gemini HTTP request for model %s: %w", modelName, requestError)
			continue
		}
		httpRequest.Header.Set("Content-Type", "application/json")

		httpResponse, responseError := service.HTTPClient.Do(httpRequest)
		if responseError != nil {
			lastError = fmt.Errorf("gemini HTTP request failed for model %s: %w", modelName, responseError)
			continue
		}

		responseBytes, readError := io.ReadAll(httpResponse.Body)
		httpResponse.Body.Close()
		if readError != nil {
			lastError = fmt.Errorf("failed reading gemini response for model %s: %w", modelName, readError)
			continue
		}

		if httpResponse.StatusCode == http.StatusTooManyRequests {
			lastError = fmt.Errorf("gemini model %s returned 429 rate limit", modelName)
			if modelIndex < len(modelsToTry)-1 {
				continue
			}
			return "", fmt.Errorf("all models in Gemini cascade returned 429 rate limit: %w", lastError)
		}

		if httpResponse.StatusCode != http.StatusOK {
			lastError = fmt.Errorf("gemini model %s returned status %d: %s", modelName, httpResponse.StatusCode, string(responseBytes))
			if modelIndex < len(modelsToTry)-1 {
				continue
			}
			return "", lastError
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
			lastError = fmt.Errorf("failed unmarshaling gemini response from model %s: %w", modelName, unmarshalError)
			continue
		}

		if len(responseEnvelope.Candidates) == 0 || len(responseEnvelope.Candidates[0].Content.Parts) == 0 {
			lastError = fmt.Errorf("gemini model %s returned empty response candidate", modelName)
			continue
		}

		return responseEnvelope.Candidates[0].Content.Parts[0].Text, nil
	}

	if lastError != nil {
		return "", fmt.Errorf("gemini generation failed across model cascade: %w", lastError)
	}
	return "", fmt.Errorf("gemini generation failed with no available models")
}
