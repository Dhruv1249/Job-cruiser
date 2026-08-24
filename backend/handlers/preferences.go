package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/Dhruv1249/Job-cruiser/backend/services"
	"github.com/Dhruv1249/Job-cruiser/backend/utils"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/genai"
)

/*
PreferencesHandler manages user settings, bio text, master CV, and overleaf configuration.
*/
type PreferencesHandler struct {
	DB           *pgxpool.Pool
	MatchService services.BatchMatchEvaluator
	NimService   *services.NvidiaNimService
	AESKey       []byte
	APIKey       string
}

type PreferencesRequest struct {
	FullName               string   `json:"full_name" binding:"required"`
	Phone                  string   `json:"phone"`
	Location               string   `json:"location"`
	LinkedInURL            string   `json:"linkedin_url"`
	GitHubURL              string   `json:"github_url"`
	PortfolioURL           string   `json:"portfolio_url"`
	TargetRoles            []string `json:"target_roles" binding:"required"`
	TargetIndustries       []string `json:"target_industries"`
	TargetLocations        []string `json:"target_locations"`
	WorkModels             []string `json:"work_models" binding:"required"`
	MinSalary              int      `json:"min_salary"`
	Currency               string   `json:"currency"`
	MasterCVText           string   `json:"master_cv_text"`
	BioExperienceText      string   `json:"bio_experience_text"`
	AIMatchingEnabled      bool     `json:"ai_matching_enabled"`
	TargetResumePages      int      `json:"target_resume_pages"`
	TargetCoverLetterPages int      `json:"target_cover_letter_pages"`
}

type ParseCVRequest struct {
	RawCVText string `json:"raw_cv_text" binding:"required"`
}

type ParsedExperienceItem struct {
	Company    string `json:"company"`
	Role       string `json:"role"`
	Duration   string `json:"duration"`
	Highlights string `json:"highlights"`
}

type ParsedProjectItem struct {
	Title       string   `json:"title"`
	TechStack   []string `json:"tech_stack"`
	Description string   `json:"description"`
	Link        string   `json:"link"`
}

type ParsedAchievementItem struct {
	Title   string `json:"title"`
	Details string `json:"details"`
}

type ParsedCertificationItem struct {
	Name   string `json:"name"`
	Issuer string `json:"issuer"`
}

type ParsedEducationItem struct {
	Institution string `json:"institution"`
	Degree      string `json:"degree"`
	Year        string `json:"year"`
	Grade       string `json:"grade"`
}

type ParsedCVResponse struct {
	BioSummary         string                    `json:"bio_summary"`
	Location           string                    `json:"location"`
	Skills             []string                  `json:"skills"`
	Education          []ParsedEducationItem     `json:"education"`
	Experience         []ParsedExperienceItem    `json:"experience"`
	Projects           []ParsedProjectItem       `json:"projects"`
	Achievements       []ParsedAchievementItem   `json:"achievements"`
	Certifications     []ParsedCertificationItem `json:"certifications"`
	DiscoveredKeywords []string                  `json:"discovered_keywords"`
	NewKeywords        []string                  `json:"new_keywords"`
}

