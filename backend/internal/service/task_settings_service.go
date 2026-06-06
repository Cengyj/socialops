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
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
)

const taskSettingsKeyPrefix = "socialops:task_settings:user:"

const (
	maxTaskTemplatePoolValues  = 500
	maxTaskTemplateValueLength = 2048
)

// TaskTemplateParams stores action-specific parameter pools.
type TaskTemplateParams struct {
	Targets  []string `json:"targets,omitempty"`
	Contents []string `json:"contents,omitempty"`
}

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
}

func NewTaskSettingsService(entClient *dbent.Client) *TaskSettingsService {
	return &TaskSettingsService{entClient: entClient}
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
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, infraerrors.BadRequest("TASK_TEMPLATE_ID_REQUIRED", "task template id is required")
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

func (s *TaskSettingsService) SaveTemplate(ctx context.Context, userID int64, input *TaskTemplateInput) (*TaskTemplate, error) {
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

func (s *TaskSettingsService) DeleteTemplate(ctx context.Context, userID int64, id string) error {
	doc, err := s.load(ctx, userID)
	if err != nil {
		return err
	}
	id = strings.TrimSpace(id)
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
	doc, err := s.load(ctx, userID)
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
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
	case SocialTaskActionLoginCheck:
	case SocialTaskActionFollow, SocialTaskActionLike, SocialTaskActionRetweet:
		if result.Targets == 0 {
			result.Errors = append(result.Errors, "target list is required")
		}
	case SocialTaskActionPost:
		if result.Contents == 0 {
			result.Errors = append(result.Errors, "content pool is required")
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

func normalizeTaskTemplateType(raw string) (string, error) {
	action := strings.TrimSpace(raw)
	switch action {
	case SocialTaskActionLoginCheck, SocialTaskActionPost, SocialTaskActionLike, SocialTaskActionRetweet, SocialTaskActionFollow:
		return action, nil
	default:
		return "", ErrSocialTaskUnsupportedAction
	}
}

func normalizeTaskTemplateParams(params TaskTemplateParams) TaskTemplateParams {
	return TaskTemplateParams{
		Targets:  normalizeAccountWorkbenchTaskValues(params.Targets),
		Contents: normalizeAccountWorkbenchTaskValues(params.Contents),
	}
}

func normalizeTaskTemplateParamsForType(templateType string, params TaskTemplateParams) TaskTemplateParams {
	normalized := normalizeTaskTemplateParams(params)
	switch templateType {
	case SocialTaskActionLoginCheck:
		return TaskTemplateParams{}
	case SocialTaskActionFollow, SocialTaskActionLike, SocialTaskActionRetweet:
		return TaskTemplateParams{Targets: normalized.Targets}
	case SocialTaskActionPost:
		return TaskTemplateParams{Contents: normalized.Contents}
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
	item, err := s.entClient.Setting.Query().Where(setting.KeyEQ(key)).Only(ctx)
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
			tmpl.Params = normalizeTaskTemplateParams(tmpl.Params)
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
	raw, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	return s.entClient.Setting.Create().
		SetKey(taskSettingsKey(userID)).
		SetValue(string(raw)).
		SetUpdatedAt(now).
		OnConflictColumns(setting.FieldKey).
		UpdateNewValues().
		Exec(ctx)
}

func taskSettingsKey(userID int64) string {
	return fmt.Sprintf("%s%d", taskSettingsKeyPrefix, userID)
}

func cloneTaskTemplate(tmpl *TaskTemplate) *TaskTemplate {
	if tmpl == nil {
		return nil
	}
	cloned := *tmpl
	cloned.Params = TaskTemplateParams{
		Targets:  append([]string(nil), tmpl.Params.Targets...),
		Contents: append([]string(nil), tmpl.Params.Contents...),
	}
	return &cloned
}
