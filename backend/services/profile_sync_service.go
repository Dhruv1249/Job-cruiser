package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

/*
ProfileCustomLinkItem represents an arbitrary external portfolio, blog, or project hyperlink.
*/
type ProfileCustomLinkItem struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

/*
ProfileExperienceItem represents a past or current professional employment position.
*/
type ProfileExperienceItem struct {
	Company    string   `json:"company"`
	Role       string   `json:"role"`
	Duration   string   `json:"duration"`
	Highlights string   `json:"highlights"`
	TechStack  []string `json:"tech_stack"`
}

/*
ProfileProjectItem represents a technical software application, tool, or engineering project.
*/
type ProfileProjectItem struct {
	Title         string   `json:"title"`
	TechStack     []string `json:"tech_stack"`
	Description   string   `json:"description"`
	Link          string   `json:"link"`
	GithubURL     string   `json:"github_url"`
	DeploymentURL string   `json:"deployment_url"`
	Duration      string   `json:"duration"`
}

/*
ProfileAchievementItem represents an award, competitive accomplishment, or honor.
*/
type ProfileAchievementItem struct {
	Title   string `json:"title"`
	Details string `json:"details"`
	Date    string `json:"date"`
	Link    string `json:"link"`
}

/*
ProfileCertificationItem represents an industry credential or verified license.
*/
type ProfileCertificationItem struct {
	Name   string `json:"name"`
	Issuer string `json:"issuer"`
	Date   string `json:"date"`
	Link   string `json:"link"`
}

/*
ProfileEducationItem represents an academic degree or coursework program.
*/
type ProfileEducationItem struct {
	Institution string `json:"institution"`
	Degree      string `json:"degree"`
	Year        string `json:"year"`
	Grade       string `json:"grade"`
}

/*
ProfileResearchPatentItem represents a research publication, conference paper, or granted patent.
*/
type ProfileResearchPatentItem struct {
	Title                     string `json:"title"`
	Authors                   string `json:"authors"`
	PublicationOrPatentNumber string `json:"publication_or_patent_number"`
	Date                      string `json:"date"`
	Link                      string `json:"link"`
	Description               string `json:"description"`
}

/*
ProfileOpenSourceItem represents an open-source software project or community repository contribution.
*/
type ProfileOpenSourceItem struct {
	ProjectName      string   `json:"project_name"`
	ContributionRole string   `json:"contribution_role"`
	TechStack        []string `json:"tech_stack"`
	Link             string   `json:"link"`
	Duration         string   `json:"duration"`
	Description      string   `json:"description"`
}

/*
CandidateProfileSyncPayload encapsulates candidate biographical information, contact links,
and structured resume sections for Open-Overleaf synchronization.
*/
type CandidateProfileSyncPayload struct {
	FullName                 string                      `json:"name"`
	ProfessionalHeadline     string                      `json:"professional_headline"`
	Email                    string                      `json:"email"`
	Phone                    string                      `json:"phone"`
	Location                 string                      `json:"location"`
	Country                  string                      `json:"country"`
	LinkedInURL              string                      `json:"linkedin_url"`
	GithubURL                string                      `json:"github_url"`
	PortfolioURL             string                      `json:"portfolio_url"`
	CustomLinks              []ProfileCustomLinkItem     `json:"custom_links"`
	BioSummary               string                      `json:"bio_summary"`
	Skills                   []string                    `json:"skills"`
	Experiences              []ProfileExperienceItem     `json:"experiences"`
	Projects                 []ProfileProjectItem        `json:"projects"`
	Education                []ProfileEducationItem      `json:"education"`
	Achievements             []ProfileAchievementItem    `json:"achievements"`
	Certifications           []ProfileCertificationItem  `json:"certifications"`
	ResearchPatents          []ProfileResearchPatentItem `json:"research_patents"`
	OpenSourceContributions  []ProfileOpenSourceItem     `json:"open_source_contributions"`
	SyncedAt                 string                      `json:"synced_at"`
	LastSyncedAt             string                      `json:"last_synced_at"`
	SyncedBy                 string                      `json:"synced_by"`
}

/*
ProfileSyncService handles formatting and syncing candidate profiles to Open-Overleaf projects.
*/
type ProfileSyncService struct {
	DatabasePool *pgxpool.Pool
	AESKey       []byte
	MCPSecret    string
}

