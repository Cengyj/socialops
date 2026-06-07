package repository

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAffiliateUserOverviewSQLIncludesMaturedFrozenQuota(t *testing.T) {
	query := strings.Join(strings.Fields(affiliateUserOverviewSQL), " ")

	require.Contains(t, query, "ua.aff_quota + COALESCE(matured.matured_frozen_quota, 0)")
	require.Contains(t, query, "frozen_until <= NOW()")
}

func TestAffiliateRecordQueriesUseLedgerAuditFields(t *testing.T) {
	source, err := os.ReadFile("affiliate_repo.go")
	require.NoError(t, err)
	content := string(source)

	require.Contains(t, content, "JOIN payment_orders po ON po.id = ual.source_order_id")
	require.Contains(t, content, "po.provider_snapshot->>'currency'")
	require.Contains(t, content, "item.Currency = normalizeAffiliatePaymentCurrency")
	require.Contains(t, content, "ual.amount::double precision")
	require.Contains(t, content, "ual.balance_after::double precision")
	require.NotContains(t, content, "parseAffiliateRebateAmount")
	require.NotContains(t, content, "ProviderSnapshot")
	require.NotContains(t, content, `"current_balance": "u.balance"`)
}

func TestAffiliateAccrueQuotaCapsInsideTransaction(t *testing.T) {
	source, err := os.ReadFile("affiliate_repo.go")
	require.NoError(t, err)
	content := string(source)

	body := affiliateRepoMethodBody(t, content, "AccrueQuota")
	require.Contains(t, body, "FOR UPDATE")
	require.Contains(t, body, "source_user_id = $2")
	require.Contains(t, body, "LEAST($3::double precision, $4::double precision - existing.total)")
	require.Contains(t, body, "roundAffiliateAmount(amount, 8)")
}

func affiliateRepoMethodBody(t *testing.T, content string, method string) string {
	t.Helper()
	needle := "func (r *affiliateRepository) " + method + "("
	start := strings.Index(content, needle)
	require.NotEqual(t, -1, start, "method %s not found", method)

	next := strings.Index(content[start+len(needle):], "\nfunc ")
	if next == -1 {
		return content[start:]
	}
	return content[start : start+len(needle)+next]
}
