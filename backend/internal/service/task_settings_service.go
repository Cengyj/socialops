package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/setting"
	"github.com/Wei-Shaw/socialops/internal/domain"
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
)

const taskSettingsKeyPrefix = "socialops:task_settings:user:"

const (
	maxTaskTemplatePoolValues  = 500
	maxTaskTemplateValueLength = 2048
)

// TaskTemplateParams stores action-specific parameter pools.
type TaskTemplateParams = domain.SocialTaskTemplateParams

type SocialTaskMediaRef = domain.SocialTaskMediaRef

type SocialProfileUpdateParams = domain.SocialProfileUpdateParams

type SocialTaskPayload = domain.SocialTaskPayload

type SocialPostPayload = domain.SocialPostPayload

type SocialTaskTemplateSnapshot = domain.SocialTaskTemplateSnapshot

// TaskTemplate stores one current-user execution parameter template.
type TaskTemplate struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Type      string             `json:"type"`
	Params    TaskTemplateParams `json:"params"`
	IsDefault bool               `json:"is_default"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
}

type TaskTemplateInput struct {
	ID        string             `json:"id,omitempty"`
	Name      string             `json:"name"`
	Type      string             `json:"type"`
	Params    TaskTemplateParams `json:"params"`
	IsDefault bool               `json:"is_default"`
}

type TaskTemplateValidationResult struct {
	Valid    bool     `json:"valid"`
	Type     string   `json:"type"`
	Targets  int      `json:"targets"`
	Contents int      `json:"contents"`
	Errors   []string `json:"errors"`
}

type taskTemplateDocument struct {
	Templates []*TaskTemplate `json:"templates"`
}

type TaskSettingsService struct {
	entClient *dbent.Client
	taskMedia *SocialTaskMediaService
}

func NewTaskSettingsService(entClient *dbent.Client) *TaskSettingsService {
	return &TaskSettingsService{
		entClient: entClient,
		taskMedia: NewSocialTaskMediaService(entClient),
	}
}

func (s *TaskSettingsService) ListTemplates(ctx context.Context, userID int64) ([]*TaskTemplate, error) {
	doc, err := s.load(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(doc.Templates) == 0 {
		return []*TaskTemplate{}, nil
	}
	return doc.Templates, nil
}

func (s *TaskSettingsService) GetTemplate(ctx context.Context, userID int64, id string) (*TaskTemplate, error) {
	id, err := normalizeTaskTemplateID(id)
	if err != nil {
		return nil, err
	}
	doc, err := s.load(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, tmpl := range doc.Templates {
		if tmpl != nil && tmpl.ID == id {
			return cloneTaskTemplate(tmpl), nil
		}
	}
	return nil, infraerrors.NotFound("TASK_TEMPLATE_NOT_FOUND", "task template not found")
}

func (s *TaskSettingsService) GetDefaultTemplate(ctx context.Context, userID int64, templateType string) (*TaskTemplate, error) {
	templateType, err := normalizeTaskTemplateType(templateType)
	if err != nil {
		return nil, err
	}
	doc, err := s.load(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, tmpl := range doc.Templates {
		if tmpl != nil && tmpl.Type == templateType && tmpl.IsDefault {
			return cloneTaskTemplate(tmpl), nil
		}
	}
	return nil, infraerrors.NotFound("TASK_TEMPLATE_NOT_FOUND", "task template not found")
}

func (s *TaskSettingsService) ApplyDefaultTemplateToTaskInput(ctx context.Context, userID int64, input *AccountWorkbenchTaskInput) error {
	if input == nil {
		return infraerrors.BadRequest("SOCIAL_TASK_INPUT_REQUIRED", "social task input is required")
	}
	action, ok := NormalizeSocialTaskAction(input.Action)
	if !ok {
		return ErrSocialTaskUnsupportedAction
	}
	input.Action = action
	if !accountWorkbenchTaskActionRequiresDefaultTemplate(action) {
		return nil
	}
	if s == nil {
		return infraerrors.ServiceUnavailable("TASK_TEMPLATE_SERVICE_UNAVAILABLE", "task template service is unavailable")
	}
	tmpl, err := s.GetDefaultTemplate(ctx, userID, action)
	if err != nil {
		if infraerrors.IsNotFound(err) {
			return infraerrors.BadRequest("TASK_DEFAULT_TEMPLATE_REQUIRED", "default task template is required for this action")
		}
		return err
	}
	if result := ValidateTaskTemplate(tmpl); !result.Valid {
		return infraerrors.BadRequest("TASK_TEMPLATE_INVALID", strings.Join(result.Errors, "; "))
	}
	applyTaskTemplateToAccountWorkbenchInput(input, tmpl)
	return nil
}

func accountWorkbenchTaskActionRequiresDefaultTemplate(action string) bool {
	return action != SocialTaskActionLogin && action != SocialTaskActionLoginCheck
}

func applyTaskTemplateToAccountWorkbenchInput(input *AccountWorkbenchTaskInput, tmpl *TaskTemplate) {
	if input == nil || tmpl == nil {
		return
	}
	cloned := cloneTaskTemplate(tmpl)
	input.Action = cloned.Type
	input.TargetPool = append([]string(nil), cloned.Params.Targets...)
	input.ContentPool = append([]string(nil), cloned.Params.Contents...)
	input.Payload = socialTaskPayloadFromTemplate(cloned)
	input.TemplateSnapshot = &SocialTaskTemplateSnapshot{
		TemplateID:   cloned.ID,
		TemplateName: cloned.Name,
		TemplateType: cloned.Type,
		Params:       cloned.Params,
	}
}

func socialTaskPayloadFromTemplate(tmpl *TaskTemplate) *SocialTaskPayload {
	if tmpl == nil {
		return nil
	}
	switch tmpl.Type {
	case SocialTaskActionPost:
		payload := &SocialTaskPayload{
			Post: &SocialPostPayload{
				QuotePostURL: tmpl.Params.QuotePostURL,
				Media:        append([]SocialTaskMediaRef(nil), tmpl.Params.Media...),
			},
		}
		if payload.IsZero() {
			return nil
		}
		return payload
	case SocialTaskActionUpdateProfile:
		if tmpl.Params.Profile == nil {
			return nil
		}
		profile := *tmpl.Params.Profile
		payload := &SocialTaskPayload{Profile: &profile}
		if payload.IsZero() {
			return nil
		}
		return payload
	case SocialTaskActionUpdateAvatar:
		if tmpl.Params.Avatar == nil {
			return nil
		}
		avatar := *tmpl.Params.Avatar
		payload := &SocialTaskPayload{Avatar: &avatar}
		if payload.IsZero() {
			return nil
		}
		return payload
	case SocialTaskActionUpdateBanner:
		if tmpl.Params.Banner == nil {
			return nil
		}
		banner := *tmpl.Params.Banner
		payload := &SocialTaskPayload{Banner: &banner}
		if payload.IsZero() {
			return nil
		}
		return payload
	default:
		return nil
	}
}

func (s *TaskSettingsService) SaveTemplate(ctx context.Context, userID int64, input *TaskTemplateInput) (*TaskTemplate, error) {
	if input == nil {
		return nil, infraerrors.BadRequest("TASK_TEMPLATE_INPUT_REQUIRED", "task template input is required")
	}
	if dbent.TxFromContext(ctx) != nil {
		return s.saveTemplateWithPreparedMedia(ctx, userID, input)
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)
	saved, err := s.saveTemplateWithPreparedMedia(txCtx, userID, input)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return saved, nil
}

func (s *TaskSettingsService) saveTemplateWithPreparedMedia(ctx context.Context, userID int64, input *TaskTemplateInput) (*TaskTemplate, error) {
	if input == nil {
		return nil, infraerrors.BadRequest("TASK_TEMPLATE_INPUT_REQUIRED", "task template input is required")
	}
	normalized, err := normalizeTaskTemplateInput(input)
	if err != nil {
		return nil, err
	}
	if result := ValidateTaskTemplate(normalized); !result.Valid {
		return nil, infraerrors.BadRequest("TASK_TEMPLATE_INVALID", strings.Join(result.Errors, "; "))
	}
	if s.taskMedia != nil {
		normalized, err = s.materializeTaskTemplateMedia(ctx, userID, normalized)
		if err != nil {
			return nil, err
		}
	}

	doc, err := s.load(ctx, userID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	var saved *TaskTemplate
	for index, tmpl := range doc.Templates {
		if tmpl == nil || tmpl.ID != normalized.ID || normalized.ID == "" {
			continue
		}
		saved = normalized
		saved.CreatedAt = tmpl.CreatedAt
		saved.UpdatedAt = now
		doc.Templates[index] = saved
		break
	}
	if saved == nil {
		saved = normalized
		if saved.ID == "" {
			saved.ID = fmt.Sprintf("tmpl_%d", now.UnixNano())
		}
		saved.CreatedAt = now
		saved.UpdatedAt = now
		doc.Templates = append(doc.Templates, saved)
	}
	if saved.IsDefault {
		clearTaskTemplateDefaults(doc, saved.Type, saved.ID)
	}
	if err := s.save(ctx, userID, doc); err != nil {
		return nil, err
	}
	return cloneTaskTemplate(saved), nil
}

func (s *TaskSettingsService) materializeTaskTemplateMedia(ctx context.Context, userID int64, tmpl *TaskTemplate) (*TaskTemplate, error) {
	if tmpl == nil {
		return nil, nil
	}
	snapshot := &domain.SocialTaskTemplateSnapshot{
		TemplateID:   strings.TrimSpace(tmpl.ID),
		TemplateName: strings.TrimSpace(tmpl.Name),
		TemplateType: strings.TrimSpace(tmpl.Type),
		Params:       cloneTaskTemplate(tmpl).Params,
	}
	materialized, err := s.taskMedia.materializeTemplateSnapshot(ctx, userID, snapshot, make(map[string]domain.SocialTaskMediaRef))
	if err != nil {
		return nil, err
	}
	if materialized == nil {
		return cloneTaskTemplate(tmpl), nil
	}
	cloned := cloneTaskTemplate(tmpl)
	cloned.Params = materialized.Params
	return cloned, nil
}

func (s *TaskSettingsService) PreviewTemplateMedia(ctx context.Context, userID int64, storageKey string) (*ResolvedSocialTaskMedia, error) {
	if s == nil || s.taskMedia == nil {
		return nil, infraerrors.ServiceUnavailable("TASK_TEMPLATE_MEDIA_SERVICE_UNAVAILABLE", "task template media service is unavailable")
	}
	storageKey = strings.TrimSpace(storageKey)
	if storageKey == "" {
		return nil, infraerrors.BadRequest("TASK_TEMPLATE_MEDIA_STORAGE_KEY_REQUIRED", "task template media storage key is required")
	}
	if !strings.HasPrefix(strings.ToLower(storageKey), "social-task/") {
		return nil, infraerrors.BadRequest("TASK_TEMPLATE_MEDIA_SOURCE_UNSUPPORTED", "task template media source is not supported")
	}
	return s.taskMedia.Resolve(ctx, userID, &domain.SocialTaskMediaRef{
		Source:     "library",
		StorageKey: storageKey,
	})
}

func (s *TaskSettingsService) DeleteTemplate(ctx context.Context, userID int64, id string) error {
	id, err := normalizeTaskTemplateID(id)
	if err != nil {
		return err
	}
	doc, err := s.load(ctx, userID)
	if err != nil {
		return err
	}
	for index, tmpl := range doc.Templates {
		if tmpl != nil && tmpl.ID == id {
			doc.Templates = append(doc.Templates[:index], doc.Templates[index+1:]...)
			return s.save(ctx, userID, doc)
		}
	}
	return infraerrors.NotFound("TASK_TEMPLATE_NOT_FOUND", "task template not found")
}

func (s *TaskSettingsService) CopyTemplate(ctx context.Context, userID int64, id string) (*TaskTemplate, error) {
	source, err := s.GetTemplate(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	source.ID = ""
	source.Name = strings.TrimSpace(source.Name) + " Copy"
	source.IsDefault = false
	return s.SaveTemplate(ctx, userID, &TaskTemplateInput{
		Name:      source.Name,
		Type:      source.Type,
		Params:    source.Params,
		IsDefault: source.IsDefault,
	})
}

func (s *TaskSettingsService) SetDefaultTemplate(ctx context.Context, userID int64, id string) (*TaskTemplate, error) {
	id, err := normalizeTaskTemplateID(id)
	if err != nil {
		return nil, err
	}
	doc, err := s.load(ctx, userID)
	if err != nil {
		return nil, err
	}
	var selected *TaskTemplate
	for _, tmpl := range doc.Templates {
		if tmpl != nil && tmpl.ID == id {
			selected = tmpl
			break
		}
	}
	if selected == nil {
		return nil, infraerrors.NotFound("TASK_TEMPLATE_NOT_FOUND", "task template not found")
	}
	if result := ValidateTaskTemplate(selected); !result.Valid {
		return nil, infraerrors.BadRequest("TASK_TEMPLATE_INVALID", strings.Join(result.Errors, "; "))
	}
	clearTaskTemplateDefaults(doc, selected.Type, selected.ID)
	selected.IsDefault = true
	selected.UpdatedAt = time.Now().UTC()
	if err := s.save(ctx, userID, doc); err != nil {
		return nil, err
	}
	return cloneTaskTemplate(selected), nil
}

func ValidateTaskTemplate(tmpl *TaskTemplate) TaskTemplateValidationResult {
	result := TaskTemplateValidationResult{Valid: true}
	if tmpl == nil {
		return TaskTemplateValidationResult{Valid: false, Errors: []string{"template is required"}}
	}
	params := normalizeTaskTemplateParamsForType(tmpl.Type, tmpl.Params)
	result.Type = tmpl.Type
	result.Targets = len(params.Targets)
	result.Contents = len(params.Contents)
	if result.Targets > maxTaskTemplatePoolValues {
		result.Errors = append(result.Errors, fmt.Sprintf("target list cannot exceed %d items", maxTaskTemplatePoolValues))
	}
	if result.Contents > maxTaskTemplatePoolValues {
		result.Errors = append(result.Errors, fmt.Sprintf("content pool cannot exceed %d items", maxTaskTemplatePoolValues))
	}
	for _, target := range params.Targets {
		if utf8.RuneCountInString(target) > maxTaskTemplateValueLength {
			result.Errors = append(result.Errors, fmt.Sprintf("target item cannot exceed %d characters", maxTaskTemplateValueLength))
			break
		}
	}
	for _, content := range params.Contents {
		if utf8.RuneCountInString(content) > maxTaskTemplateValueLength {
			result.Errors = append(result.Errors, fmt.Sprintf("content item cannot exceed %d characters", maxTaskTemplateValueLength))
			break
		}
	}
	switch tmpl.Type {
	case SocialTaskActionFollow, SocialTaskActionLike, SocialTaskActionRetweet:
		if result.Targets == 0 {
			result.Errors = append(result.Errors, "target list is required")
		}
	case SocialTaskActionPost:
		if err := validateSocialTaskSupportedPostMedia(params.Media); err != nil {
			result.Errors = append(result.Errors, err.Error())
		}
		if result.Contents == 0 && len(params.Media) == 0 {
			result.Errors = append(result.Errors, "post template requires content pool or media")
		}
	case SocialTaskActionUpdateProfile:
		if params.Profile == nil || params.Profile.IsZero() {
			result.Errors = append(result.Errors, "profile settings are required")
		}
	case SocialTaskActionUpdateAvatar:
		if params.Avatar == nil || params.Avatar.IsZero() {
			result.Errors = append(result.Errors, "avatar media is required")
		} else if err := validateSocialTaskExecutableInlineMediaSource("avatar", params.Avatar); err != nil {
			result.Errors = append(result.Errors, err.Error())
		} else if err := validateSocialTaskImageMedia("avatar", params.Avatar); err != nil {
			result.Errors = append(result.Errors, err.Error())
		} else if err := validateSocialTaskExactImageDimensions("avatar", params.Avatar, socialTaskAvatarImageWidth, socialTaskAvatarImageHeight); err != nil {
			result.Errors = append(result.Errors, err.Error())
		}
	case SocialTaskActionUpdateBanner:
		if params.Banner == nil || params.Banner.IsZero() {
			result.Errors = append(result.Errors, "banner media is required")
		} else if err := validateSocialTaskExecutableInlineMediaSource("banner", params.Banner); err != nil {
			result.Errors = append(result.Errors, err.Error())
		} else if err := validateSocialTaskImageMedia("banner", params.Banner); err != nil {
			result.Errors = append(result.Errors, err.Error())
		} else if err := validateSocialTaskExactImageDimensions("banner", params.Banner, socialTaskBannerImageWidth, socialTaskBannerImageHeight); err != nil {
			result.Errors = append(result.Errors, err.Error())
		}
	default:
		result.Errors = append(result.Errors, "unsupported task template type")
	}
	result.Valid = len(result.Errors) == 0
	return result
}

func ValidateTaskTemplateInput(input *TaskTemplateInput) TaskTemplateValidationResult {
	tmpl, err := normalizeTaskTemplateInput(input)
	if err != nil {
		return TaskTemplateValidationResult{Valid: false, Errors: []string{err.Error()}}
	}
	return ValidateTaskTemplate(tmpl)
}

func (s *TaskSettingsService) ValidateTemplateInput(input *TaskTemplateInput) TaskTemplateValidationResult {
	return ValidateTaskTemplateInput(input)
}

func normalizeTaskTemplateInput(input *TaskTemplateInput) (*TaskTemplate, error) {
	templateType, err := normalizeTaskTemplateType(input.Type)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, infraerrors.BadRequest("TASK_TEMPLATE_NAME_REQUIRED", "task template name is required")
	}
	return &TaskTemplate{
		ID:        strings.TrimSpace(input.ID),
		Name:      name,
		Type:      templateType,
		Params:    normalizeTaskTemplateParamsForType(templateType, input.Params),
		IsDefault: input.IsDefault,
	}, nil
}

func normalizeTaskTemplateID(raw string) (string, error) {
	id := strings.TrimSpace(raw)
	if id == "" {
		return "", infraerrors.BadRequest("TASK_TEMPLATE_ID_REQUIRED", "task template id is required")
	}
	return id, nil
}

func normalizeTaskTemplateType(raw string) (string, error) {
	action := strings.TrimSpace(raw)
	switch action {
	case SocialTaskActionPost,
		SocialTaskActionLike,
		SocialTaskActionRetweet,
		SocialTaskActionFollow,
		SocialTaskActionUpdateProfile,
		SocialTaskActionUpdateAvatar,
		SocialTaskActionUpdateBanner:
		return action, nil
	default:
		return "", ErrSocialTaskUnsupportedAction
	}
}

func normalizeTaskTemplateParams(params TaskTemplateParams) TaskTemplateParams {
	normalized := TaskTemplateParams{
		Targets:  normalizeAccountWorkbenchTaskValues(params.Targets),
		Contents: normalizeAccountWorkbenchTaskValues(params.Contents),
		Media:    normalizeSocialTaskMediaRefs(params.Media),
	}
	normalized.QuotePostURL = strings.TrimSpace(params.QuotePostURL)
	if params.Profile != nil {
		profile := normalizeSocialProfileUpdateParams(*params.Profile)
		if !profile.IsZero() {
			normalized.Profile = &profile
		}
	}
	if params.Avatar != nil {
		avatar := normalizeSocialTaskMediaRef(*params.Avatar)
		if !avatar.IsZero() {
			normalized.Avatar = &avatar
		}
	}
	if params.Banner != nil {
		banner := normalizeSocialTaskMediaRef(*params.Banner)
		if !banner.IsZero() {
			normalized.Banner = &banner
		}
	}
	return normalized
}

func normalizeTaskTemplateParamsForType(templateType string, params TaskTemplateParams) TaskTemplateParams {
	normalized := normalizeTaskTemplateParams(params)
	switch templateType {
	case SocialTaskActionFollow, SocialTaskActionLike, SocialTaskActionRetweet:
		return TaskTemplateParams{Targets: normalized.Targets}
	case SocialTaskActionPost:
		return TaskTemplateParams{
			Contents:     normalized.Contents,
			QuotePostURL: normalized.QuotePostURL,
			Media:        normalized.Media,
		}
	case SocialTaskActionUpdateProfile:
		return TaskTemplateParams{Profile: normalized.Profile}
	case SocialTaskActionUpdateAvatar:
		return TaskTemplateParams{Avatar: normalized.Avatar}
	case SocialTaskActionUpdateBanner:
		return TaskTemplateParams{Banner: normalized.Banner}
	default:
		return normalized
	}
}

func clearTaskTemplateDefaults(doc *taskTemplateDocument, templateType, exceptID string) {
	if doc == nil {
		return
	}
	for _, tmpl := range doc.Templates {
		if tmpl != nil && tmpl.Type == templateType && tmpl.ID != exceptID {
			tmpl.IsDefault = false
		}
	}
}

func (s *TaskSettingsService) load(ctx context.Context, userID int64) (*taskTemplateDocument, error) {
	key := taskSettingsKey(userID)
	client := taskSettingsClientFromContext(ctx, s.entClient)
	item, err := client.Setting.Query().Where(setting.KeyEQ(key)).Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return &taskTemplateDocument{}, nil
		}
		return nil, err
	}
	doc := &taskTemplateDocument{}
	if strings.TrimSpace(item.Value) == "" {
		return doc, nil
	}
	if err := json.Unmarshal([]byte(item.Value), doc); err != nil {
		return nil, infraerrors.BadRequest("TASK_TEMPLATE_STORE_INVALID", "task template store is invalid")
	}
	clean := make([]*TaskTemplate, 0, len(doc.Templates))
	for _, tmpl := range doc.Templates {
		if tmpl == nil {
			continue
		}
		if templateType, err := normalizeTaskTemplateType(tmpl.Type); err == nil {
			tmpl.Type = templateType
			tmpl.Params = normalizeTaskTemplateParamsForType(templateType, tmpl.Params)
		} else {
			continue
		}
		clean = append(clean, tmpl)
	}
	doc.Templates = clean
	return doc, nil
}

func (s *TaskSettingsService) save(ctx context.Context, userID int64, doc *taskTemplateDocument) error {
	if doc == nil {
		doc = &taskTemplateDocument{}
	}
	client := taskSettingsClientFromContext(ctx, s.entClient)
	raw, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	return client.Setting.Create().
		SetKey(taskSettingsKey(userID)).
		SetValue(string(raw)).
		SetUpdatedAt(now).
		OnConflictColumns(setting.FieldKey).
		UpdateNewValues().
		Exec(ctx)
}

func taskSettingsClientFromContext(ctx context.Context, entClient *dbent.Client) *dbent.Client {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return tx.Client()
	}
	return entClient
}

func taskSettingsKey(userID int64) string {
	return fmt.Sprintf("%s%d", taskSettingsKeyPrefix, userID)
}

func cloneTaskTemplate(tmpl *TaskTemplate) *TaskTemplate {
	if tmpl == nil {
		return nil
	}
	cloned := *tmpl
	clonedParams := TaskTemplateParams{
		Targets:      append([]string(nil), tmpl.Params.Targets...),
		Contents:     append([]string(nil), tmpl.Params.Contents...),
		QuotePostURL: tmpl.Params.QuotePostURL,
	}
	if len(tmpl.Params.Media) > 0 {
		clonedParams.Media = append([]SocialTaskMediaRef(nil), tmpl.Params.Media...)
	}
	if tmpl.Params.Profile != nil {
		profile := *tmpl.Params.Profile
		clonedParams.Profile = &profile
	}
	if tmpl.Params.Avatar != nil {
		avatar := *tmpl.Params.Avatar
		clonedParams.Avatar = &avatar
	}
	if tmpl.Params.Banner != nil {
		banner := *tmpl.Params.Banner
		clonedParams.Banner = &banner
	}
	cloned.Params = clonedParams
	return &cloned
}

func normalizeSocialTaskMediaRefs(items []SocialTaskMediaRef) []SocialTaskMediaRef {
	if len(items) == 0 {
		return nil
	}
	normalized := make([]SocialTaskMediaRef, 0, len(items))
	for _, item := range items {
		clean := normalizeSocialTaskMediaRef(item)
		if clean.IsZero() {
			continue
		}
		normalized = append(normalized, clean)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizeSocialTaskMediaRef(item SocialTaskMediaRef) SocialTaskMediaRef {
	item.Source = strings.TrimSpace(item.Source)
	item.StorageKey = strings.TrimSpace(item.StorageKey)
	item.URL = strings.TrimSpace(item.URL)
	item.ContentType = strings.TrimSpace(item.ContentType)
	item.FileName = strings.TrimSpace(item.FileName)
	item.SHA256 = strings.TrimSpace(item.SHA256)
	if width, height, known, err := socialTaskMediaDimensions(&item); err == nil && known {
		item.Width = width
		item.Height = height
	}
	return item
}

func normalizeSocialProfileUpdateParams(profile SocialProfileUpdateParams) SocialProfileUpdateParams {
	profile.DisplayName = strings.TrimSpace(profile.DisplayName)
	profile.ScreenName = strings.TrimSpace(profile.ScreenName)
	profile.Description = strings.TrimSpace(profile.Description)
	profile.Location = strings.TrimSpace(profile.Location)
	profile.URL = strings.TrimSpace(profile.URL)
	return profile
}
