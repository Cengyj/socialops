//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// ---------- resolveBalanceThreshold ----------

func TestResolveBalanceThreshold_Fixed(t *testing.T) {
	// Fixed type always returns the raw threshold regardless of totalRecharged.
	require.Equal(t, 10.0, resolveBalanceThreshold(10, "fixed", 1000))
	require.Equal(t, 10.0, resolveBalanceThreshold(10, "fixed", 0))
	require.Equal(t, 0.0, resolveBalanceThreshold(0, "fixed", 1000))
}

func TestResolveBalanceThreshold_Percentage(t *testing.T) {
	// 10% of 1000 = 100
	require.Equal(t, 100.0, resolveBalanceThreshold(10, thresholdTypePercentage, 1000))
	// 50% of 200 = 100
	require.Equal(t, 100.0, resolveBalanceThreshold(50, thresholdTypePercentage, 200))
}

func TestResolveBalanceThreshold_PercentageZeroRecharged(t *testing.T) {
	// When totalRecharged is 0, percentage falls through to raw threshold
	// (treated as fixed). This is the defensive behavior.
	require.Equal(t, 10.0, resolveBalanceThreshold(10, thresholdTypePercentage, 0))
}

func TestResolveBalanceThreshold_EmptyType(t *testing.T) {
	// Empty type is treated as fixed (not percentage).
	require.Equal(t, 10.0, resolveBalanceThreshold(10, "", 1000))
}

// ---------- sanitizeEmailHeader ----------

func TestSanitizeEmailHeader_CRLF(t *testing.T) {
	require.Equal(t, "Subject injected", sanitizeEmailHeader("Subject\r\n injected"))
}

func TestSanitizeEmailHeader_OnlyCR(t *testing.T) {
	require.Equal(t, "foobar", sanitizeEmailHeader("foo\rbar"))
}

func TestSanitizeEmailHeader_OnlyLF(t *testing.T) {
	require.Equal(t, "foobar", sanitizeEmailHeader("foo\nbar"))
}

func TestSanitizeEmailHeader_Clean(t *testing.T) {
	require.Equal(t, "SocialOps", sanitizeEmailHeader("SocialOps"))
}

func TestSanitizeEmailHeader_Empty(t *testing.T) {
	require.Equal(t, "", sanitizeEmailHeader(""))
}

func TestSanitizeEmailHeader_MultipleNewlines(t *testing.T) {
	require.Equal(t, "abc", sanitizeEmailHeader("a\r\nb\r\nc"))
}

// ---------- collectBalanceNotifyRecipients ----------

func TestCollectBalanceNotifyRecipients_Empty(t *testing.T) {
	s := &BalanceNotifyService{}
	u := &User{BalanceNotifyExtraEmails: nil}
	require.Empty(t, s.collectBalanceNotifyRecipients(u))
}

func TestCollectBalanceNotifyRecipients_FiltersDisabledAndUnverified(t *testing.T) {
	s := &BalanceNotifyService{}
	u := &User{
		BalanceNotifyExtraEmails: []NotifyEmailEntry{
			{Email: "a@example.com", Verified: true, Disabled: false},
			{Email: "b@example.com", Verified: true, Disabled: true},   // disabled
			{Email: "c@example.com", Verified: false, Disabled: false}, // unverified
			{Email: "d@example.com", Verified: true, Disabled: false},
		},
	}
	got := s.collectBalanceNotifyRecipients(u)
	require.Equal(t, []string{"a@example.com", "d@example.com"}, got)
}

func TestCollectBalanceNotifyRecipients_DeduplicatesCaseInsensitive(t *testing.T) {
	s := &BalanceNotifyService{}
	u := &User{
		BalanceNotifyExtraEmails: []NotifyEmailEntry{
			{Email: "User@Example.com", Verified: true},
			{Email: "user@example.com", Verified: true},
			{Email: "USER@EXAMPLE.COM", Verified: true},
		},
	}
	got := s.collectBalanceNotifyRecipients(u)
	require.Len(t, got, 1)
	// The original casing of the first entry is preserved.
	require.Equal(t, "User@Example.com", got[0])
}

func TestCollectBalanceNotifyRecipients_SkipsEmpty(t *testing.T) {
	s := &BalanceNotifyService{}
	u := &User{
		BalanceNotifyExtraEmails: []NotifyEmailEntry{
			{Email: "  ", Verified: true},
			{Email: "", Verified: true},
			{Email: "valid@example.com", Verified: true},
		},
	}
	got := s.collectBalanceNotifyRecipients(u)
	require.Equal(t, []string{"valid@example.com"}, got)
}

func TestCollectBalanceNotifyRecipients_TrimsWhitespace(t *testing.T) {
	s := &BalanceNotifyService{}
	u := &User{
		BalanceNotifyExtraEmails: []NotifyEmailEntry{
			{Email: "  trimmed@example.com  ", Verified: true},
		},
	}
	got := s.collectBalanceNotifyRecipients(u)
	require.Equal(t, []string{"trimmed@example.com"}, got)
}