/*
UpdatePreferences saves or updates a user's preferences profile.
*/
func (h *PreferencesHandler) UpdatePreferences(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req PreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	targetResumePages := req.TargetResumePages
	if targetResumePages <= 0 {
		targetResumePages = 1
	}
	targetCoverLetterPages := req.TargetCoverLetterPages
	if targetCoverLetterPages <= 0 {
		targetCoverLetterPages = 1
	}

	query := `
		INSERT INTO user_preferences (user_id, full_name, phone, location, linkedin_url, github_url, portfolio_url, target_roles, target_industries, target_locations, work_models, min_salary, currency, master_cv_text, bio_experience_text, target_resume_pages, target_cover_letter_pages)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (user_id) 
		DO UPDATE SET 
			full_name = EXCLUDED.full_name,
			phone = EXCLUDED.phone,
			location = EXCLUDED.location,
			linkedin_url = EXCLUDED.linkedin_url,
			github_url = EXCLUDED.github_url,
			portfolio_url = EXCLUDED.portfolio_url,
			target_roles = EXCLUDED.target_roles,
			target_industries = EXCLUDED.target_industries,
			target_locations = EXCLUDED.target_locations,
			work_models = EXCLUDED.work_models,
			min_salary = EXCLUDED.min_salary,
			currency = EXCLUDED.currency,
			master_cv_text = EXCLUDED.master_cv_text,
			bio_experience_text = EXCLUDED.bio_experience_text,
			target_resume_pages = EXCLUDED.target_resume_pages,
			target_cover_letter_pages = EXCLUDED.target_cover_letter_pages,
			updated_at = CURRENT_TIMESTAMP;
	`

	_, err := h.DB.Exec(context.Background(), query, userID, req.FullName, req.Phone, req.Location, req.LinkedInURL, req.GitHubURL, req.PortfolioURL, req.TargetRoles, req.TargetIndustries, req.TargetLocations, req.WorkModels, req.MinSalary, req.Currency, req.MasterCVText, req.BioExperienceText, targetResumePages, targetCoverLetterPages)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save preferences: " + err.Error()})
		return
	}

	if h.MatchService != nil && req.AIMatchingEnabled {
		go h.MatchService.EvaluateForSingleUser(context.Background(), fmt.Sprintf("%v", userID))
	}

	c.JSON(http.StatusOK, gin.H{"message": "Preferences saved successfully"})
}

/*
GetPreferences retrieves a user's settings profile.
*/
func (h *PreferencesHandler) GetPreferences(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	query := `
		SELECT 
			COALESCE(p.full_name, ''), 
			COALESCE(p.phone, u.phone, ''),
			COALESCE(p.location, u.location, ''),
			COALESCE(p.linkedin_url, u.links->>'linkedin', ''),
			COALESCE(p.github_url, u.links->>'github', ''),
			COALESCE(p.portfolio_url, u.links->>'portfolio', ''),
			COALESCE(p.target_roles, '[]'::jsonb), 
			COALESCE(p.target_industries, '[]'::jsonb), 
			COALESCE(p.target_locations, '["India (On-site & Hybrid)", "India (Remote)", "Global Remote"]'::jsonb), 
			COALESCE(p.work_models, '[]'::jsonb), 
			COALESCE(p.min_salary, 0), 
			COALESCE(p.currency, 'USD'), 
			COALESCE(p.master_cv_text, ''), 
			COALESCE(p.bio_experience_text, ''),
			COALESCE(u.ai_matching_enabled, false),
			COALESCE(p.target_resume_pages, 1),
			COALESCE(p.target_cover_letter_pages, 1),
			(p.user_id IS NOT NULL AND jsonb_array_length(COALESCE(p.target_roles, '[]'::jsonb)) > 0) AS has_preferences
		FROM users u
		LEFT JOIN user_preferences p ON u.id = p.user_id
		WHERE u.id = $1;
	`

	var pref PreferencesRequest
	var hasPreferences bool
	err := h.DB.QueryRow(context.Background(), query, userID).Scan(
		&pref.FullName, &pref.Phone, &pref.Location, &pref.LinkedInURL, &pref.GitHubURL, &pref.PortfolioURL, &pref.TargetRoles, &pref.TargetIndustries, &pref.TargetLocations, &pref.WorkModels, &pref.MinSalary, &pref.Currency, &pref.MasterCVText, &pref.BioExperienceText, &pref.AIMatchingEnabled, &pref.TargetResumePages, &pref.TargetCoverLetterPages, &hasPreferences,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch preferences: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":            pref,
		"has_preferences": hasPreferences,
	})
}

type OverleafConfigRequest struct {
	DeploymentURL           string `json:"deployment_url" binding:"required"`
	MCPSecret               string `json:"mcp_secret"`
	ProjectName             string `json:"project_name"`
	ResumeTemplatePath      string `json:"resume_template_path"`
	CoverLetterTemplatePath string `json:"cover_letter_template_path"`
}

