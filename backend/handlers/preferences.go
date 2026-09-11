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

type CustomLinkItem struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type PreferencesRequest struct {
	FullName                          string                    `json:"full_name" binding:"required"`
	Email                             string                    `json:"email"`
	Phone                             string                    `json:"phone"`
	Location                          string                    `json:"location"`
	Country                           string                    `json:"country"`
	LinkedInURL                       string                    `json:"linkedin_url"`
	GitHubURL                         string                    `json:"github_url"`
	PortfolioURL                      string                    `json:"portfolio_url"`
	CustomLinks                       []CustomLinkItem          `json:"custom_links"`
	TargetRoles                       []string                  `json:"target_roles"`
	TargetIndustries                  []string                  `json:"target_industries"`
	TargetLocations                   []string                  `json:"target_locations"`
	WorkModels                        []string                  `json:"work_models"`
	MinSalary                         int                       `json:"min_salary"`
	Currency                          string                    `json:"currency"`
	MasterCVText                      string                    `json:"master_cv_text"`
	BioExperienceText                 string                    `json:"bio_experience_text"`
	BioSummary                        string                    `json:"bio_summary"`
	AIMatchingEnabled                 bool                      `json:"ai_matching_enabled"`
	TargetResumePages                 int                       `json:"target_resume_pages"`
	TargetCoverLetterPages            int                       `json:"target_cover_letter_pages"`
	MatchThresholdNotificationEnabled bool                      `json:"match_threshold_notification_enabled"`
	MatchThresholdPercentage          int                       `json:"match_threshold_percentage"`
	Experiences                       []ParsedExperienceItem     `json:"experiences"`
	Projects                          []ParsedProjectItem        `json:"projects"`
	Education                         []ParsedEducationItem      `json:"education"`
	Skills                            []string                   `json:"skills"`
	Achievements                      []ParsedAchievementItem    `json:"achievements"`
	Certifications                    []ParsedCertificationItem  `json:"certifications"`
	ResearchPatents                   []ParsedResearchPatentItem `json:"research_patents"`
	OpenSourceContributions           []ParsedOpenSourceItem     `json:"open_source_contributions"`
	CustomFormAnswers                 map[string]any             `json:"custom_form_answers"`
}

/*
ProfileUpdateRequest encapsulates personal background, contact information, social links, and structured resume items.
*/
type ProfileUpdateRequest struct {
	FullName                string                     `json:"full_name" binding:"required"`
	Email                   string                     `json:"email"`
	Phone                   string                     `json:"phone"`
	Location                string                     `json:"location"`
	Country                 string                     `json:"country"`
	LinkedInURL             string                     `json:"linkedin_url"`
	GitHubURL               string                     `json:"github_url"`
	PortfolioURL            string                     `json:"portfolio_url"`
	CustomLinks             []CustomLinkItem           `json:"custom_links"`
	BioSummary              string                     `json:"bio_summary"`
	BioExperienceText       string                     `json:"bio_experience_text"`
	Experiences             []ParsedExperienceItem     `json:"experiences"`
	Projects                []ParsedProjectItem        `json:"projects"`
	Education               []ParsedEducationItem      `json:"education"`
	Skills                  []string                   `json:"skills"`
	Achievements            []ParsedAchievementItem    `json:"achievements"`
	Certifications          []ParsedCertificationItem  `json:"certifications"`
	ResearchPatents         []ParsedResearchPatentItem `json:"research_patents"`
	OpenSourceContributions []ParsedOpenSourceItem     `json:"open_source_contributions"`
	CustomFormAnswers       map[string]any             `json:"custom_form_answers"`
}

/*
ExtractStructuredResumeDetails parses JSON embedded within structured resume text delimiters.
*/
func ExtractStructuredResumeDetails(masterCVText string) (
	[]ParsedExperienceItem,
	[]ParsedProjectItem,
	[]ParsedEducationItem,
	[]string,
	[]ParsedAchievementItem,
	[]ParsedCertificationItem,
	[]ParsedResearchPatentItem,
	[]ParsedOpenSourceItem,
) {
	var experiences []ParsedExperienceItem
	var projects []ParsedProjectItem
	var education []ParsedEducationItem
	var skills []string
	var achievements []ParsedAchievementItem
	var certifications []ParsedCertificationItem
	var researchPatents []ParsedResearchPatentItem
	var openSourceContributions []ParsedOpenSourceItem

	delimiterIndex := strings.Index(masterCVText, "--- STRUCTURED RESUME DETAILS ---")
	if delimiterIndex == -1 {
		return experiences, projects, education, skills, achievements, certifications, researchPatents, openSourceContributions
	}

	jsonPayloadText := strings.TrimSpace(masterCVText[delimiterIndex+len("--- STRUCTURED RESUME DETAILS ---"):])
	if jsonPayloadText == "" {
		return experiences, projects, education, skills, achievements, certifications, researchPatents, openSourceContributions
	}

	var payloadMap map[string]json.RawMessage
	if err := json.Unmarshal([]byte(jsonPayloadText), &payloadMap); err != nil {
		return experiences, projects, education, skills, achievements, certifications, researchPatents, openSourceContributions
	}

	if expRaw, ok := payloadMap["experiences"]; ok {
		_ = json.Unmarshal(expRaw, &experiences)
	}
	if projRaw, ok := payloadMap["projects"]; ok {
		_ = json.Unmarshal(projRaw, &projects)
	}
	if eduRaw, ok := payloadMap["education"]; ok {
		_ = json.Unmarshal(eduRaw, &education)
	}
	if skillsRaw, ok := payloadMap["skills"]; ok {
		_ = json.Unmarshal(skillsRaw, &skills)
	}
	if achRaw, ok := payloadMap["achievements"]; ok {
		_ = json.Unmarshal(achRaw, &achievements)
	}
	if certRaw, ok := payloadMap["certifications"]; ok {
		_ = json.Unmarshal(certRaw, &certifications)
	}
	if researchRaw, ok := payloadMap["research_patents"]; ok {
		_ = json.Unmarshal(researchRaw, &researchPatents)
	}
	if osRaw, ok := payloadMap["open_source_contributions"]; ok {
		_ = json.Unmarshal(osRaw, &openSourceContributions)
	}

	return experiences, projects, education, skills, achievements, certifications, researchPatents, openSourceContributions
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
	Duration    string   `json:"duration"`
}

