//go:build unit

package service

import (
	"context"
	"encoding/json"
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
		Name:      "Valid login check",
		Type:      SocialTaskActionLoginCheck,
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

func TestTaskSettingsLoadDropsParamsUnusedByTemplateType(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewTaskSettingsService(client)
	userID := int64(43)
	now := time.Now().UTC()
	raw, err := json.Marshal(taskTemplateDocument{Templates: []*TaskTemplate{
		{
			ID:   "tmpl_login",
			Name: "Login check",
			Type: SocialTaskActionLoginCheck,
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
	require.Len(t, templates, 2)
	require.Empty(t, templates[0].Params.Targets)
	require.Empty(t, templates[0].Params.Contents)
	require.Empty(t, templates[1].Params.Targets)
	require.Equal(t, []string{"hello"}, templates[1].Params.Contents)
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
