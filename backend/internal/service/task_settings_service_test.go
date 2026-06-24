package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/png"
	"strings"
	"testing"

	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestValidateTaskTemplateInputRejectsTweetAliasAndUnavailableTaskTypes(t *testing.T) {
	cases := []string{"tweet", "unsupported_action"}
	for _, taskType := range cases {
		t.Run(taskType, func(t *testing.T) {
			result := ValidateTaskTemplateInput(&TaskTemplateInput{
				Name: "unsupported action",
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

func TestTaskSettingsServiceValidateTemplateInputUsesExistingValidationRules(t *testing.T) {
	result := (*TaskSettingsService)(nil).ValidateTemplateInput(&TaskTemplateInput{
		Name: "follow targets",
		Type: SocialTaskActionFollow,
		Params: TaskTemplateParams{
			Targets: []string{" @northwind "},
		},
	})

	require.True(t, result.Valid)
	require.Empty(t, result.Errors)
	require.Equal(t, SocialTaskActionFollow, result.Type)
	require.Equal(t, 1, result.Targets)
}

func TestTaskSettingsTemplateIDOperationsRejectBlankIDsConsistently(t *testing.T) {
	ctx := context.Background()
	svc := (*TaskSettingsService)(nil)

	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "get template",
			call: func() error {
				_, err := svc.GetTemplate(ctx, 42, "  ")
				return err
			},
		},
		{
			name: "delete template",
			call: func() error {
				return svc.DeleteTemplate(ctx, 42, "  ")
			},
		},
		{
			name: "copy template",
			call: func() error {
				_, err := svc.CopyTemplate(ctx, 42, "  ")
				return err
			},
		},
		{
			name: "set default template",
			call: func() error {
				_, err := svc.SetDefaultTemplate(ctx, 42, "  ")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()

			require.Error(t, err)
			require.True(t, infraerrors.IsBadRequest(err))
			require.Equal(t, "TASK_TEMPLATE_ID_REQUIRED", infraerrors.Reason(err))
			require.Equal(t, "task template id is required", infraerrors.Message(err))
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

func TestValidateTaskTemplateInputSupportsStructuredTemplateTypes(t *testing.T) {
	profileResult := ValidateTaskTemplateInput(&TaskTemplateInput{
		Name: "Profile refresh",
		Type: SocialTaskActionUpdateProfile,
		Params: TaskTemplateParams{
			Profile: &SocialProfileUpdateParams{
				DisplayName: "Northwind Ops",
				Description: "Operator account",
			},
		},
	})

	require.True(t, profileResult.Valid)
	require.Empty(t, profileResult.Errors)

	avatarResult := ValidateTaskTemplateInput(&TaskTemplateInput{
		Name: "Avatar refresh",
		Type: SocialTaskActionUpdateAvatar,
		Params: TaskTemplateParams{
			Avatar: &SocialTaskMediaRef{
				Source:      "inline",
				ContentType: "image/png",
				URL:         inlinePNGDataURLForTaskTemplateValidation(t, 400, 400),
			},
		},
	})

	require.True(t, avatarResult.Valid)
	require.Empty(t, avatarResult.Errors)

	bannerResult := ValidateTaskTemplateInput(&TaskTemplateInput{
		Name: "Banner refresh",
		Type: SocialTaskActionUpdateBanner,
		Params: TaskTemplateParams{
			Banner: &SocialTaskMediaRef{
				Source:      "inline",
				ContentType: "image/png",
				URL:         inlinePNGDataURLForTaskTemplateValidation(t, 1500, 500),
			},
		},
	})

	require.True(t, bannerResult.Valid)
	require.Empty(t, bannerResult.Errors)
}

func TestValidateTaskTemplateInputSupportsMediaOnlyPostTemplates(t *testing.T) {
	mediaOnlyResult := ValidateTaskTemplateInput(&TaskTemplateInput{
		Name: "Media only post",
		Type: SocialTaskActionPost,
		Params: TaskTemplateParams{
			Media: []SocialTaskMediaRef{{
				Source:      "inline",
				ContentType: "image/png",
				URL:         inlinePNGDataURLForTaskTemplateValidation(t, 1200, 675),
				FileName:    "launch-card.png",
				Width:       1200,
				Height:      675,
			}},
		},
	})

	require.True(t, mediaOnlyResult.Valid)
	require.Empty(t, mediaOnlyResult.Errors)
}

func TestValidateTaskTemplateInputRejectsTooManyPostMediaItems(t *testing.T) {
	media := make([]SocialTaskMediaRef, 5)
	for i := range media {
		media[i] = SocialTaskMediaRef{
			Source:      "inline",
			ContentType: "image/png",
			URL:         inlinePNGDataURLForTaskTemplateValidation(t, 640, 640),
			FileName:    "post-image.png",
			Width:       640,
			Height:      640,
		}
	}

	result := ValidateTaskTemplateInput(&TaskTemplateInput{
		Name: "Too many post images",
		Type: SocialTaskActionPost,
		Params: TaskTemplateParams{
			Contents: []string{"hello media"},
			Media:    media,
		},
	})

	require.False(t, result.Valid)
	require.Contains(t, result.Errors, "post media cannot exceed 4 items")
}

func TestValidateTaskTemplateInputRejectsQuoteOnlyPostTemplates(t *testing.T) {
	quoteOnlyResult := ValidateTaskTemplateInput(&TaskTemplateInput{
		Name: "Quote only post",
		Type: SocialTaskActionPost,
		Params: TaskTemplateParams{
			QuotePostURL: "https://x.com/northwind/status/1",
		},
	})

	require.False(t, quoteOnlyResult.Valid)
	require.Contains(t, quoteOnlyResult.Errors, "post template requires content pool or media")
}

func TestValidateTaskTemplateInputRejectsSingleVideoPostMediaTemplates(t *testing.T) {
	videoResult := ValidateTaskTemplateInput(&TaskTemplateInput{
		Name: "Video post",
		Type: SocialTaskActionPost,
		Params: TaskTemplateParams{
			Contents: []string{"hello video"},
			Media: []SocialTaskMediaRef{{
				Source:      "inline",
				ContentType: "video/mp4",
				FileName:    "clip.mp4",
				URL:         "data:video/mp4;base64,QUJD",
			}},
		},
	})
	require.False(t, videoResult.Valid)
	require.Contains(t, videoResult.Errors, "video media is not supported for SocialOps execution")
}

func TestValidateTaskTemplateInputRejectsPostMediaWithUnsupportedContentTypes(t *testing.T) {
	unsupportedResult := ValidateTaskTemplateInput(&TaskTemplateInput{
		Name: "Unsupported post media",
		Type: SocialTaskActionPost,
		Params: TaskTemplateParams{
			Contents: []string{"hello file"},
			Media: []SocialTaskMediaRef{{
				Source:      "inline",
				ContentType: "application/pdf",
				FileName:    "spec.pdf",
				URL:         "data:application/pdf;base64,QUJD",
			}},
		},
	})
	require.False(t, unsupportedResult.Valid)
	require.Contains(t, unsupportedResult.Errors, "post media content type is not supported")
}

func TestValidateTaskTemplateInputRejectsNonExecutableMediaLibraryRefs(t *testing.T) {
	postResult := ValidateTaskTemplateInput(&TaskTemplateInput{
		Name: "Library post",
		Type: SocialTaskActionPost,
		Params: TaskTemplateParams{
			Contents: []string{"hello library"},
			Media: []SocialTaskMediaRef{{
				Source:      "library",
				StorageKey:  "media/post-image.jpg",
				ContentType: "image/jpeg",
			}},
		},
	})
	require.False(t, postResult.Valid)
	require.Contains(t, postResult.Errors, "post media #1 media source is not supported for SocialOps execution")

	avatarResult := ValidateTaskTemplateInput(&TaskTemplateInput{
		Name: "Avatar refresh",
		Type: SocialTaskActionUpdateAvatar,
		Params: TaskTemplateParams{
			Avatar: &SocialTaskMediaRef{
				Source:      "library",
				StorageKey:  "media/avatar.jpg",
				ContentType: "image/jpeg",
				Width:       400,
				Height:      400,
			},
		},
	})
	require.False(t, avatarResult.Valid)
	require.Contains(t, avatarResult.Errors, "avatar media source is not supported for SocialOps execution")

	bannerResult := ValidateTaskTemplateInput(&TaskTemplateInput{
		Name: "Banner refresh",
		Type: SocialTaskActionUpdateBanner,
		Params: TaskTemplateParams{
			Banner: &SocialTaskMediaRef{
				Source:      "library",
				StorageKey:  "media/banner.jpg",
				ContentType: "image/jpeg",
				Width:       1500,
				Height:      500,
			},
		},
	})
	require.False(t, bannerResult.Valid)
	require.Contains(t, bannerResult.Errors, "banner media source is not supported for SocialOps execution")
}

func TestValidateTaskTemplateInputAcceptsInternalTaskMediaLibraryRefs(t *testing.T) {
	postResult := ValidateTaskTemplateInput(&TaskTemplateInput{
		Name: "Stored post",
		Type: SocialTaskActionPost,
		Params: TaskTemplateParams{
			Contents: []string{"hello library"},
			Media: []SocialTaskMediaRef{{
				Source:      "library",
				StorageKey:  "social-task/42/post-image.jpg",
				ContentType: "image/jpeg",
			}},
		},
	})
	require.True(t, postResult.Valid)
	require.Empty(t, postResult.Errors)

	avatarResult := ValidateTaskTemplateInput(&TaskTemplateInput{
		Name: "Stored avatar",
		Type: SocialTaskActionUpdateAvatar,
		Params: TaskTemplateParams{
			Avatar: &SocialTaskMediaRef{
				Source:      "library",
				StorageKey:  "social-task/42/avatar.png",
				ContentType: "image/png",
				Width:       400,
				Height:      400,
			},
		},
	})
	require.True(t, avatarResult.Valid)
	require.Empty(t, avatarResult.Errors)

	bannerResult := ValidateTaskTemplateInput(&TaskTemplateInput{
		Name: "Stored banner",
		Type: SocialTaskActionUpdateBanner,
		Params: TaskTemplateParams{
			Banner: &SocialTaskMediaRef{
				Source:      "library",
				StorageKey:  "social-task/42/banner.png",
				ContentType: "image/png",
				Width:       1500,
				Height:      500,
			},
		},
	})
	require.True(t, bannerResult.Valid)
	require.Empty(t, bannerResult.Errors)
}

func TestValidateTaskTemplateInputRejectsAvatarAndBannerThatNeedNormalization(t *testing.T) {
	avatarResult := ValidateTaskTemplateInput(&TaskTemplateInput{
		Name: "Avatar refresh",
		Type: SocialTaskActionUpdateAvatar,
		Params: TaskTemplateParams{
			Avatar: &SocialTaskMediaRef{
				Source:      "inline",
				ContentType: "image/png",
				URL:         inlinePNGDataURLForTaskTemplateValidation(t, 300, 300),
			},
		},
	})
	require.False(t, avatarResult.Valid)
	require.Contains(t, avatarResult.Errors, "avatar image must be 400x400 pixels")

	bannerResult := ValidateTaskTemplateInput(&TaskTemplateInput{
		Name: "Banner refresh",
		Type: SocialTaskActionUpdateBanner,
		Params: TaskTemplateParams{
			Banner: &SocialTaskMediaRef{
				Source:      "inline",
				ContentType: "image/png",
				URL:         inlinePNGDataURLForTaskTemplateValidation(t, 1200, 500),
			},
		},
	})
	require.False(t, bannerResult.Valid)
	require.Contains(t, bannerResult.Errors, "banner image must be 1500x500 pixels")
}

func TestValidateTaskTemplateInputAcceptsExactAvatarAndBannerImageDimensions(t *testing.T) {
	avatarResult := ValidateTaskTemplateInput(&TaskTemplateInput{
		Name: "Avatar refresh",
		Type: SocialTaskActionUpdateAvatar,
		Params: TaskTemplateParams{
			Avatar: &SocialTaskMediaRef{
				Source:      "inline",
				ContentType: "image/png",
				URL:         inlinePNGDataURLForTaskTemplateValidation(t, 400, 400),
			},
		},
	})
	require.True(t, avatarResult.Valid)
	require.Empty(t, avatarResult.Errors)

	bannerResult := ValidateTaskTemplateInput(&TaskTemplateInput{
		Name: "Banner refresh",
		Type: SocialTaskActionUpdateBanner,
		Params: TaskTemplateParams{
			Banner: &SocialTaskMediaRef{
				Source:      "inline",
				ContentType: "image/png",
				URL:         inlinePNGDataURLForTaskTemplateValidation(t, 1500, 500),
			},
		},
	})
	require.True(t, bannerResult.Valid)
	require.Empty(t, bannerResult.Errors)
}

func TestValidateTaskTemplateInputUsesActualInlineImageDimensionsOverSuppliedMetadata(t *testing.T) {
	avatarResult := ValidateTaskTemplateInput(&TaskTemplateInput{
		Name: "Avatar refresh",
		Type: SocialTaskActionUpdateAvatar,
		Params: TaskTemplateParams{
			Avatar: &SocialTaskMediaRef{
				Source:      "inline",
				ContentType: "image/png",
				URL:         inlinePNGDataURLForTaskTemplateValidation(t, 300, 300),
				Width:       400,
				Height:      400,
			},
		},
	})
	require.False(t, avatarResult.Valid)
	require.Contains(t, avatarResult.Errors, "avatar image must be 400x400 pixels")

	bannerResult := ValidateTaskTemplateInput(&TaskTemplateInput{
		Name: "Banner refresh",
		Type: SocialTaskActionUpdateBanner,
		Params: TaskTemplateParams{
			Banner: &SocialTaskMediaRef{
				Source:      "inline",
				ContentType: "image/png",
				URL:         inlinePNGDataURLForTaskTemplateValidation(t, 1400, 500),
				Width:       1500,
				Height:      500,
			},
		},
	})
	require.False(t, bannerResult.Valid)
	require.Contains(t, bannerResult.Errors, "banner image must be 1500x500 pixels")
}

func TestValidateTaskTemplateInputRejectsExternalLibraryAvatarAndBannerWithoutExactDimensionsMetadata(t *testing.T) {
	avatarResult := ValidateTaskTemplateInput(&TaskTemplateInput{
		Name: "Avatar refresh",
		Type: SocialTaskActionUpdateAvatar,
		Params: TaskTemplateParams{
			Avatar: &SocialTaskMediaRef{
				Source:      "library",
				StorageKey:  "media/avatar.jpg",
				ContentType: "image/jpeg",
			},
		},
	})
	require.False(t, avatarResult.Valid)
	require.Contains(t, avatarResult.Errors, "avatar media source is not supported for SocialOps execution")

	bannerResult := ValidateTaskTemplateInput(&TaskTemplateInput{
		Name: "Banner refresh",
		Type: SocialTaskActionUpdateBanner,
		Params: TaskTemplateParams{
			Banner: &SocialTaskMediaRef{
				Source:      "library",
				StorageKey:  "media/banner.jpg",
				ContentType: "image/jpeg",
				Width:       1500,
			},
		},
	})
	require.False(t, bannerResult.Valid)
	require.Contains(t, bannerResult.Errors, "banner media source is not supported for SocialOps execution")
}

func TestValidateTaskTemplateInputRejectsInternalLibraryAvatarAndBannerWithoutExactDimensionsMetadata(t *testing.T) {
	avatarResult := ValidateTaskTemplateInput(&TaskTemplateInput{
		Name: "Avatar refresh",
		Type: SocialTaskActionUpdateAvatar,
		Params: TaskTemplateParams{
			Avatar: &SocialTaskMediaRef{
				Source:      "library",
				StorageKey:  "social-task/42/avatar.jpg",
				ContentType: "image/jpeg",
			},
		},
	})
	require.False(t, avatarResult.Valid)
	require.Contains(t, avatarResult.Errors, "avatar image must be 400x400 pixels")

	bannerResult := ValidateTaskTemplateInput(&TaskTemplateInput{
		Name: "Banner refresh",
		Type: SocialTaskActionUpdateBanner,
		Params: TaskTemplateParams{
			Banner: &SocialTaskMediaRef{
				Source:      "library",
				StorageKey:  "social-task/42/banner.jpg",
				ContentType: "image/jpeg",
				Width:       1500,
			},
		},
	})
	require.False(t, bannerResult.Valid)
	require.Contains(t, bannerResult.Errors, "banner image must be 1500x500 pixels")
}

func TestNormalizeTaskTemplateInputPreservesStructuredPostAndProfileParams(t *testing.T) {
	tmpl, err := normalizeTaskTemplateInput(&TaskTemplateInput{
		Name: "Structured post",
		Type: SocialTaskActionPost,
		Params: TaskTemplateParams{
			Targets:      []string{"ignored"},
			Contents:     []string{" hello "},
			QuotePostURL: " https://x.com/northwind/status/1 ",
			Media: []SocialTaskMediaRef{
				{
					Source:      "library",
					StorageKey:  "social-task/media/post-image.jpg",
					ContentType: "image/jpeg",
				},
			},
			Profile: &SocialProfileUpdateParams{
				DisplayName: "ignored",
			},
		},
	})

	require.NoError(t, err)
	require.Empty(t, tmpl.Params.Targets)
	require.Equal(t, []string{"hello"}, tmpl.Params.Contents)
	require.Equal(t, "https://x.com/northwind/status/1", tmpl.Params.QuotePostURL)
	require.Len(t, tmpl.Params.Media, 1)
	require.Nil(t, tmpl.Params.Profile)

	profileTemplate, err := normalizeTaskTemplateInput(&TaskTemplateInput{
		Name: "Structured profile",
		Type: SocialTaskActionUpdateProfile,
		Params: TaskTemplateParams{
			Contents: []string{"ignored"},
			Profile: &SocialProfileUpdateParams{
				DisplayName: " Northwind Ops ",
				Location:    " Singapore ",
			},
			Avatar: &SocialTaskMediaRef{
				StorageKey: "ignored",
			},
		},
	})

	require.NoError(t, err)
	require.Empty(t, profileTemplate.Params.Contents)
	require.NotNil(t, profileTemplate.Params.Profile)
	require.Equal(t, "Northwind Ops", profileTemplate.Params.Profile.DisplayName)
	require.Equal(t, "Singapore", profileTemplate.Params.Profile.Location)
	require.Nil(t, profileTemplate.Params.Avatar)
}

func inlinePNGDataURLForTaskTemplateValidation(t *testing.T, width, height int) string {
	t.Helper()

	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	require.NoError(t, png.Encode(&buf, img))
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}
