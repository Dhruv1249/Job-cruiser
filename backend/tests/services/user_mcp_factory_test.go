package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Dhruv1249/Job-cruiser/backend/services"
)

func TestBuildMCPClientForUserProducesValidClient(t *testing.T) {
	credentials := &services.UserOverleafCredentials{
		DeploymentURL: "https://overleaf.example.com",
		CustomSecret:  "user-encrypted-mcp-secret",
	}
	mcpClient := services.BuildMCPClientForUser(credentials, "")
	if mcpClient == nil {
		t.Fatal("expected non-nil MCPClient")
	}
	if mcpClient.BaseURL != "https://overleaf.example.com" {
		t.Fatalf("unexpected base URL: %s", mcpClient.BaseURL)
	}
	if mcpClient.Token != "user-encrypted-mcp-secret" {
		t.Fatalf("expected bearer token 'user-encrypted-mcp-secret', got %q", mcpClient.Token)
	}
}

func TestBuildMCPClientTokenIsDeterministic(t *testing.T) {
	credentials := &services.UserOverleafCredentials{
		DeploymentURL: "https://overleaf.example.com",
		CustomSecret:  "shared-secret",
	}
	firstToken := services.BuildMCPClientForUser(credentials, "shared-secret").Token
	secondToken := services.BuildMCPClientForUser(credentials, "shared-secret").Token
	if firstToken != secondToken {
		t.Fatal("BuildMCPClientForUser must produce deterministic token for same inputs")
	}
}

func TestLoadUserOverleafCredentialsNoConfig(t *testing.T) {
	aesKey := make([]byte, 32)
	_, credErr := services.LoadUserOverleafCredentials(context.Background(), nil, "non-existent-user", aesKey)
	if credErr == nil {
		t.Fatal("expected error when pool is nil")
	}
}

func TestErrNoOverleafConfigIsDistinctError(t *testing.T) {
	if services.ErrNoOverleafConfig == nil {
		t.Fatal("ErrNoOverleafConfig must be non-nil")
	}
	if !errors.Is(services.ErrNoOverleafConfig, services.ErrNoOverleafConfig) {
		t.Fatal("ErrNoOverleafConfig must satisfy errors.Is with itself")
	}
	if services.ErrNoOverleafSecret == nil {
		t.Fatal("ErrNoOverleafSecret must be non-nil")
	}
}