type ParsedAchievementItem struct {
	Title   string `json:"title"`
	Details string `json:"details"`
	Date    string `json:"date"`
}

type ParsedCertificationItem struct {
	Name   string `json:"name"`
	Issuer string `json:"issuer"`
	Date   string `json:"date"`
}

type ParsedEducationItem struct {
	Institution string `json:"institution"`
	Degree      string `json:"degree"`
	Year        string `json:"year"`
	Grade       string `json:"grade"`
}

/*
ParsedResearchPatentItem represents a research publication, conference paper, or granted patent.
*/
type ParsedResearchPatentItem struct {
	Title                     string `json:"title"`
	Authors                   string `json:"authors"`
	PublicationOrPatentNumber string `json:"publication_or_patent_number"`
	Date                      string `json:"date"`
	Link                      string `json:"link"`
	Description               string `json:"description"`
}

/*
ParsedOpenSourceItem represents an open-source software project or codebase contribution.
*/
type ParsedOpenSourceItem struct {
	ProjectName      string   `json:"project_name"`
	ContributionRole string   `json:"contribution_role"`
	TechStack        []string `json:"tech_stack"`
	Link             string   `json:"link"`
	Duration         string   `json:"duration"`
	Description      string   `json:"description"`
}

type ParsedCVResponse struct {
	BioSummary              string                     `json:"bio_summary"`
	Location                string                     `json:"location"`
	Skills                  []string                   `json:"skills"`
	Education               []ParsedEducationItem      `json:"education"`
	Experience              []ParsedExperienceItem     `json:"experience"`
	Projects                []ParsedProjectItem        `json:"projects"`
	Achievements            []ParsedAchievementItem    `json:"achievements"`
	Certifications          []ParsedCertificationItem  `json:"certifications"`
	ResearchPatents         []ParsedResearchPatentItem `json:"research_patents"`
	OpenSourceContributions []ParsedOpenSourceItem     `json:"open_source_contributions"`
	DiscoveredKeywords      []string                   `json:"discovered_keywords"`
	NewKeywords             []string                   `json:"new_keywords"`
}

