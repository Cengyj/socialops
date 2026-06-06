package repository

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnnouncementRepositoryUsesTxContextAwareClient(t *testing.T) {
	source, err := os.ReadFile("announcement_repo.go")
	require.NoError(t, err)
	content := string(source)

	for _, method := range []string{
		"Create",
		"GetByID",
		"Update",
		"Delete",
		"List",
		"ListActive",
	} {
		t.Run(method, func(t *testing.T) {
			body := repositoryMethodBody(t, content, "announcementRepository", method)
			require.Contains(t, body, "client := clientFromContext(ctx, r.client)",
				"%s must reuse dbent.NewTxContext transaction clients", method)
		})
	}
}

func TestAnnouncementReadRepositoryUsesTxContextAwareClient(t *testing.T) {
	source, err := os.ReadFile("announcement_read_repo.go")
	require.NoError(t, err)
	content := string(source)

	for _, method := range []string{
		"MarkRead",
		"GetReadMapByUser",
		"GetReadMapByUsers",
		"CountByAnnouncementID",
	} {
		t.Run(method, func(t *testing.T) {
			body := repositoryMethodBody(t, content, "announcementReadRepository", method)
			require.Contains(t, body, "client := clientFromContext(ctx, r.client)",
				"%s must reuse dbent.NewTxContext transaction clients", method)
		})
	}
}

func repositoryMethodBody(t *testing.T, content string, receiver string, method string) string {
	t.Helper()
	needle := "func (r *" + receiver + ") " + method + "("
	start := strings.Index(content, needle)
	require.NotEqual(t, -1, start, "method %s not found", method)

	next := strings.Index(content[start+len(needle):], "\nfunc ")
	if next == -1 {
		return content[start:]
	}
	return content[start : start+len(needle)+next]
}
