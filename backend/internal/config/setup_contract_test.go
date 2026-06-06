package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupInstallTrimsDatabaseSSLModeLikeConnectionTest(t *testing.T) {
	root := findRepositoryRoot(t)
	handlerPath := filepath.Join(root, "backend", "internal", "setup", "handler.go")
	data, err := os.ReadFile(handlerPath)
	if err != nil {
		t.Fatalf("read setup handler: %v", err)
	}
	source := string(data)

	testDB := extractFunctionBody(t, source, "testDatabase")
	install := extractFunctionBody(t, source, "install")

	required := "req.SSLMode = strings.TrimSpace(req.SSLMode)"
	if !strings.Contains(testDB, required) {
		t.Fatalf("setup testDatabase must trim database sslmode before validation")
	}

	installRequired := "req.Database.SSLMode = strings.TrimSpace(req.Database.SSLMode)"
	if !strings.Contains(install, installRequired) {
		t.Fatalf("setup install must trim database sslmode before validateSSLMode, matching /setup/test-db")
	}
}

func extractFunctionBody(t *testing.T, source, name string) string {
	t.Helper()
	marker := "func " + name + "("
	start := strings.Index(source, marker)
	if start < 0 {
		t.Fatalf("function %s not found", name)
	}
	brace := strings.Index(source[start:], "{")
	if brace < 0 {
		t.Fatalf("function %s has no body", name)
	}
	bodyStart := start + brace
	depth := 0
	for i := bodyStart; i < len(source); i++ {
		switch source[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return source[bodyStart+1 : i]
			}
		}
	}
	t.Fatalf("function %s body is not closed", name)
	return ""
}

func findRepositoryRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "PROJECT_GUIDE.md")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("repository root with PROJECT_GUIDE.md not found from %s", dir)
		}
		dir = parent
	}
}
