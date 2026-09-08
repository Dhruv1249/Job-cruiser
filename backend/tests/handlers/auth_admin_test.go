package handlers_test

import (
	"os"
	"testing"

	"github.com/Dhruv1249/Job-cruiser/backend/handlers"
)

/*
TestIsMasterAdminEmailUnconfigured verifies that false is returned when MASTER_ADMIN_EMAIL is not set, preventing hardcoded admin promotion.
*/
func TestIsMasterAdminEmailUnconfigured(testingContext *testing.T) {
	os.Unsetenv("MASTER_ADMIN_EMAIL")

	if handlers.IsMasterAdminEmail("dhr1249.lm@gmail.com") {
		testingContext.Fatal("Expected false when MASTER_ADMIN_EMAIL is unset, got true")
	}

	if handlers.IsMasterAdminEmail("admin@example.com") {
		testingContext.Fatal("Expected false for arbitrary email when MASTER_ADMIN_EMAIL is unset, got true")
	}
}

/*
TestIsMasterAdminEmailConfigured verifies case-insensitive matching against configured MASTER_ADMIN_EMAIL.
*/
func TestIsMasterAdminEmailConfigured(testingContext *testing.T) {
	os.Setenv("MASTER_ADMIN_EMAIL", "verified.admin@domain.com")
	defer os.Unsetenv("MASTER_ADMIN_EMAIL")

	if !handlers.IsMasterAdminEmail("verified.admin@domain.com") {
		testingContext.Fatal("Expected true for exact match with MASTER_ADMIN_EMAIL")
	}

	if !handlers.IsMasterAdminEmail("VERIFIED.ADMIN@DOMAIN.COM") {
		testingContext.Fatal("Expected true for case-insensitive match with MASTER_ADMIN_EMAIL")
	}

	if handlers.IsMasterAdminEmail("other.user@domain.com") {
		testingContext.Fatal("Expected false for non-matching email")
	}
}
