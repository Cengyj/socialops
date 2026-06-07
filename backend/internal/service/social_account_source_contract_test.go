package service

import (
	"os"
	"strings"
	"testing"
)

func TestSocialAccountServiceDoesNotExposeAccountSource(t *testing.T) {
	serviceSource := mustReadSource(t, "social_account_service.go")
	schemaSource := mustReadSource(t, "../../ent/schema/social_account.go")

	for _, forbidden := range []string{
		"SocialAccount" + "Source",
		"Source string",
		"`json:\"source\"`",
		"Set" + "Source(",
		"Source" + "EQ(",
		"Field" + "Source",
		"index.Fields(\"source\")",
		"field.String(\"source\")",
	} {
		if strings.Contains(serviceSource, forbidden) || strings.Contains(schemaSource, forbidden) {
			t.Fatalf("social account source contract still contains %q", forbidden)
		}
	}
}

func mustReadSource(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}
