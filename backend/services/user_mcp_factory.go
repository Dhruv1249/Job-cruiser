package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Dhruv1249/Job-cruiser/backend/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNoOverleafConfig = errors.New("user has not configured open-overleaf")

var ErrNoOverleafSecret = errors.New("open-overleaf access token or secret is not configured")

/*
UserOverleafCredentials holds open-overleaf connection details and active template paths for a single user.
*/
type UserOverleafCredentials struct {
	DeploymentURL           string
	CustomSecret            string
	ProjectName             string
	ResumeTemplatePath      string
	CoverLetterTemplatePath string
}

/*
LoadUserOverleafCredentials reads user_overleaf_config from Postgres for the given userID.
Returns ErrNoOverleafConfig when the user has no stored open-overleaf configuration,
or ErrNoOverleafSecret when the access token/secret is missing.
*/
func LoadUserOverleafCredentials(
	ctx context.Context,
	databasePool *pgxpool.Pool,
	userID string,
	aesKey []byte,
) (*UserOverleafCredentials, error) {
	if databasePool == nil {
		return nil, errors.New("database pool unavailable")
	}

	var deploymentURL, projectName, encryptedToken, resumeTemplatePath, coverLetterTemplatePath *string
	var tokenEncrypted bool

	queryError := databasePool.QueryRow(
		ctx,
		`SELECT deployment_url, COALESCE(project_name, 'job_applications'), encrypted_access_token, COALESCE(token_encrypted, false),
		        COALESCE(resume_template_path, 'templates/resume.tex'), COALESCE(cover_letter_template_path, 'templates/cover_letter.tex')
		 FROM user_overleaf_config WHERE user_id = $1`,
		userID,
	).Scan(&deploymentURL, &projectName, &encryptedToken, &tokenEncrypted, &resumeTemplatePath, &coverLetterTemplatePath)

	if queryError != nil {
		if errors.Is(queryError, pgx.ErrNoRows) || strings.Contains(queryError.Error(), "no rows") {
			return nil, ErrNoOverleafConfig
		}
		return nil, fmt.Errorf("failed querying overleaf config for user %s: %w", userID, queryError)
	}

	if deploymentURL == nil || strings.TrimSpace(*deploymentURL) == "" {
		return nil, ErrNoOverleafConfig
	}

	cleanProject := "job_applications"
	if projectName != nil && strings.TrimSpace(*projectName) != "" {
		cleanProject = strings.TrimSpace(*projectName)
	}

	cleanResumeTemplate := "templates/resume.tex"
	if resumeTemplatePath != nil && strings.TrimSpace(*resumeTemplatePath) != "" {
		cleanResumeTemplate = strings.TrimSpace(*resumeTemplatePath)
	}

	cleanCoverLetterTemplate := "templates/cover_letter.tex"
	if coverLetterTemplatePath != nil && strings.TrimSpace(*coverLetterTemplatePath) != "" {
		cleanCoverLetterTemplate = strings.TrimSpace(*coverLetterTemplatePath)
	}

	customSecret := ""
	if encryptedToken != nil && *encryptedToken != "" {
		if tokenEncrypted && len(aesKey) == 32 {
			decrypted, decryptError := utils.DecryptToken(*encryptedToken, aesKey)
			if decryptError == nil {
				customSecret = decrypted
			} else {
				customSecret = *encryptedToken
			}
		} else {
			customSecret = *encryptedToken
		}
	}

	if customSecret == "" {
		return nil, ErrNoOverleafSecret
	}

	return &UserOverleafCredentials{
		DeploymentURL:           strings.TrimSpace(*deploymentURL),
		CustomSecret:            customSecret,
		ProjectName:             cleanProject,
		ResumeTemplatePath:      cleanResumeTemplate,
		CoverLetterTemplatePath: cleanCoverLetterTemplate,
	}, nil
}

/*
BuildMCPClientForUser constructs an MCPClient using the user's overleaf deployment URL
and the per-user secret as the bearer token.
*/
func BuildMCPClientForUser(credentials *UserOverleafCredentials, mcpSecret string) *MCPClient {
	bearerToken := credentials.CustomSecret
	if bearerToken == "" {
		bearerToken = mcpSecret
	}
	return NewMCPClient(credentials.DeploymentURL, bearerToken)
}

/*
LoadUserMCPClient is a convenience wrapper that loads credentials from Postgres and builds
a ready-to-use MCPClient for the given userID. Returns ErrNoOverleafConfig when unconfigured.
*/
func LoadUserMCPClient(
	ctx context.Context,
	databasePool *pgxpool.Pool,
	userID string,
	aesKey []byte,
	mcpSecret string,
) (*MCPClient, *UserOverleafCredentials, error) {
	credentials, credError := LoadUserOverleafCredentials(ctx, databasePool, userID, aesKey)
	if credError != nil {
		return nil, nil, credError
	}
	mcpClient := BuildMCPClientForUser(credentials, mcpSecret)
	return mcpClient, credentials, nil
}
