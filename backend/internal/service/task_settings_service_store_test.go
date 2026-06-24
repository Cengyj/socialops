//go:build unit

package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"strings"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestTaskSettingsSetDefaultRejectsInvalidStoredTemplate(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewTaskSettingsService(client)
	userID := int64(42)
	now := time.Now().UTC()
	valid := &TaskTemplate{
		ID:        "tmpl_valid",
		Name:      "Valid follow",
		Type:      SocialTaskActionFollow,
		Params:    TaskTemplateParams{Targets: []string{"@valid"}},
		IsDefault: true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	invalid := &TaskTemplate{
		ID:        "tmpl_invalid",
		Name:      "Invalid follow",
		Type:      SocialTaskActionFollow,
		Params:    TaskTemplateParams{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	raw, err := json.Marshal(taskTemplateDocument{Templates: []*TaskTemplate{valid, invalid}})
	require.NoError(t, err)
	client.Setting.Create().
		SetKey(taskSettingsKey(userID)).
		SetValue(string(raw)).
		SaveX(ctx)

	_, err = svc.SetDefaultTemplate(ctx, userID, invalid.ID)

	require.Error(t, err)
	require.Equal(t, "TASK_TEMPLATE_INVALID", infraerrors.Reason(err))
	templates, listErr := svc.ListTemplates(ctx, userID)
	require.NoError(t, listErr)
	require.True(t, templates[0].IsDefault)
	require.False(t, templates[1].IsDefault)
}

func TestTaskSettingsLoadFiltersUnsupportedTypesAndDropsParamsUnusedByTemplateType(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewTaskSettingsService(client)
	userID := int64(43)
	now := time.Now().UTC()
	raw, err := json.Marshal(taskTemplateDocument{Templates: []*TaskTemplate{
		{
			ID:   "tmpl_unsupported",
			Name: "Unsupported no-parameter action",
			Type: "unsupported_zero_parameter_action",
			Params: TaskTemplateParams{
				Targets:  []string{"@stale"},
				Contents: []string{"stale content"},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:   "tmpl_post",
			Name: "Post",
			Type: SocialTaskActionPost,
			Params: TaskTemplateParams{
				Targets:  []string{"@ignored"},
				Contents: []string{" hello "},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}})
	require.NoError(t, err)
	client.Setting.Create().
		SetKey(taskSettingsKey(userID)).
		SetValue(string(raw)).
		SaveX(ctx)

	templates, err := svc.ListTemplates(ctx, userID)

	require.NoError(t, err)
	require.Len(t, templates, 1)
	require.Empty(t, templates[0].Params.Targets)
	require.Equal(t, []string{"hello"}, templates[0].Params.Contents)
}

func TestTaskSettingsListTemplatesReturnsEmptySliceForNewUser(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewTaskSettingsService(client)

	templates, err := svc.ListTemplates(ctx, 404)

	require.NoError(t, err)
	require.NotNil(t, templates)
	require.Empty(t, templates)
	raw, err := json.Marshal(templates)
	require.NoError(t, err)
	require.Equal(t, "[]", string(raw))
}

func TestTaskSettingsApplyDefaultTemplateToTaskInputExpandsDefaultTemplate(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewTaskSettingsService(client)
	userID := int64(45)

	tmpl, err := svc.SaveTemplate(ctx, userID, &TaskTemplateInput{
		Name: "Daily follows",
		Type: SocialTaskActionFollow,
		Params: TaskTemplateParams{
			Targets: []string{" @northwind ", "@contoso"},
		},
		IsDefault: true,
	})
	require.NoError(t, err)

	input := &AccountWorkbenchTaskInput{Action: " follow "}
	err = svc.ApplyDefaultTemplateToTaskInput(ctx, userID, input)

	require.NoError(t, err)
	require.Equal(t, SocialTaskActionFollow, input.Action)
	require.Equal(t, []string{"@northwind", "@contoso"}, input.TargetPool)
	require.Empty(t, input.ContentPool)
	require.Nil(t, input.Payload)
	require.NotNil(t, input.TemplateSnapshot)
	require.Equal(t, tmpl.ID, input.TemplateSnapshot.TemplateID)
	require.Equal(t, "Daily follows", input.TemplateSnapshot.TemplateName)
	require.Equal(t, SocialTaskActionFollow, input.TemplateSnapshot.TemplateType)
	require.Equal(t, []string{"@northwind", "@contoso"}, input.TemplateSnapshot.Params.Targets)
}

func TestTaskSettingsApplyDefaultTemplateToTaskInputSkipsLoginCheckWithoutService(t *testing.T) {
	input := &AccountWorkbenchTaskInput{Action: " login_check "}

	err := (*TaskSettingsService)(nil).ApplyDefaultTemplateToTaskInput(context.Background(), 46, input)

	require.NoError(t, err)
	require.Equal(t, SocialTaskActionLoginCheck, input.Action)
	require.Nil(t, input.TemplateSnapshot)
}

func TestTaskSettingsSaveTemplateMaterializesInlinePostMediaIntoTaskMediaAssets(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewTaskSettingsService(client)
	userID := int64(88)

	saved, err := svc.SaveTemplate(ctx, userID, &TaskTemplateInput{
		Name: "Materialized post",
		Type: SocialTaskActionPost,
		Params: TaskTemplateParams{
			Contents: []string{"hello media"},
			Media: []SocialTaskMediaRef{{
				Source:      "inline",
				ContentType: "image/png",
				FileName:    "post-inline.png",
				URL:         inlinePNGDataURLForTaskTemplateValidation(t, 640, 640),
			}},
		},
		IsDefault: true,
	})

	require.NoError(t, err)
	require.Len(t, saved.Params.Media, 1)
	require.Equal(t, "library", saved.Params.Media[0].Source)
	require.True(t, strings.HasPrefix(saved.Params.Media[0].StorageKey, "social-task/"))
	require.Empty(t, saved.Params.Media[0].URL)
	require.Equal(t, "image/png", saved.Params.Media[0].ContentType)
	require.Equal(t, "post-inline.png", saved.Params.Media[0].FileName)
	require.Equal(t, 640, saved.Params.Media[0].Width)
	require.Equal(t, 640, saved.Params.Media[0].Height)

	stored, err := svc.GetTemplate(ctx, userID, saved.ID)
	require.NoError(t, err)
	require.Len(t, stored.Params.Media, 1)
	require.Equal(t, "library", stored.Params.Media[0].Source)
	require.Equal(t, saved.Params.Media[0].StorageKey, stored.Params.Media[0].StorageKey)
	require.Empty(t, stored.Params.Media[0].URL)

	rows, err := client.QueryContext(ctx, `
SELECT user_id, storage_provider, storage_key, url, content_type, file_name, byte_size, width, height
FROM social_task_media_assets`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	require.True(t, rows.Next())
	var storedUserID int64
	var provider string
	var storageKey string
	var rawURL string
	var contentType string
	var fileName string
	var byteSize int64
	var width int
	var height int
	require.NoError(t, rows.Scan(&storedUserID, &provider, &storageKey, &rawURL, &contentType, &fileName, &byteSize, &width, &height))
	require.Equal(t, userID, storedUserID)
	require.Equal(t, "inline", provider)
	require.Equal(t, saved.Params.Media[0].StorageKey, storageKey)
	require.Contains(t, rawURL, "data:image/png;base64,")
	require.Equal(t, "image/png", contentType)
	require.Equal(t, "post-inline.png", fileName)
	require.Positive(t, byteSize)
	require.Equal(t, 640, width)
	require.Equal(t, 640, height)
	require.False(t, rows.Next())
}

func TestTaskSettingsSaveTemplateMaterializesInlineProfileMediaIntoTaskMediaAssets(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewTaskSettingsService(client)
	userID := int64(89)

	avatar, err := svc.SaveTemplate(ctx, userID, &TaskTemplateInput{
		Name: "Materialized avatar",
		Type: SocialTaskActionUpdateAvatar,
		Params: TaskTemplateParams{
			Avatar: &SocialTaskMediaRef{
				Source:      "inline",
				ContentType: "image/png",
				FileName:    "avatar-inline.png",
				URL:         inlinePNGDataURLForTaskTemplateValidation(t, 400, 400),
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, avatar.Params.Avatar)
	require.Equal(t, "library", avatar.Params.Avatar.Source)
	require.True(t, strings.HasPrefix(avatar.Params.Avatar.StorageKey, "social-task/"))
	require.Empty(t, avatar.Params.Avatar.URL)
	require.Equal(t, 400, avatar.Params.Avatar.Width)
	require.Equal(t, 400, avatar.Params.Avatar.Height)

	banner, err := svc.SaveTemplate(ctx, userID, &TaskTemplateInput{
		Name: "Materialized banner",
		Type: SocialTaskActionUpdateBanner,
		Params: TaskTemplateParams{
			Banner: &SocialTaskMediaRef{
				Source:      "inline",
				ContentType: "image/png",
				FileName:    "banner-inline.png",
				URL:         inlinePNGDataURLForTaskTemplateValidation(t, 1500, 500),
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, banner.Params.Banner)
	require.Equal(t, "library", banner.Params.Banner.Source)
	require.True(t, strings.HasPrefix(banner.Params.Banner.StorageKey, "social-task/"))
	require.Empty(t, banner.Params.Banner.URL)
	require.Equal(t, 1500, banner.Params.Banner.Width)
	require.Equal(t, 500, banner.Params.Banner.Height)

	rows, err := client.QueryContext(ctx, `
SELECT storage_provider, storage_key, file_name, width, height
FROM social_task_media_assets
ORDER BY id ASC`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var providers []string
	var storageKeys []string
	var fileNames []string
	var widths []int
	var heights []int
	for rows.Next() {
		var provider string
		var storageKey string
		var fileName string
		var width int
		var height int
		require.NoError(t, rows.Scan(&provider, &storageKey, &fileName, &width, &height))
		providers = append(providers, provider)
		storageKeys = append(storageKeys, storageKey)
		fileNames = append(fileNames, fileName)
		widths = append(widths, width)
		heights = append(heights, height)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []string{"inline", "inline"}, providers)
	require.Len(t, storageKeys, 2)
	require.True(t, strings.HasPrefix(storageKeys[0], "social-task/"))
	require.True(t, strings.HasPrefix(storageKeys[1], "social-task/"))
	require.Equal(t, []string{"avatar-inline.png", "banner-inline.png"}, fileNames)
	require.Equal(t, []int{400, 1500}, widths)
	require.Equal(t, []int{400, 500}, heights)
}

func TestTaskSettingsPreviewTemplateMediaResolvesStoredTaskAsset(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewTaskSettingsService(client)
	userID := int64(90)

	dataURL, rawPNG := taskSettingsPreviewPNG(t, 400, 400)
	saved, err := svc.SaveTemplate(ctx, userID, &TaskTemplateInput{
		Name: "Preview avatar",
		Type: SocialTaskActionUpdateAvatar,
		Params: TaskTemplateParams{
			Avatar: &SocialTaskMediaRef{
				Source:      "inline",
				ContentType: "image/png",
				FileName:    "preview-avatar.png",
				URL:         dataURL,
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, saved.Params.Avatar)

	resolved, err := svc.PreviewTemplateMedia(ctx, userID, saved.Params.Avatar.StorageKey)

	require.NoError(t, err)
	require.Equal(t, "image/png", resolved.ContentType)
	require.Equal(t, "preview-avatar.png", resolved.FileName)
	require.Equal(t, rawPNG, resolved.Body)
	require.Equal(t, 400, resolved.Width)
	require.Equal(t, 400, resolved.Height)
}

func TestTaskSettingsPreviewTemplateMediaRejectsExternalLibraryRefs(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewTaskSettingsService(client)

	_, err := svc.PreviewTemplateMedia(ctx, 91, "media/avatar.jpg")

	require.Error(t, err)
	require.Equal(t, "TASK_TEMPLATE_MEDIA_SOURCE_UNSUPPORTED", infraerrors.Reason(err))
}

func taskSettingsPreviewPNG(t *testing.T, width, height int) (string, []byte) {
	t.Helper()

	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	require.NoError(t, png.Encode(&buf, img))
	body := buf.Bytes()
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(body), body
}
