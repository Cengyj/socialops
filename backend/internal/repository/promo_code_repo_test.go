package repository

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPromoCodeRepositoryUsesTxContextAwareClient(t *testing.T) {
	source, err := os.ReadFile("promo_code_repo.go")
	require.NoError(t, err)
	content := string(source)

	for _, method := range []string{
		"GetByID",
		"GetByCode",
		"GetByCodeForUpdate",
		"ListWithFilters",
		"CreateUsage",
		"GetUsageByPromoCodeAndUser",
		"ListUsagesByPromoCode",
		"IncrementUsedCount",
	} {
		t.Run(method, func(t *testing.T) {
			body := promoCodeRepoMethodBody(t, content, method)
			require.Contains(t, body, "client := clientFromContext(ctx, r.client)",
				"%s must reuse dbent.NewTxContext transaction clients", method)
		})
	}
}

func promoCodeRepoMethodBody(t *testing.T, content string, method string) string {
	t.Helper()
	needle := "func (r *promoCodeRepository) " + method + "("
	start := strings.Index(content, needle)
	require.NotEqual(t, -1, start, "method %s not found", method)

	next := strings.Index(content[start+len(needle):], "\nfunc ")
	if next == -1 {
		return content[start:]
	}
	return content[start : start+len(needle)+next]
}