/*
NewProfileSyncService creates an initialized ProfileSyncService instance.
*/
func NewProfileSyncService(databasePool *pgxpool.Pool, aesKey []byte, mcpSecret string) *ProfileSyncService {
	return &ProfileSyncService{
		DatabasePool: databasePool,
		AESKey:       aesKey,
		MCPSecret:    mcpSecret,
	}
}

/*
MergeProjectTechIntoSkills combines project technologies into candidate skills avoiding duplicates.
*/
func MergeProjectTechIntoSkills(existingSkills []string, projects []ProfileProjectItem) []string {
	skillSet := make(map[string]struct{})
	for _, skill := range existingSkills {
		normalizedSkill := strings.ToLower(strings.TrimSpace(skill))
		if normalizedSkill != "" {
			skillSet[normalizedSkill] = struct{}{}
		}
	}
	mergedSkills := append([]string{}, existingSkills...)
	for _, project := range projects {
		for _, tech := range project.TechStack {
			trimmedTech := strings.TrimSpace(tech)
			normalizedTech := strings.ToLower(trimmedTech)
			if normalizedTech != "" {
				if _, exists := skillSet[normalizedTech]; !exists {
					skillSet[normalizedTech] = struct{}{}
					mergedSkills = append(mergedSkills, trimmedTech)
				}
			}
		}
	}
	return mergedSkills
}

/*
EscapeLaTeXText escapes reserved LaTeX formatting characters within arbitrary strings.
*/
func EscapeLaTeXText(rawText string) string {
	replacer := strings.NewReplacer(
		"\\", `\textbackslash{}`,
		"&", `\&`,
		"%", `\%`,
		"$", `\$`,
		"#", `\#`,
		"_", `\_`,
		"{", `\{`,
		"}", `\}`,
		"~", `\textasciitilde{}`,
		"^", `\textasciicircum{}`,
	)
	return replacer.Replace(rawText)
}

