package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNoOverleafConfig = errors.New("user has not configured open-overleaf")

// UserOverleafCredentials holds open-overleaf connection details for a single user.
type UserOverleafCredentials struct {
	DeploymentURL string
}

// LoadUserOverleafCredentials reads user_overleaf_config from Postgres for the given userID.
// Returns ErrNoOverleafConfig when the user has no stored open-overleaf configuration.
func LoadUserOverleafCredentials(
	ctx context.Context,
	databasePool *pgxpool.Pool,
	userID string,
	aesKey []byte,
) (*UserOverleafCredentials, error) {
	if databasePool == nil {
		return nil, errors.New("database pool unavailable")
	}

	var deploymentURL *string

	queryError := databasePool.QueryRow(
		ctx,
		`SELECT deployment_url FROM user_overleaf_config WHERE user_id = $1`,
		userID,
	).Scan(&deploymentURL)

	if queryError != nil {
		if errors.Is(queryError, pgx.ErrNoRows) || strings.Contains(queryError.Error(), "no rows") {
			return nil, ErrNoOverleafConfig
		}
		return nil, fmt.Errorf("failed querying overleaf config for user %s: %w", userID, queryError)
	}

	if deploymentURL == nil || strings.TrimSpace(*deploymentURL) == "" {
		return nil, ErrNoOverleafConfig
	}

	return &UserOverleafCredentials{
		DeploymentURL: strings.TrimSpace(*deploymentURL),
	}, nil
}

// BuildMCPClientForUser constructs an MCPClient using the user's overleaf deployment URL
// and the shared MCP secret as the bearer token.
func BuildMCPClientForUser(credentials *UserOverleafCredentials, mcpSecret string) *MCPClient {
	bearerToken := mcpSecret
	if bearerToken == "" {
		bearerToken = "open_overleaf_mcp_secret"
	}
	return NewMCPClient(credentials.DeploymentURL, bearerToken)
}

// LoadUserMCPClient is a convenience wrapper that loads credentials from Postgres and builds
// a ready-to-use MCPClient for the given userID. Returns ErrNoOverleafConfig when unconfigured.
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