/*
UpdateProfile persists personal contact info, bio, links, and structured experience records.
*/
func (h *PreferencesHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req ProfileUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	customLinksJSON, marshalLinksError := json.Marshal(req.CustomLinks)
	if marshalLinksError != nil {
		customLinksJSON = []byte("[]")
	}

	experiencesJSON, marshalExpError := json.Marshal(req.Experiences)
	if marshalExpError != nil {
		experiencesJSON = []byte("[]")
	}

	projectsJSON, marshalProjError := json.Marshal(req.Projects)
	if marshalProjError != nil {
		projectsJSON = []byte("[]")
	}

	educationJSON, marshalEduError := json.Marshal(req.Education)
	if marshalEduError != nil {
		educationJSON = []byte("[]")
	}

	skillsJSON, marshalSkillsError := json.Marshal(req.Skills)
	if marshalSkillsError != nil {
		skillsJSON = []byte("[]")
	}

	achievementsJSON, marshalAchError := json.Marshal(req.Achievements)
	if marshalAchError != nil {
		achievementsJSON = []byte("[]")
	}

	certificationsJSON, marshalCertError := json.Marshal(req.Certifications)
	if marshalCertError != nil {
		certificationsJSON = []byte("[]")
	}

	researchPatentsJSON, marshalResearchError := json.Marshal(req.ResearchPatents)
	if marshalResearchError != nil {
		researchPatentsJSON = []byte("[]")
	}

	openSourceJSON, marshalOpenSourceError := json.Marshal(req.OpenSourceContributions)
	if marshalOpenSourceError != nil {
		openSourceJSON = []byte("[]")
	}

	var customFormAnswersJSON []byte
	if req.CustomFormAnswers != nil {
		customFormAnswersJSON, _ = json.Marshal(req.CustomFormAnswers)
	}

	effectiveBio := strings.TrimSpace(req.BioSummary)
	if effectiveBio == "" {
		effectiveBio = strings.TrimSpace(req.BioExperienceText)
	}

	linksMap := map[string]string{
		"linkedin":  req.LinkedInURL,
		"github":    req.GitHubURL,
		"portfolio": req.PortfolioURL,
	}
	linksJSON, _ := json.Marshal(linksMap)

	updateUserQuery := `
		UPDATE users 
		SET phone = $1, location = $2, links = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $4;
	`
	_, _ = h.DB.Exec(context.Background(), updateUserQuery, req.Phone, req.Location, linksJSON, userID)

	upsertQuery := `
		INSERT INTO user_preferences (
			user_id, full_name, email, phone, location, country, linkedin_url, github_url, portfolio_url,
			custom_links, bio_experience_text, master_cv_text, experiences, projects, education, skills, achievements, certifications,
			research_patents, open_source_contributions, custom_form_answers, target_roles, work_models
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11, $12, $13, $14, $15, $16, $17, $18, $19, COALESCE($20::jsonb, '{}'::jsonb), '[]'::jsonb, '[]'::jsonb)
		ON CONFLICT (user_id)
		DO UPDATE SET
			full_name = EXCLUDED.full_name,
			email = EXCLUDED.email,
			phone = EXCLUDED.phone,
			location = EXCLUDED.location,
			country = EXCLUDED.country,
			linkedin_url = EXCLUDED.linkedin_url,
			github_url = EXCLUDED.github_url,
			portfolio_url = EXCLUDED.portfolio_url,
			custom_links = EXCLUDED.custom_links,
			bio_experience_text = CASE WHEN EXCLUDED.bio_experience_text <> '' THEN EXCLUDED.bio_experience_text ELSE user_preferences.bio_experience_text END,
			master_cv_text = CASE WHEN EXCLUDED.bio_experience_text <> '' THEN EXCLUDED.bio_experience_text ELSE user_preferences.master_cv_text END,
			experiences = EXCLUDED.experiences,
			projects = EXCLUDED.projects,
			education = EXCLUDED.education,
			skills = EXCLUDED.skills,
			achievements = EXCLUDED.achievements,
			certifications = EXCLUDED.certifications,
			research_patents = EXCLUDED.research_patents,
			open_source_contributions = EXCLUDED.open_source_contributions,
			custom_form_answers = CASE WHEN $20::jsonb IS NOT NULL THEN $20::jsonb ELSE user_preferences.custom_form_answers END,
			updated_at = CURRENT_TIMESTAMP;
	`
	_, err := h.DB.Exec(
		context.Background(),
		upsertQuery,
		userID,
		req.FullName,
		req.Email,
		req.Phone,
		req.Location,
		req.Country,
		req.LinkedInURL,
		req.GitHubURL,
		req.PortfolioURL,
		customLinksJSON,
		effectiveBio,
		experiencesJSON,
		projectsJSON,
		educationJSON,
		skillsJSON,
		achievementsJSON,
		certificationsJSON,
		researchPatentsJSON,
		openSourceJSON,
		customFormAnswersJSON,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
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
	matchThresholdPercentage := req.MatchThresholdPercentage
	if matchThresholdPercentage <= 0 {
		matchThresholdPercentage = 80
	}

	customLinks := req.CustomLinks
	if customLinks == nil {
		customLinks = []CustomLinkItem{}
	}
	customLinksJSON, marshalLinksError := json.Marshal(customLinks)
	if marshalLinksError != nil {
		customLinksJSON = []byte("[]")
	}

	experiences := req.Experiences
	projects := req.Projects
	education := req.Education
	skills := req.Skills
	achievements := req.Achievements
	certifications := req.Certifications
	researchPatents := req.ResearchPatents
	openSourceContributions := req.OpenSourceContributions

	if len(experiences) == 0 && len(projects) == 0 && strings.Contains(req.MasterCVText, "--- STRUCTURED RESUME DETAILS ---") {
		expExtracted, projExtracted, eduExtracted, skillsExtracted, achExtracted, certExtracted, researchExtracted, openSourceExtracted := ExtractStructuredResumeDetails(req.MasterCVText)
		if len(expExtracted) > 0 {
			experiences = expExtracted
		}
		if len(projExtracted) > 0 {
			projects = projExtracted
		}
		if len(eduExtracted) > 0 {
			education = eduExtracted
		}
		if len(skillsExtracted) > 0 {
			skills = skillsExtracted
		}
		if len(achExtracted) > 0 {
			achievements = achExtracted
		}
		if len(certExtracted) > 0 {
			certifications = certExtracted
		}
		if len(researchExtracted) > 0 {
			researchPatents = researchExtracted
		}
		if len(openSourceExtracted) > 0 {
			openSourceContributions = openSourceExtracted
		}
	}

	targetRoles := req.TargetRoles
	if targetRoles == nil {
		targetRoles = []string{}
	}
	targetRolesJSON, marshalRolesError := json.Marshal(targetRoles)
	if marshalRolesError != nil {
		targetRolesJSON = []byte("[]")
	}

	targetIndustries := req.TargetIndustries
	if targetIndustries == nil {
		targetIndustries = []string{}
	}
	targetIndustriesJSON, marshalIndustriesError := json.Marshal(targetIndustries)
	if marshalIndustriesError != nil {
		targetIndustriesJSON = []byte("[]")
	}

	targetLocations := req.TargetLocations
	if targetLocations == nil {
		targetLocations = []string{}
	}
	targetLocationsJSON, marshalLocationsError := json.Marshal(targetLocations)
	if marshalLocationsError != nil {
		targetLocationsJSON = []byte("[]")
	}

	workModels := req.WorkModels
	if workModels == nil {
		workModels = []string{}
	}
	workModelsJSON, marshalWorkModelsError := json.Marshal(workModels)
	if marshalWorkModelsError != nil {
		workModelsJSON = []byte("[]")
	}

	if experiences == nil {
		experiences = []ParsedExperienceItem{}
	}
	if projects == nil {
		projects = []ParsedProjectItem{}
	}
	if education == nil {
		education = []ParsedEducationItem{}
	}
	if skills == nil {
		skills = []string{}
	}
	if achievements == nil {
		achievements = []ParsedAchievementItem{}
	}
	if certifications == nil {
		certifications = []ParsedCertificationItem{}
	}
	if researchPatents == nil {
		researchPatents = []ParsedResearchPatentItem{}
	}
	if openSourceContributions == nil {
		openSourceContributions = []ParsedOpenSourceItem{}
	}

	experiencesJSON, _ := json.Marshal(experiences)
	projectsJSON, _ := json.Marshal(projects)
	educationJSON, _ := json.Marshal(education)
	skillsJSON, _ := json.Marshal(skills)
	achievementsJSON, _ := json.Marshal(achievements)
	certificationsJSON, _ := json.Marshal(certifications)
	researchPatentsJSON, _ := json.Marshal(researchPatents)
	openSourceJSON, _ := json.Marshal(openSourceContributions)

	var customFormAnswersJSON []byte
	if req.CustomFormAnswers != nil {
		customFormAnswersJSON, _ = json.Marshal(req.CustomFormAnswers)
	}

	query := `
		INSERT INTO user_preferences (
			user_id, full_name, email, phone, location, country, linkedin_url, github_url, portfolio_url,
			custom_links, target_roles, target_industries, target_locations, work_models,
			min_salary, currency, master_cv_text, bio_experience_text, target_resume_pages,
			target_cover_letter_pages, match_threshold_notification_enabled, match_threshold_percentage,
			experiences, projects, education, skills, achievements, certifications,
			research_patents, open_source_contributions, custom_form_answers
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, COALESCE($31::jsonb, '{}'::jsonb))
		ON CONFLICT (user_id) 
		DO UPDATE SET 
			full_name = CASE WHEN EXCLUDED.full_name <> '' AND EXCLUDED.full_name <> 'User' THEN EXCLUDED.full_name ELSE user_preferences.full_name END,
			email = CASE WHEN EXCLUDED.email <> '' THEN EXCLUDED.email ELSE user_preferences.email END,
			phone = CASE WHEN EXCLUDED.phone <> '' THEN EXCLUDED.phone ELSE user_preferences.phone END,
			location = CASE WHEN EXCLUDED.location <> '' THEN EXCLUDED.location ELSE user_preferences.location END,
			country = CASE WHEN EXCLUDED.country <> '' THEN EXCLUDED.country ELSE user_preferences.country END,
			linkedin_url = CASE WHEN EXCLUDED.linkedin_url <> '' THEN EXCLUDED.linkedin_url ELSE user_preferences.linkedin_url END,
			github_url = CASE WHEN EXCLUDED.github_url <> '' THEN EXCLUDED.github_url ELSE user_preferences.github_url END,
			portfolio_url = CASE WHEN EXCLUDED.portfolio_url <> '' THEN EXCLUDED.portfolio_url ELSE user_preferences.portfolio_url END,
			custom_links = CASE WHEN jsonb_typeof(EXCLUDED.custom_links) = 'array' AND jsonb_array_length(EXCLUDED.custom_links) > 0 THEN EXCLUDED.custom_links ELSE user_preferences.custom_links END,
			target_roles = EXCLUDED.target_roles,
			target_industries = EXCLUDED.target_industries,
			target_locations = EXCLUDED.target_locations,
			work_models = EXCLUDED.work_models,
			min_salary = EXCLUDED.min_salary,
			currency = EXCLUDED.currency,
			master_cv_text = CASE WHEN EXCLUDED.master_cv_text <> '' THEN EXCLUDED.master_cv_text ELSE user_preferences.master_cv_text END,
			bio_experience_text = CASE WHEN EXCLUDED.bio_experience_text <> '' THEN EXCLUDED.bio_experience_text ELSE user_preferences.bio_experience_text END,
			target_resume_pages = EXCLUDED.target_resume_pages,
			target_cover_letter_pages = EXCLUDED.target_cover_letter_pages,
			match_threshold_notification_enabled = EXCLUDED.match_threshold_notification_enabled,
			match_threshold_percentage = EXCLUDED.match_threshold_percentage,
			experiences = CASE WHEN jsonb_typeof(EXCLUDED.experiences) = 'array' AND jsonb_array_length(EXCLUDED.experiences) > 0 THEN EXCLUDED.experiences ELSE user_preferences.experiences END,
			projects = CASE WHEN jsonb_typeof(EXCLUDED.projects) = 'array' AND jsonb_array_length(EXCLUDED.projects) > 0 THEN EXCLUDED.projects ELSE user_preferences.projects END,
			education = CASE WHEN jsonb_typeof(EXCLUDED.education) = 'array' AND jsonb_array_length(EXCLUDED.education) > 0 THEN EXCLUDED.education ELSE user_preferences.education END,
			skills = CASE WHEN jsonb_typeof(EXCLUDED.skills) = 'array' AND jsonb_array_length(EXCLUDED.skills) > 0 THEN EXCLUDED.skills ELSE user_preferences.skills END,
			achievements = CASE WHEN jsonb_typeof(EXCLUDED.achievements) = 'array' AND jsonb_array_length(EXCLUDED.achievements) > 0 THEN EXCLUDED.achievements ELSE user_preferences.achievements END,
			certifications = CASE WHEN jsonb_typeof(EXCLUDED.certifications) = 'array' AND jsonb_array_length(EXCLUDED.certifications) > 0 THEN EXCLUDED.certifications ELSE user_preferences.certifications END,
			research_patents = CASE WHEN jsonb_typeof(EXCLUDED.research_patents) = 'array' AND jsonb_array_length(EXCLUDED.research_patents) > 0 THEN EXCLUDED.research_patents ELSE user_preferences.research_patents END,
			open_source_contributions = CASE WHEN jsonb_typeof(EXCLUDED.open_source_contributions) = 'array' AND jsonb_array_length(EXCLUDED.open_source_contributions) > 0 THEN EXCLUDED.open_source_contributions ELSE user_preferences.open_source_contributions END,
			custom_form_answers = CASE WHEN $31::jsonb IS NOT NULL THEN $31::jsonb ELSE user_preferences.custom_form_answers END,
			updated_at = CURRENT_TIMESTAMP;
	`

	_, err := h.DB.Exec(
		context.Background(),
		query,
		userID,
		req.FullName,
		req.Email,
		req.Phone,
		req.Location,
		req.Country,
		req.LinkedInURL,
		req.GitHubURL,
		req.PortfolioURL,
		customLinksJSON,
		targetRolesJSON,
		targetIndustriesJSON,
		targetLocationsJSON,
		workModelsJSON,
		req.MinSalary,
		req.Currency,
		req.MasterCVText,
		req.BioExperienceText,
		targetResumePages,
		targetCoverLetterPages,
		req.MatchThresholdNotificationEnabled,
		matchThresholdPercentage,
		experiencesJSON,
		projectsJSON,
		educationJSON,
		skillsJSON,
		achievementsJSON,
		certificationsJSON,
		researchPatentsJSON,
		openSourceJSON,
		customFormAnswersJSON,
	)
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
			COALESCE(p.email, u.primary_email, ''),
			COALESCE(p.phone, u.phone, ''),
			COALESCE(p.location, u.location, ''),
			COALESCE(p.country, ''),
			COALESCE(p.linkedin_url, u.links->>'linkedin', ''),
			COALESCE(p.github_url, u.links->>'github', ''),
			COALESCE(p.portfolio_url, u.links->>'portfolio', ''),
			COALESCE(p.custom_links, '[]'::jsonb),
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
			COALESCE(p.match_threshold_notification_enabled, false),
			COALESCE(p.match_threshold_percentage, 80),
			COALESCE(p.experiences, '[]'::jsonb),
			COALESCE(p.projects, '[]'::jsonb),
			COALESCE(p.education, '[]'::jsonb),
			COALESCE(p.skills, '[]'::jsonb),
			COALESCE(p.achievements, '[]'::jsonb),
			COALESCE(p.certifications, '[]'::jsonb),
			COALESCE(p.research_patents, '[]'::jsonb),
			COALESCE(p.open_source_contributions, '[]'::jsonb),
			COALESCE(p.custom_form_answers, '{}'::jsonb),
			COALESCE(u.parsed_experience::text, ''),
			(p.user_id IS NOT NULL) AS has_preferences
		FROM users u
		LEFT JOIN user_preferences p ON u.id = p.user_id
		WHERE u.id = $1;
	`

	var pref PreferencesRequest
	var hasPreferences bool
	var customLinksJSON []byte
	var experiencesJSON []byte
	var projectsJSON []byte
	var educationJSON []byte
	var skillsJSON []byte
	var achievementsJSON []byte
	var certificationsJSON []byte
	var researchPatentsJSON []byte
	var openSourceJSON []byte
	var customFormAnswersJSON []byte
	var rawParsedExperience string

	err := h.DB.QueryRow(context.Background(), query, userID).Scan(
		&pref.FullName,
		&pref.Email,
		&pref.Phone,
		&pref.Location,
		&pref.Country,
		&pref.LinkedInURL,
		&pref.GitHubURL,
		&pref.PortfolioURL,
		&customLinksJSON,
		&pref.TargetRoles,
		&pref.TargetIndustries,
		&pref.TargetLocations,
		&pref.WorkModels,
		&pref.MinSalary,
		&pref.Currency,
		&pref.MasterCVText,
		&pref.BioExperienceText,
		&pref.AIMatchingEnabled,
		&pref.TargetResumePages,
		&pref.TargetCoverLetterPages,
		&pref.MatchThresholdNotificationEnabled,
		&pref.MatchThresholdPercentage,
		&experiencesJSON,
		&projectsJSON,
		&educationJSON,
		&skillsJSON,
		&achievementsJSON,
		&certificationsJSON,
		&researchPatentsJSON,
		&openSourceJSON,
		&customFormAnswersJSON,
		&rawParsedExperience,
		&hasPreferences,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch preferences: " + err.Error()})
		return
	}

	if len(customLinksJSON) > 0 {
		_ = json.Unmarshal(customLinksJSON, &pref.CustomLinks)
	}
	if len(experiencesJSON) > 0 {
		_ = json.Unmarshal(experiencesJSON, &pref.Experiences)
	}
	if len(projectsJSON) > 0 {
		_ = json.Unmarshal(projectsJSON, &pref.Projects)
	}
	if len(educationJSON) > 0 {
		_ = json.Unmarshal(educationJSON, &pref.Education)
	}
	if len(skillsJSON) > 0 {
		_ = json.Unmarshal(skillsJSON, &pref.Skills)
	}
	if len(achievementsJSON) > 0 {
		_ = json.Unmarshal(achievementsJSON, &pref.Achievements)
	}
	if len(certificationsJSON) > 0 {
		_ = json.Unmarshal(certificationsJSON, &pref.Certifications)
	}
	if len(researchPatentsJSON) > 0 {
		_ = json.Unmarshal(researchPatentsJSON, &pref.ResearchPatents)
	}
	if len(openSourceJSON) > 0 {
		_ = json.Unmarshal(openSourceJSON, &pref.OpenSourceContributions)
	}
	if len(customFormAnswersJSON) > 0 {
		_ = json.Unmarshal(customFormAnswersJSON, &pref.CustomFormAnswers)
	}
	if pref.CustomFormAnswers == nil {
		pref.CustomFormAnswers = make(map[string]any)
	}

	needsBackfill := false
	if strings.TrimSpace(rawParsedExperience) != "" {
		var parsedResponse ParsedCVResponse
		if unmarshalErr := json.Unmarshal([]byte(rawParsedExperience), &parsedResponse); unmarshalErr == nil {
			if len(pref.Experiences) == 0 && len(parsedResponse.Experience) > 0 {
				pref.Experiences = parsedResponse.Experience
				needsBackfill = true
			}
			if len(pref.Projects) == 0 && len(parsedResponse.Projects) > 0 {
				pref.Projects = parsedResponse.Projects
				needsBackfill = true
			}
			if len(pref.Education) == 0 && len(parsedResponse.Education) > 0 {
				pref.Education = parsedResponse.Education
				needsBackfill = true
			}
			if len(pref.Skills) == 0 && len(parsedResponse.Skills) > 0 {
				pref.Skills = parsedResponse.Skills
				needsBackfill = true
			}
			if len(pref.Achievements) == 0 && len(parsedResponse.Achievements) > 0 {
				pref.Achievements = parsedResponse.Achievements
				needsBackfill = true
			}
			if len(pref.Certifications) == 0 && len(parsedResponse.Certifications) > 0 {
				pref.Certifications = parsedResponse.Certifications
				needsBackfill = true
			}
			if len(pref.ResearchPatents) == 0 && len(parsedResponse.ResearchPatents) > 0 {
				pref.ResearchPatents = parsedResponse.ResearchPatents
				needsBackfill = true
			}
			if len(pref.OpenSourceContributions) == 0 && len(parsedResponse.OpenSourceContributions) > 0 {
				pref.OpenSourceContributions = parsedResponse.OpenSourceContributions
				needsBackfill = true
			}
			if strings.TrimSpace(pref.BioExperienceText) == "" && strings.TrimSpace(parsedResponse.BioSummary) != "" {
				pref.BioExperienceText = strings.TrimSpace(parsedResponse.BioSummary)
			}
		}
	}

	if (len(pref.Experiences) == 0 || len(pref.Projects) == 0 || len(pref.Skills) == 0) && strings.Contains(pref.MasterCVText, "--- STRUCTURED RESUME DETAILS ---") {
		expExtracted, projExtracted, eduExtracted, skillsExtracted, achExtracted, certExtracted, researchExtracted, openSourceExtracted := ExtractStructuredResumeDetails(pref.MasterCVText)
		if len(pref.Experiences) == 0 && len(expExtracted) > 0 {
			pref.Experiences = expExtracted
			needsBackfill = true
		}
		if len(pref.Projects) == 0 && len(projExtracted) > 0 {
			pref.Projects = projExtracted
			needsBackfill = true
		}
		if len(pref.Education) == 0 && len(eduExtracted) > 0 {
			pref.Education = eduExtracted
			needsBackfill = true
		}
		if len(pref.Skills) == 0 && len(skillsExtracted) > 0 {
			pref.Skills = skillsExtracted
			needsBackfill = true
		}
		if len(pref.Achievements) == 0 && len(achExtracted) > 0 {
			pref.Achievements = achExtracted
			needsBackfill = true
		}
		if len(pref.Certifications) == 0 && len(certExtracted) > 0 {
			pref.Certifications = certExtracted
			needsBackfill = true
		}
		if len(pref.ResearchPatents) == 0 && len(researchExtracted) > 0 {
			pref.ResearchPatents = researchExtracted
			needsBackfill = true
		}
		if len(pref.OpenSourceContributions) == 0 && len(openSourceExtracted) > 0 {
			pref.OpenSourceContributions = openSourceExtracted
			needsBackfill = true
		}
	}

	if needsBackfill {
		expBytes, _ := json.Marshal(pref.Experiences)
		projBytes, _ := json.Marshal(pref.Projects)
		eduBytes, _ := json.Marshal(pref.Education)
		skillsBytes, _ := json.Marshal(pref.Skills)
		achBytes, _ := json.Marshal(pref.Achievements)
		certBytes, _ := json.Marshal(pref.Certifications)
		researchBytes, _ := json.Marshal(pref.ResearchPatents)
		openSourceBytes, _ := json.Marshal(pref.OpenSourceContributions)
		go func(targetUserID interface{}, exp, proj, edu, sk, ach, cert, res, os []byte) {
			_, _ = h.DB.Exec(
				context.Background(),
				`UPDATE user_preferences 
				 SET experiences = $1, projects = $2, education = $3, skills = $4, achievements = $5, certifications = $6, research_patents = $7, open_source_contributions = $8
				 WHERE user_id = $9`,
				exp, proj, edu, sk, ach, cert, res, os, targetUserID,
			)
		}(userID, expBytes, projBytes, eduBytes, skillsBytes, achBytes, certBytes, researchBytes, openSourceBytes)
	}

	if strings.TrimSpace(pref.BioExperienceText) == "" {
		if strings.Contains(pref.MasterCVText, "--- STRUCTURED RESUME DETAILS ---") {
			delimiterIndex := strings.Index(pref.MasterCVText, "--- STRUCTURED RESUME DETAILS ---")
			bioPart := strings.TrimSpace(pref.MasterCVText[:delimiterIndex])
			if bioPart != "" {
				pref.BioExperienceText = bioPart
			}
		} else if strings.TrimSpace(pref.MasterCVText) != "" {
			pref.BioExperienceText = strings.TrimSpace(pref.MasterCVText)
		}
	}

	pref.BioSummary = pref.BioExperienceText

	if strings.TrimSpace(pref.BioExperienceText) != "" {
		go func(targetUserID interface{}, bioText string) {
			_, _ = h.DB.Exec(
				context.Background(),
				`UPDATE user_preferences 
				 SET bio_experience_text = $1 
				 WHERE user_id = $2 AND (bio_experience_text IS NULL OR bio_experience_text = '')`,
				bioText, targetUserID,
			)
		}(userID, pref.BioExperienceText)
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
		if len(h.AESKey) != 32 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Server encryption key is not properly configured"})
			return
		}
		encrypted, encryptError := utils.EncryptToken(cleanSecret, h.AESKey)
		if encryptError != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt access token"})
			return
		}
		encryptedToken = &encrypted
		tokenEncrypted = true
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
			"deployment_url":             url,
			"project_name":               projectName,
			"has_secret":                 encryptedToken != "",
			"mcp_secret":                 secret,
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
	Duration    string      `json:"duration"`
}

type flexAchievementItem struct {
	Title   string      `json:"title"`
	Details interface{} `json:"details"`
	Date    string      `json:"date"`
}

type flexCertificationItem struct {
	Name   string `json:"name"`
	Issuer string `json:"issuer"`
	Date   string `json:"date"`
}

type flexResearchPatentItem struct {
	Title                     string `json:"title"`
	Authors                   string `json:"authors"`
	PublicationOrPatentNumber string `json:"publication_or_patent_number"`
	Date                      string `json:"date"`
	Link                      string `json:"link"`
	Description               string `json:"description"`
}

type flexOpenSourceItem struct {
	ProjectName      string      `json:"project_name"`
	ContributionRole string      `json:"contribution_role"`
	TechStack        interface{} `json:"tech_stack"`
	Link             string      `json:"link"`
	Duration         string      `json:"duration"`
	Description      string      `json:"description"`
}

type flexCVResponse struct {
	BioSummary              string                   `json:"bio_summary"`
	Location                string                   `json:"location"`
	Skills                  []string                 `json:"skills"`
	Education               []flexEducationItem      `json:"education"`
	Experience              []flexExperienceItem     `json:"experience"`
	Projects                []flexProjectItem        `json:"projects"`
	Achievements            []flexAchievementItem    `json:"achievements"`
	Certifications          []flexCertificationItem  `json:"certifications"`
	ResearchPatents         []flexResearchPatentItem `json:"research_patents"`
	OpenSourceContributions []flexOpenSourceItem     `json:"open_source_contributions"`
	DiscoveredKeywords      []string                 `json:"discovered_keywords"`
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

	prompt := fmt.Sprintf(`You are an expert technical resume parser. Extract structured education, experience, projects, research/patents, open-source contributions, skills, achievements, certifications, location, a FIRST-PERSON bio summary, and technical domain keywords from the provided raw CV text.

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
      "link": "URL or empty",
      "duration": "Jan 2024 - Present"
    }
  ],
  "research_patents": [
    {
      "title": "Paper or Patent Title",
      "authors": "Author List",
      "publication_or_patent_number": "Conference/Journal or Patent #",
      "date": "May 2024",
      "link": "URL or empty",
      "description": "Short summary"
    }
  ],
  "open_source_contributions": [
    {
      "project_name": "Repo / Org Name",
      "contribution_role": "Contributor / Author",
      "tech_stack": ["Go", "Rust"],
      "link": "PR or Repo URL",
      "duration": "2023 - Present",
      "description": "Short summary of work"
    }
  ],
  "achievements": [
    {
      "title": "Achievement Title",
      "details": "Details",
      "date": "Oct 2024"
    }
  ],
  "certifications": [
    {
      "name": "Certification Name",
      "issuer": "Issuing Org",
      "date": "Jan 2024"
    }
  ],
  "discovered_keywords": ["Golang", "Postgres", "Flutter", "Kubernetes"]
}`, strings.Join(masterList, ", "), req.RawCVText)

	var rawJSON string
	client, errClient := genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  apiKey,
	})
	if errClient != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize Gemini AI client: " + errClient.Error()})
		return
	}

	cvSchemaJSON := `{
		"type": "object",
		"properties": {
			"bio_summary": {"type": "string"},
			"location": {"type": "string"},
			"skills": {
				"type": "array",
				"items": {"type": "string"}
			},
			"education": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"institution": {"type": "string"},
						"degree": {"type": "string"},
						"year": {"type": "string"},
						"grade": {"type": "string"}
					},
					"required": ["institution", "degree"]
				}
			},
			"experience": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"company": {"type": "string"},
						"role": {"type": "string"},
						"duration": {"type": "string"},
						"highlights": {
							"type": "array",
							"items": {"type": "string"}
						}
					},
					"required": ["company", "role"]
				}
			},
			"projects": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"title": {"type": "string"},
						"tech_stack": {
							"type": "array",
							"items": {"type": "string"}
						},
						"description": {"type": "string"},
						"link": {"type": "string"},
						"duration": {"type": "string"}
					},
					"required": ["title"]
				}
			},
			"research_patents": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"title": {"type": "string"},
						"authors": {"type": "string"},
						"publication_or_patent_number": {"type": "string"},
						"date": {"type": "string"},
						"link": {"type": "string"},
						"description": {"type": "string"}
					},
					"required": ["title"]
				}
			},
			"open_source_contributions": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"project_name": {"type": "string"},
						"contribution_role": {"type": "string"},
						"tech_stack": {
							"type": "array",
							"items": {"type": "string"}
						},
						"link": {"type": "string"},
						"duration": {"type": "string"},
						"description": {"type": "string"}
					},
					"required": ["project_name"]
				}
			},
			"achievements": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"title": {"type": "string"},
						"details": {"type": "string"},
						"date": {"type": "string"}
					},
					"required": ["title"]
				}
			},
			"certifications": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"name": {"type": "string"},
						"issuer": {"type": "string"},
						"date": {"type": "string"}
					},
					"required": ["name"]
				}
			},
			"discovered_keywords": {
				"type": "array",
				"items": {"type": "string"}
			}
		},
		"required": ["bio_summary", "location", "skills", "education", "experience", "projects", "achievements", "certifications", "discovered_keywords"]
	}`

	var responseSchema genai.Schema
	if errSchema := json.Unmarshal([]byte(cvSchemaJSON), &responseSchema); errSchema != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unmarshal CV response schema: " + errSchema.Error()})
		return
	}

	config := &genai.GenerateContentConfig{
		ThinkingConfig: &genai.ThinkingConfig{
			ThinkingLevel: genai.ThinkingLevelMinimal,
		},
		ResponseMIMEType: "application/json",
		ResponseSchema:   &responseSchema,
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
			Duration:    item.Duration,
		})
	}

	for _, item := range flexRes.ResearchPatents {
		parsedResponse.ResearchPatents = append(parsedResponse.ResearchPatents, ParsedResearchPatentItem{
			Title:                     item.Title,
			Authors:                   item.Authors,
			PublicationOrPatentNumber: item.PublicationOrPatentNumber,
			Date:                      item.Date,
			Link:                      item.Link,
			Description:               item.Description,
		})
	}

	for _, item := range flexRes.OpenSourceContributions {
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
		parsedResponse.OpenSourceContributions = append(parsedResponse.OpenSourceContributions, ParsedOpenSourceItem{
			ProjectName:      item.ProjectName,
			ContributionRole: item.ContributionRole,
			TechStack:        tsList,
			Link:             item.Link,
			Duration:         item.Duration,
			Description:      item.Description,
		})
	}

	for _, item := range flexRes.Achievements {
		parsedResponse.Achievements = append(parsedResponse.Achievements, ParsedAchievementItem{
			Title:   item.Title,
			Details: stringifyFlex(item.Details),
			Date:    item.Date,
		})
	}

	for _, item := range flexRes.Certifications {
		parsedResponse.Certifications = append(parsedResponse.Certifications, ParsedCertificationItem{
			Name:   item.Name,
			Issuer: item.Issuer,
			Date:   item.Date,
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