/*
UpdateOverleafConfig saves self-hosted open-overleaf configuration for the user.
*/
func (h *PreferencesHandler) UpdateOverleafConfig(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req OverleafConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	projectName := strings.TrimSpace(req.ProjectName)
	if projectName == "" {
		projectName = "job_applications"
	}

	resumeTemplatePath := strings.TrimSpace(req.ResumeTemplatePath)
	if resumeTemplatePath == "" {
		resumeTemplatePath = "templates/resume.tex"
	}

	coverLetterTemplatePath := strings.TrimSpace(req.CoverLetterTemplatePath)
	if coverLetterTemplatePath == "" {
		coverLetterTemplatePath = "templates/cover_letter.tex"
	}

	var encryptedToken *string
	tokenEncrypted := false

	cleanSecret := strings.TrimSpace(req.MCPSecret)
	if cleanSecret == "" {
		var existingSecret *string
		_ = h.DB.QueryRow(context.Background(), `SELECT encrypted_access_token FROM user_overleaf_config WHERE user_id = $1`, userID).Scan(&existingSecret)
		if existingSecret == nil || *existingSecret == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Open-Overleaf MCP Secret / Access Token is required to secure your instance"})
			return
		}
	}

	if cleanSecret != "" {
		if len(h.AESKey) == 32 {
			encrypted, encryptError := utils.EncryptToken(cleanSecret, h.AESKey)
			if encryptError == nil {
				encryptedToken = &encrypted
				tokenEncrypted = true
			} else {
				encryptedToken = &cleanSecret
			}
		} else {
			encryptedToken = &cleanSecret
		}
	}

	query := `
		INSERT INTO user_overleaf_config (user_id, deployment_url, project_name, encrypted_access_token, token_encrypted, resume_template_path, cover_letter_template_path)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id)
		DO UPDATE SET
			deployment_url = EXCLUDED.deployment_url,
			project_name = EXCLUDED.project_name,
			resume_template_path = EXCLUDED.resume_template_path,
			cover_letter_template_path = EXCLUDED.cover_letter_template_path,
			encrypted_access_token = CASE WHEN EXCLUDED.encrypted_access_token IS NOT NULL THEN EXCLUDED.encrypted_access_token ELSE user_overleaf_config.encrypted_access_token END,
			token_encrypted = CASE WHEN EXCLUDED.encrypted_access_token IS NOT NULL THEN EXCLUDED.token_encrypted ELSE user_overleaf_config.token_encrypted END,
			updated_at = CURRENT_TIMESTAMP;
	`

	_, err := h.DB.Exec(context.Background(), query, userID, strings.TrimSpace(req.DeploymentURL), projectName, encryptedToken, tokenEncrypted, resumeTemplatePath, coverLetterTemplatePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save open-overleaf configuration"})
		return
	}

	go func(uID string) {
		client, _, clientErr := services.LoadUserMCPClient(context.Background(), h.DB, uID, h.AESKey, "")
		if clientErr == nil && client != nil {
			_ = services.EnsureDefaultTemplatesExist(context.Background(), client, projectName)
		}
	}(fmt.Sprintf("%v", userID))

	c.JSON(http.StatusOK, gin.H{"message": "Self-hosted open-overleaf configured successfully"})
}

/*
GetOverleafConfig retrieves self-hosted open-overleaf settings for the user.
*/
func (h *PreferencesHandler) GetOverleafConfig(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	query := `
		SELECT deployment_url, COALESCE(project_name, 'job_applications'), COALESCE(encrypted_access_token, ''), COALESCE(token_encrypted, false),
		       COALESCE(resume_template_path, 'templates/resume.tex'), COALESCE(cover_letter_template_path, 'templates/cover_letter.tex')
		FROM user_overleaf_config
		WHERE user_id = $1;
	`

	var url, projectName, encryptedToken, resumeTemplatePath, coverLetterTemplatePath string
	var tokenEncrypted bool
	err := h.DB.QueryRow(context.Background(), query, userID).Scan(&url, &projectName, &encryptedToken, &tokenEncrypted, &resumeTemplatePath, &coverLetterTemplatePath)
	if err != nil {
		if err.Error() == "no rows in result set" {
			c.JSON(http.StatusOK, gin.H{"data": nil})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch open-overleaf config"})
		return
	}

	secret := ""
	if encryptedToken != "" {
		if tokenEncrypted && len(h.AESKey) == 32 {
			decrypted, decErr := utils.DecryptToken(encryptedToken, h.AESKey)
			if decErr == nil {
				secret = decrypted
			} else {
				secret = encryptedToken
			}
		} else {
			secret = encryptedToken
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"deployment_url":              url,
			"project_name":                projectName,
			"has_secret":                  encryptedToken != "",
			"mcp_secret":                  secret,
			"resume_template_path":       resumeTemplatePath,
			"cover_letter_template_path": coverLetterTemplatePath,
		},
	})
}