/*
FormatProfileJSON serializes the candidate profile payload into formatted JSON bytes.
*/
func FormatProfileJSON(candidateProfile CandidateProfileSyncPayload) ([]byte, error) {
	if candidateProfile.SyncedAt == "" {
		candidateProfile.SyncedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if candidateProfile.LastSyncedAt == "" {
		candidateProfile.LastSyncedAt = candidateProfile.SyncedAt
	}
	if candidateProfile.SyncedBy == "" {
		candidateProfile.SyncedBy = "Job-cruiser"
	}
	return json.MarshalIndent(candidateProfile, "", "  ")
}

/*
FormatProfileMarkdown generates a clean, structured Markdown experience bank suitable for AI reading.
*/
func FormatProfileMarkdown(candidateProfile CandidateProfileSyncPayload) string {
	var builder strings.Builder

	syncedTime := candidateProfile.SyncedAt
	if syncedTime == "" {
		syncedTime = time.Now().UTC().Format(time.RFC3339)
	}

	builder.WriteString("# Candidate Profile & Experience Bank\n\n")
	builder.WriteString(fmt.Sprintf("<!-- Last Synced: %s by Job-cruiser -->\n", syncedTime))
	builder.WriteString(fmt.Sprintf("> **Last Synced**: %s (via Job-cruiser)\n\n", syncedTime))

	builder.WriteString("## Personal Information\n")
	if candidateProfile.FullName != "" {
		builder.WriteString(fmt.Sprintf("- **Name**: %s\n", candidateProfile.FullName))
	}
	if candidateProfile.ProfessionalHeadline != "" {
		builder.WriteString(fmt.Sprintf("- **Professional Headline**: %s\n", candidateProfile.ProfessionalHeadline))
	}

	contactSegments := make([]string, 0, 3)
	if candidateProfile.Email != "" {
		contactSegments = append(contactSegments, fmt.Sprintf("**Email**: %s", candidateProfile.Email))
	}
	if candidateProfile.Phone != "" {
		contactSegments = append(contactSegments, fmt.Sprintf("**Phone**: %s", candidateProfile.Phone))
	}
	if candidateProfile.Location != "" {
		contactSegments = append(contactSegments, fmt.Sprintf("**Location**: %s", candidateProfile.Location))
	}
	if len(contactSegments) > 0 {
		builder.WriteString(fmt.Sprintf("- %s\n", strings.Join(contactSegments, " | ")))
	}

	linkSegments := make([]string, 0, 3+len(candidateProfile.CustomLinks))
	if candidateProfile.LinkedInURL != "" {
		linkSegments = append(linkSegments, fmt.Sprintf("[LinkedIn](%s)", candidateProfile.LinkedInURL))
	}
	if candidateProfile.GithubURL != "" {
		linkSegments = append(linkSegments, fmt.Sprintf("[GitHub](%s)", candidateProfile.GithubURL))
	}
	if candidateProfile.PortfolioURL != "" {
		linkSegments = append(linkSegments, fmt.Sprintf("[Portfolio](%s)", candidateProfile.PortfolioURL))
	}
	for _, customLink := range candidateProfile.CustomLinks {
		if customLink.URL != "" {
			label := customLink.Label
			if label == "" {
				label = "Link"
			}
			linkSegments = append(linkSegments, fmt.Sprintf("[%s](%s)", label, customLink.URL))
		}
	}
	if len(linkSegments) > 0 {
		builder.WriteString(fmt.Sprintf("- **Links**: %s\n", strings.Join(linkSegments, " | ")))
	}

	if strings.TrimSpace(candidateProfile.BioSummary) != "" {
		builder.WriteString("\n## Professional Summary\n")
		builder.WriteString(strings.TrimSpace(candidateProfile.BioSummary))
		builder.WriteString("\n")
	}

	if len(candidateProfile.Skills) > 0 {
		builder.WriteString("\n## Core Skills & Technologies\n")
		builder.WriteString(strings.Join(candidateProfile.Skills, ", "))
		builder.WriteString("\n")
	}

	if len(candidateProfile.Experiences) > 0 {
		builder.WriteString("\n## Work Experience\n")
		for _, experience := range candidateProfile.Experiences {
			builder.WriteString(fmt.Sprintf("\n### %s — %s\n", experience.Role, experience.Company))
			if experience.Duration != "" {
				builder.WriteString(fmt.Sprintf("- **Duration**: %s\n", experience.Duration))
			}
			if len(experience.TechStack) > 0 {
				builder.WriteString(fmt.Sprintf("- **Tech Stack**: %s\n", strings.Join(experience.TechStack, ", ")))
			}
			if experience.Highlights != "" {
				highlightLines := strings.Split(experience.Highlights, "\n")
				for _, line := range highlightLines {
					trimmedLine := strings.TrimSpace(line)
					if trimmedLine != "" {
						trimmedLine = strings.TrimPrefix(trimmedLine, "- ")
						trimmedLine = strings.TrimPrefix(trimmedLine, "* ")
						builder.WriteString(fmt.Sprintf("- %s\n", trimmedLine))
					}
				}
			}
		}
	}

	if len(candidateProfile.Projects) > 0 {
		builder.WriteString("\n## Technical Projects\n")
		for _, project := range candidateProfile.Projects {
			builder.WriteString(fmt.Sprintf("\n### %s\n", project.Title))
			if project.Duration != "" {
				builder.WriteString(fmt.Sprintf("- **Duration**: %s\n", project.Duration))
			}
			if len(project.TechStack) > 0 {
				builder.WriteString(fmt.Sprintf("- **Technologies**: %s\n", strings.Join(project.TechStack, ", ")))
			}
			projectLinkSegments := make([]string, 0, 2)
			if project.GithubURL != "" {
				projectLinkSegments = append(projectLinkSegments, fmt.Sprintf("[Source Code](%s)", project.GithubURL))
			}
			if project.DeploymentURL != "" {
				projectLinkSegments = append(projectLinkSegments, fmt.Sprintf("[Live Demo](%s)", project.DeploymentURL))
			} else if project.Link != "" && project.Link != project.GithubURL {
				projectLinkSegments = append(projectLinkSegments, fmt.Sprintf("[Link](%s)", project.Link))
			}
			if len(projectLinkSegments) > 0 {
				builder.WriteString(fmt.Sprintf("- **Project Links**: %s\n", strings.Join(projectLinkSegments, " | ")))
			}
			if strings.TrimSpace(project.Description) != "" {
				descLines := strings.Split(project.Description, "\n")
				for _, line := range descLines {
					trimmedLine := strings.TrimSpace(line)
					if trimmedLine != "" {
						builder.WriteString(fmt.Sprintf("- %s\n", trimmedLine))
					}
				}
			}
		}
	}

	if len(candidateProfile.Education) > 0 {
		builder.WriteString("\n## Education\n")
		for _, educationItem := range candidateProfile.Education {
			gradeSegment := ""
			if educationItem.Grade != "" {
				gradeSegment = fmt.Sprintf(" — Grade: %s", educationItem.Grade)
			}
			yearSegment := ""
			if educationItem.Year != "" {
				yearSegment = fmt.Sprintf(" (%s)", educationItem.Year)
			}
			builder.WriteString(fmt.Sprintf("- **%s**, %s%s%s\n", educationItem.Degree, educationItem.Institution, yearSegment, gradeSegment))
		}
	}

	if len(candidateProfile.Certifications) > 0 {
		builder.WriteString("\n## Certifications\n")
		for _, certificationItem := range candidateProfile.Certifications {
			issuerSegment := ""
			if certificationItem.Issuer != "" {
				issuerSegment = fmt.Sprintf(" (%s)", certificationItem.Issuer)
			}
			dateSegment := ""
			if certificationItem.Date != "" {
				dateSegment = fmt.Sprintf(" — %s", certificationItem.Date)
			}
			builder.WriteString(fmt.Sprintf("- **%s**%s%s\n", certificationItem.Name, issuerSegment, dateSegment))
		}
	}

	if len(candidateProfile.Achievements) > 0 {
		builder.WriteString("\n## Honors & Achievements\n")
		for _, achievementItem := range candidateProfile.Achievements {
			builder.WriteString(fmt.Sprintf("- **%s**: %s\n", achievementItem.Title, achievementItem.Details))
		}
	}

	if len(candidateProfile.ResearchPatents) > 0 {
		builder.WriteString("\n## Research & Patents\n")
		for _, researchItem := range candidateProfile.ResearchPatents {
			builder.WriteString(fmt.Sprintf("- **%s** (%s) - %s\n", researchItem.Title, researchItem.PublicationOrPatentNumber, researchItem.Description))
		}
	}

	if len(candidateProfile.OpenSourceContributions) > 0 {
		builder.WriteString("\n## Open Source Contributions\n")
		for _, openSourceItem := range candidateProfile.OpenSourceContributions {
			builder.WriteString(fmt.Sprintf("- **%s** (%s): %s\n", openSourceItem.ProjectName, openSourceItem.ContributionRole, openSourceItem.Description))
		}
	}

	return builder.String()
}

/*
FormatProfileLaTeXVariables produces standard LaTeX variable macros for document inclusion.
*/
func FormatProfileLaTeXVariables(candidateProfile CandidateProfileSyncPayload) string {
	var builder strings.Builder

	syncedTime := candidateProfile.SyncedAt
	if syncedTime == "" {
		syncedTime = time.Now().UTC().Format(time.RFC3339)
	}

	builder.WriteString("% Auto-generated candidate profile macros synced from Job-cruiser\n")
	builder.WriteString(fmt.Sprintf("%% Last Synced: %s\n", syncedTime))
	builder.WriteString(fmt.Sprintf("\\newcommand{\\candidateLastSyncedAt}{%s}\n", EscapeLaTeXText(syncedTime)))
	builder.WriteString(fmt.Sprintf("\\newcommand{\\candidateName}{%s}\n", EscapeLaTeXText(candidateProfile.FullName)))
	builder.WriteString(fmt.Sprintf("\\newcommand{\\candidateHeadline}{%s}\n", EscapeLaTeXText(candidateProfile.ProfessionalHeadline)))
	builder.WriteString(fmt.Sprintf("\\newcommand{\\candidateEmail}{%s}\n", EscapeLaTeXText(candidateProfile.Email)))
	builder.WriteString(fmt.Sprintf("\\newcommand{\\candidatePhone}{%s}\n", EscapeLaTeXText(candidateProfile.Phone)))
	builder.WriteString(fmt.Sprintf("\\newcommand{\\candidateLocation}{%s}\n", EscapeLaTeXText(candidateProfile.Location)))
	builder.WriteString(fmt.Sprintf("\\newcommand{\\candidateCountry}{%s}\n", EscapeLaTeXText(candidateProfile.Country)))
	builder.WriteString(fmt.Sprintf("\\newcommand{\\candidateLinkedIn}{%s}\n", EscapeLaTeXText(candidateProfile.LinkedInURL)))
	builder.WriteString(fmt.Sprintf("\\newcommand{\\candidateGitHub}{%s}\n", EscapeLaTeXText(candidateProfile.GithubURL)))
	builder.WriteString(fmt.Sprintf("\\newcommand{\\candidatePortfolio}{%s}\n", EscapeLaTeXText(candidateProfile.PortfolioURL)))

	return builder.String()
}

/*
FetchCandidateProfilePayload retrieves comprehensive candidate profile information from PostgreSQL.
*/
func (service *ProfileSyncService) FetchCandidateProfilePayload(
	ctx context.Context,
	userID string,
) (*CandidateProfileSyncPayload, error) {
	queryStatement := `
		SELECT
			COALESCE(u.phone, ''),
			COALESCE(u.location, ''),
			COALESCE(u.links, '{}'::jsonb),
			COALESCE(up.full_name, ''),
			COALESCE(up.professional_headline, ''),
			COALESCE(up.email, u.primary_email, ''),
			COALESCE(up.phone, u.phone, ''),
			COALESCE(up.location, u.location, ''),
			COALESCE(up.country, ''),
			COALESCE(up.linkedin_url, ''),
			COALESCE(up.github_url, ''),
			COALESCE(up.portfolio_url, ''),
			COALESCE(up.custom_links, '[]'::jsonb),
			COALESCE(up.bio_experience_text, ''),
			COALESCE(up.skills, '[]'::jsonb),
			COALESCE(up.projects, '[]'::jsonb),
			COALESCE(up.experiences, '[]'::jsonb),
			COALESCE(up.education, '[]'::jsonb),
			COALESCE(up.achievements, '[]'::jsonb),
			COALESCE(up.certifications, '[]'::jsonb),
			COALESCE(up.research_patents, '[]'::jsonb),
			COALESCE(up.open_source_contributions, '[]'::jsonb)
		FROM users u
		LEFT JOIN user_preferences up ON u.id = up.user_id
		WHERE u.id = $1`

	var userPhone, userLocation string
	var userLinksRaw []byte
	var fullName, professionalHeadline, email, phone, location, country string
	var linkedinURL, githubURL, portfolioURL, bioText string
	var customLinksRaw, skillsRaw, projectsRaw, experiencesRaw []byte
	var educationRaw, achievementsRaw, certificationsRaw, researchRaw, openSourceRaw []byte

	queryError := service.DatabasePool.QueryRow(ctx, queryStatement, userID).Scan(
		&userPhone, &userLocation, &userLinksRaw,
		&fullName, &professionalHeadline, &email, &phone, &location, &country,
		&linkedinURL, &githubURL, &portfolioURL,
		&customLinksRaw, &bioText, &skillsRaw, &projectsRaw, &experiencesRaw,
		&educationRaw, &achievementsRaw, &certificationsRaw, &researchRaw, &openSourceRaw,
	)
	if queryError != nil {
		return nil, fmt.Errorf("failed querying user profile for sync: %w", queryError)
	}

	var customLinks []ProfileCustomLinkItem
	_ = json.Unmarshal(customLinksRaw, &customLinks)

	var skills []string
	_ = json.Unmarshal(skillsRaw, &skills)

	var experiences []ProfileExperienceItem
	_ = json.Unmarshal(experiencesRaw, &experiences)

	var projects []ProfileProjectItem
	_ = json.Unmarshal(projectsRaw, &projects)

	var education []ProfileEducationItem
	_ = json.Unmarshal(educationRaw, &education)

	var achievements []ProfileAchievementItem
	_ = json.Unmarshal(achievementsRaw, &achievements)

	var certifications []ProfileCertificationItem
	_ = json.Unmarshal(certificationsRaw, &certifications)

	var researchPatents []ProfileResearchPatentItem
	_ = json.Unmarshal(researchRaw, &researchPatents)

	var openSourceContributions []ProfileOpenSourceItem
	_ = json.Unmarshal(openSourceRaw, &openSourceContributions)

	mergedSkills := MergeProjectTechIntoSkills(skills, projects)

	return &CandidateProfileSyncPayload{
		FullName:                fullName,
		ProfessionalHeadline:    professionalHeadline,
		Email:                   email,
		Phone:                   phone,
		Location:                location,
		Country:                 country,
		LinkedInURL:             linkedinURL,
		GithubURL:               githubURL,
		PortfolioURL:            portfolioURL,
		CustomLinks:             customLinks,
		BioSummary:              bioText,
		Skills:                  mergedSkills,
		Experiences:             experiences,
		Projects:                projects,
		Education:               education,
		Achievements:            achievements,
		Certifications:          certifications,
		ResearchPatents:         researchPatents,
		OpenSourceContributions: openSourceContributions,
		SyncedAt:                time.Now().UTC().Format(time.RFC3339),
	}, nil
}

/*
SyncUserProfileToOverleaf writes structured profile.json, markdown PROFILE.md, and LaTeX profile_vars.tex
into the user's Open-Overleaf workspace via MCP.
*/
func (service *ProfileSyncService) SyncUserProfileToOverleaf(
	ctx context.Context,
	userID string,
) ([]string, error) {
	mcpClient, credentials, mcpClientError := LoadUserMCPClient(
		ctx,
		service.DatabasePool,
		userID,
		service.AESKey,
		service.MCPSecret,
	)
	if mcpClientError != nil {
		return nil, mcpClientError
	}

	candidateProfile, fetchError := service.FetchCandidateProfilePayload(ctx, userID)
	if fetchError != nil {
		return nil, fetchError
	}

	profileJSONBytes, jsonError := FormatProfileJSON(*candidateProfile)
	if jsonError != nil {
		return nil, fmt.Errorf("failed formatting profile JSON: %w", jsonError)
	}

	profileMarkdown := FormatProfileMarkdown(*candidateProfile)
	profileLaTeXVars := FormatProfileLaTeXVariables(*candidateProfile)

	projectName := credentials.ProjectName

	syncedFiles := []string{
		"profile/profile.json",
		"profile/PROFILE.md",
		"profile/profile_vars.tex",
	}

	if writeError := mcpClient.WriteProjectFile(ctx, projectName, "profile/profile.json", string(profileJSONBytes)); writeError != nil {
		return nil, fmt.Errorf("failed writing profile/profile.json to open-overleaf: %w", writeError)
	}

	if writeError := mcpClient.WriteProjectFile(ctx, projectName, "profile/PROFILE.md", profileMarkdown); writeError != nil {
		return nil, fmt.Errorf("failed writing profile/PROFILE.md to open-overleaf: %w", writeError)
	}

	if writeError := mcpClient.WriteProjectFile(ctx, projectName, "profile/profile_vars.tex", profileLaTeXVars); writeError != nil {
		return nil, fmt.Errorf("failed writing profile/profile_vars.tex to open-overleaf: %w", writeError)
	}

	if service.DatabasePool != nil {
		_, _ = service.DatabasePool.Exec(ctx, `UPDATE user_overleaf_config SET last_synced_at = CURRENT_TIMESTAMP WHERE user_id = $1`, userID)
	}

	return syncedFiles, nil
}

/*
SyncAllDueProfiles queries all users who have auto sync enabled and whose sync interval has passed, then syncs them.
*/
func (service *ProfileSyncService) SyncAllDueProfiles(ctx context.Context) (int, error) {
	if service.DatabasePool == nil {
		return 0, nil
	}

	query := `
		SELECT uoc.user_id::text
		FROM user_overleaf_config uoc
		WHERE COALESCE(uoc.auto_sync_profile, true) = true
		  AND uoc.deployment_url IS NOT NULL
		  AND uoc.deployment_url != ''
		  AND uoc.encrypted_access_token IS NOT NULL
		  AND uoc.encrypted_access_token != ''
		  AND (
		    uoc.last_synced_at IS NULL
		    OR uoc.last_synced_at <= NOW() - (COALESCE(uoc.sync_interval_hours, 24) || ' hours')::interval
		  )
		LIMIT 100;
	`

	rows, queryErr := service.DatabasePool.Query(ctx, query)
	if queryErr != nil {
		return 0, fmt.Errorf("failed querying users due for profile sync: %w", queryErr)
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var uid string
		if scanErr := rows.Scan(&uid); scanErr == nil {
			userIDs = append(userIDs, uid)
		}
	}

	syncedCount := 0
	for _, uid := range userIDs {
		_, syncErr := service.SyncUserProfileToOverleaf(ctx, uid)
		if syncErr == nil {
			syncedCount++
		}
	}

	return syncedCount, nil
}

/*
StartBackgroundSyncScheduler runs a recurring ticker that checks for and synchronizes due profiles to Open-Overleaf.
*/
func (service *ProfileSyncService) StartBackgroundSyncScheduler(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		_, _ = service.SyncAllDueProfiles(ctx)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				_, _ = service.SyncAllDueProfiles(ctx)
			}
		}
	}()
}
