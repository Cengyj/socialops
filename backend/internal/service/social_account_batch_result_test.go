//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddSocialAccountBatchSuccessRecordsSucceededItem(t *testing.T) {
	result := &SocialAccountBatchResult{Total: 2}

	addSocialAccountBatchSuccess(result, 42, "@northwind_ops")
	addSocialAccountBatchSuccess(nil, 99, "@ignored")

	require.Equal(t, 2, result.Total)
	require.Equal(t, 1, result.Succeeded)
	require.Zero(t, result.Skipped)
	require.Zero(t, result.Failed)
	require.Empty(t, result.Errors)
	require.Equal(t, []SocialAccountBatchItemResult{{
		ID:     42,
		Name:   "@northwind_ops",
		Status: "succeeded",
	}}, result.Items)
}

func TestSocialAccountBatchIDTrackerReportsDuplicateIDs(t *testing.T) {
	tracker := newSocialAccountBatchIDTracker(2)

	require.True(t, tracker.record(42))
	require.False(t, tracker.record(42))
	require.True(t, tracker.record(43))
}