type flexEducationItem struct {
	Institution string `json:"institution"`
	Degree      string `json:"degree"`
	Year        string `json:"year"`
	Grade       string `json:"grade"`
}

type flexExperienceItem struct {
	Company    string      `json:"company"`
	Role       string      `json:"role"`
	Duration   string      `json:"duration"`
	Highlights interface{} `json:"highlights"`
}

type flexProjectItem struct {
	Title       string      `json:"title"`
	TechStack   interface{} `json:"tech_stack"`
	Description string      `json:"description"`
	Link        string      `json:"link"`
}

type flexAchievementItem struct {
	Title   string      `json:"title"`
	Details interface{} `json:"details"`
}

type flexCertificationItem struct {
	Name   string `json:"name"`
	Issuer string `json:"issuer"`
}

type flexCVResponse struct {
	BioSummary         string                  `json:"bio_summary"`
	Location           string                  `json:"location"`
	Skills             []string                `json:"skills"`
	Education          []flexEducationItem     `json:"education"`
	Experience         []flexExperienceItem    `json:"experience"`
	Projects           []flexProjectItem       `json:"projects"`
	Achievements       []flexAchievementItem   `json:"achievements"`
	Certifications     []flexCertificationItem `json:"certifications"`
	DiscoveredKeywords []string                `json:"discovered_keywords"`
}

func stringifyFlex(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []interface{}:
		var parts []string
		for _, item := range val {
			if str, ok := item.(string); ok && strings.TrimSpace(str) != "" {
				parts = append(parts, strings.TrimSpace(str))
			}
		}
		return strings.Join(parts, "\n• ")
	default:
		return fmt.Sprintf("%v", val)
	}
}

