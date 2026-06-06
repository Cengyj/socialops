package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateTaskTemplateInputRejectsLegacyAndUnavailableTaskTypes(t *testing.T) {
	cases := []string{"tweet", "dm", "message"}
	for _, taskType := range cases {
		t.Run(taskType, func(t *testing.T) {
			result := ValidateTaskTemplateInput(&TaskTemplateInput{
				Name: "legacy action",
				Type: taskType,
				Params: TaskTemplateParams{
					Targets:  []string{"@target"},
					Contents: []string{"hello"},
				},
			})

			require.False(t, result.Valid)
			require.Contains(t, result.Errors[0], "unsupported")
		})
	}
}

func TestValidateTaskTemplateInputRejectsOversizedParameterPools(t *testing.T) {
	targets := make([]string, 501)
	for i := range targets {
		targets[i] = "@target"
	}

	result := ValidateTaskTemplateInput(&TaskTemplateInput{
		Name: "too many targets",
		Type: SocialTaskActionFollow,
		Params: TaskTemplateParams{
			Targets: targets,
		},
	})

	require.False(t, result.Valid)
	require.Contains(t, result.Errors, "target list cannot exceed 500 items")
}

func TestValidateTaskTemplateInputRejectsOverlongParameterValues(t *testing.T) {
	result := ValidateTaskTemplateInput(&TaskTemplateInput{
		Name: "overlong post",
		Type: SocialTaskActionPost,
		Params: TaskTemplateParams{
			Contents: []string{strings.Repeat("a", 2049)},
		},
	})

	require.False(t, result.Valid)
	require.Contains(t, result.Errors, "content item cannot exceed 2048 characters")
}

func TestNormalizeTaskTemplateInputDropsParamsUnusedByTemplateType(t *testing.T) {
	cases := []struct {
		name             string
		templateType     string
		params           TaskTemplateParams
		expectedTargets  []string
		expectedContents []string
	}{
		{
			name:         "login check drops all params",
			templateType: SocialTaskActionLoginCheck,
			params: TaskTemplateParams{
				Targets:  []string{"@target"},
				Contents: []string{"hello"},
			},
		},
		{
			name:         "follow keeps only targets",
			templateType: SocialTaskActionFollow,
			params: TaskTemplateParams{
				Targets:  []string{" @target "},
				Contents: []string{"ignored"},
			},
			expectedTargets: []string{"@target"},
		},
		{
			name:         "post keeps only contents",
			templateType: SocialTaskActionPost,
			params: TaskTemplateParams{
				Targets:  []string{"ignored"},
				Contents: []string{" hello "},
			},
			expectedContents: []string{"hello"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmpl, err := normalizeTaskTemplateInput(&TaskTemplateInput{
				Name:   "template",
				Type:   tc.templateType,
				Params: tc.params,
			})

			require.NoError(t, err)
			require.Equal(t, tc.expectedTargets, tmpl.Params.Targets)
			require.Equal(t, tc.expectedContents, tmpl.Params.Contents)
		})
	}
}