/*
ParseCV uses Gemini AI to parse raw CV text into structured sections and flags new domain keywords for Master Admin approval.
*/
func (h *PreferencesHandler) ParseCV(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req ParseCVRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	apiKey := h.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}

	ctx := c.Request.Context()

	masterMap := make(map[string]bool)
	var masterList []string
	rows, err := h.DB.Query(ctx, "SELECT LOWER(keyword) FROM master_keywords;")
	if err == nil {
		for rows.Next() {
			var kw string
			if err := rows.Scan(&kw); err == nil {
				masterMap[kw] = true
				masterList = append(masterList, kw)
			}
		}
	}

	prompt := fmt.Sprintf(`You are an expert technical resume parser. Extract structured education, experience, projects, skills, achievements, certifications, location, a FIRST-PERSON bio summary, and technical domain keywords from the provided raw CV text.

[EXISTING MASTER KEYWORD TAXONOMY]:
%s

[RAW CV TEXT]:
%s

CRITICAL INSTRUCTION FOR BIO SUMMARY:
Write bio_summary strictly in FIRST-PERSON ("I am a Full Stack Developer specializing in..."). NEVER use third person ("Dhruv is...", "He specializes in..."). Use "I".

Return ONLY a strict JSON object matching this schema without markdown formatting or codeblocks:
{
  "bio_summary": "First-person professional summary using 'I'",
  "location": "City, Country or State",
  "skills": ["Python", "Go", "React", "Postgres"],
  "education": [
    {
      "institution": "University Name",
      "degree": "B.Tech Computer Science",
      "year": "2023 - 2026",
      "grade": "CGPA 8.66"
    }
  ],
  "experience": [
    {
      "company": "Company Name",
      "role": "Role Title",
      "duration": "2021 - Present",
      "highlights": ["Key contribution 1", "Key contribution 2"]
    }
  ],
  "projects": [
    {
      "title": "Project Name",
      "tech_stack": ["Go", "Docker"],
      "description": "Short description",
      "link": "URL or empty"
    }
  ],
  "achievements": [
    {
      "title": "Achievement Title",
      "details": "Details"
    }
  ],
  "certifications": [
    {
      "name": "Certification Name",
      "issuer": "Issuing Org"
    }
  ],
  "discovered_keywords": ["Golang", "Postgres", "Flutter", "Kubernetes"]
}`, strings.Join(masterList, ", "), req.RawCVText)

	var rawJSON string
	if h.NimService != nil {
		completionContent, errNim := h.NimService.GenerateCompletionWithSchema(ctx, prompt, "", services.CVParsingJSONSchema)
		if errNim != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "NVIDIA NIM CV parsing failed: " + errNim.Error()})
			return
		}
		rawJSON = completionContent
	} else {
		client, errClient := genai.NewClient(ctx, &genai.ClientConfig{
			Backend: genai.BackendGeminiAPI,
			APIKey:  apiKey,
		})
		if errClient != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize Gemini AI client: " + errClient.Error()})
			return
		}

		config := &genai.GenerateContentConfig{
			ResponseMIMEType: "application/json",
			Temperature:      genai.Ptr[float32](0.0),
		}

		modelsCascade := services.GetGeminiModelCascade()
		var lastGenError error
		for _, modelName := range modelsCascade {
			result, errGen := client.Models.GenerateContent(ctx, modelName, genai.Text(prompt), config)
			if errGen != nil {
				lastGenError = errGen
				continue
			}
			rawJSON = result.Text()
			break
		}

		if rawJSON == "" && lastGenError != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gemini AI CV parsing failed: " + lastGenError.Error()})
			return
		}
	}

	var flexRes flexCVResponse
	if err := json.Unmarshal([]byte(rawJSON), &flexRes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse structured JSON from AI output: " + err.Error()})
		return
	}

	var parsedResponse ParsedCVResponse
	parsedResponse.BioSummary = flexRes.BioSummary
	parsedResponse.Location = flexRes.Location
	parsedResponse.Skills = flexRes.Skills
	parsedResponse.DiscoveredKeywords = flexRes.DiscoveredKeywords

	for _, item := range flexRes.Education {
		parsedResponse.Education = append(parsedResponse.Education, ParsedEducationItem{
			Institution: item.Institution,
			Degree:      item.Degree,
			Year:        item.Year,
			Grade:       item.Grade,
		})
	}

	for _, item := range flexRes.Experience {
		parsedResponse.Experience = append(parsedResponse.Experience, ParsedExperienceItem{
			Company:    item.Company,
			Role:       item.Role,
			Duration:   item.Duration,
			Highlights: stringifyFlex(item.Highlights),
		})
	}

	for _, item := range flexRes.Projects {
		var tsList []string
		switch ts := item.TechStack.(type) {
		case string:
			tsList = []string{ts}
		case []interface{}:
			for _, t := range ts {
				if s, ok := t.(string); ok {
					tsList = append(tsList, s)
				}
			}
		}
		parsedResponse.Projects = append(parsedResponse.Projects, ParsedProjectItem{
			Title:       item.Title,
			TechStack:   tsList,
			Description: item.Description,
			Link:        item.Link,
		})
	}

	for _, item := range flexRes.Achievements {
		parsedResponse.Achievements = append(parsedResponse.Achievements, ParsedAchievementItem{
			Title:   item.Title,
			Details: stringifyFlex(item.Details),
		})
	}

	for _, item := range flexRes.Certifications {
		parsedResponse.Certifications = append(parsedResponse.Certifications, ParsedCertificationItem{
			Name:   item.Name,
			Issuer: item.Issuer,
		})
	}

	var newKeywords []string
	for _, kw := range parsedResponse.DiscoveredKeywords {
		cleanKw := strings.TrimSpace(kw)
		lowerKw := strings.ToLower(cleanKw)
		if cleanKw != "" && !masterMap[lowerKw] {
			newKeywords = append(newKeywords, cleanKw)
			h.DB.Exec(ctx, `
				INSERT INTO pending_keyword_suggestions (keyword, discovered_from_user_id)
				VALUES ($1, $2);
			`, cleanKw, userID)
		}
	}
	parsedResponse.NewKeywords = newKeywords

	parsedExpBytes, err := json.Marshal(parsedResponse)
	if err == nil {
		h.DB.Exec(ctx, "UPDATE users SET parsed_experience = $1 WHERE id = $2", string(parsedExpBytes), userID)
	}

	c.JSON(http.StatusOK, gin.H{"data": parsedResponse})
}
