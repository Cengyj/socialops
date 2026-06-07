package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/internal/domain"
)

var (
	errSocialTaskMediaAssetUnavailable = errors.New("social task media asset is unavailable")
	errSocialTaskMediaAssetInvalid     = errors.New("social task media asset is invalid")
)

type SocialTaskMediaAsset struct {
	ID              int64
	UserID          int64
	StorageProvider string
	StorageKey      string
	URL             string
	ContentType     string
	FileName        string
	SHA256          string
	ByteSize        int64
	Width           int
	Height          int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ResolvedSocialTaskMedia struct {
	ContentType string
	FileName    string
	Body        []byte
	SHA256      string
	ByteSize    int64
	Width       int
	Height      int
}

type SocialTaskMediaResolver interface {
	Resolve(ctx context.Context, userID int64, ref *domain.SocialTaskMediaRef) (*ResolvedSocialTaskMedia, error)
}

type SocialTaskMediaService struct {
	entClient *dbent.Client
}

func NewSocialTaskMediaService(entClient *dbent.Client) *SocialTaskMediaService {
	return &SocialTaskMediaService{entClient: entClient}
}

func (s *SocialTaskMediaService) MaterializeTaskLogMedia(
	ctx context.Context,
	userID int64,
	payload *domain.SocialTaskPayload,
	snapshot *domain.SocialTaskTemplateSnapshot,
) (*domain.SocialTaskPayload, *domain.SocialTaskTemplateSnapshot, error) {
	cache := make(map[string]domain.SocialTaskMediaRef)
	materializedPayload, err := s.materializeTaskPayload(ctx, userID, payload, cache)
	if err != nil {
		return nil, nil, err
	}
	materializedSnapshot, err := s.materializeTemplateSnapshot(ctx, userID, snapshot, cache)
	if err != nil {
		return nil, nil, err
	}
	return materializedPayload, materializedSnapshot, nil
}

func (s *SocialTaskMediaService) Resolve(
	ctx context.Context,
	userID int64,
	ref *domain.SocialTaskMediaRef,
) (*ResolvedSocialTaskMedia, error) {
	if ref == nil || ref.IsZero() {
		return nil, fmt.Errorf("%w: media ref is empty", errSocialTaskMediaAssetUnavailable)
	}
	source := strings.TrimSpace(ref.Source)
	if source != "library" {
		return nil, fmt.Errorf("%w: media source %q is not stored", errSocialTaskMediaAssetInvalid, source)
	}
	storageKey := strings.TrimSpace(ref.StorageKey)
	if storageKey == "" {
		return nil, fmt.Errorf("%w: storage key is required", errSocialTaskMediaAssetUnavailable)
	}
	asset, err := s.getAssetByStorageKey(ctx, userID, storageKey)
	if err != nil {
		return nil, err
	}
	contentType, body, err := parseTwitterDataURL(strings.TrimSpace(asset.URL))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errSocialTaskMediaAssetInvalid, err)
	}
	if storedType := strings.TrimSpace(asset.ContentType); storedType != "" {
		contentType = storedType
	}
	fileName := strings.TrimSpace(asset.FileName)
	if fileName == "" {
		fileName = "task-media" + socialTaskMediaFileExtensionFromContentType(contentType)
	}
	return &ResolvedSocialTaskMedia{
		ContentType: contentType,
		FileName:    fileName,
		Body:        body,
		SHA256:      strings.TrimSpace(asset.SHA256),
		ByteSize:    asset.ByteSize,
		Width:       asset.Width,
		Height:      asset.Height,
	}, nil
}

func (s *SocialTaskMediaService) materializeTaskPayload(
	ctx context.Context,
	userID int64,
	payload *domain.SocialTaskPayload,
	cache map[string]domain.SocialTaskMediaRef,
) (*domain.SocialTaskPayload, error) {
	if payload == nil || payload.IsZero() {
		return nil, nil
	}
	cloned := cloneSocialTaskPayload(payload)
	if cloned == nil || cloned.Post == nil || len(cloned.Post.Media) == 0 {
		return s.materializeProfileMediaPayload(ctx, userID, cloned, cache)
	}
	media, err := s.materializeMediaRefs(ctx, userID, cloned.Post.Media, cache)
	if err != nil {
		return nil, err
	}
	cloned.Post.Media = media
	return s.materializeProfileMediaPayload(ctx, userID, cloned, cache)
}

func (s *SocialTaskMediaService) materializeTemplateSnapshot(
	ctx context.Context,
	userID int64,
	snapshot *domain.SocialTaskTemplateSnapshot,
	cache map[string]domain.SocialTaskMediaRef,
) (*domain.SocialTaskTemplateSnapshot, error) {
	if snapshot == nil || snapshot.IsZero() {
		return nil, nil
	}
	cloned := cloneSocialTaskTemplateSnapshot(snapshot)
	if cloned == nil || len(cloned.Params.Media) == 0 {
		return s.materializeProfileMediaSnapshot(ctx, userID, cloned, cache)
	}
	media, err := s.materializeMediaRefs(ctx, userID, cloned.Params.Media, cache)
	if err != nil {
		return nil, err
	}
	cloned.Params.Media = media
	return s.materializeProfileMediaSnapshot(ctx, userID, cloned, cache)
}

func (s *SocialTaskMediaService) materializeProfileMediaPayload(
	ctx context.Context,
	userID int64,
	payload *domain.SocialTaskPayload,
	cache map[string]domain.SocialTaskMediaRef,
) (*domain.SocialTaskPayload, error) {
	if payload == nil {
		return nil, nil
	}
	if payload.Avatar != nil && !payload.Avatar.IsZero() {
		avatar, err := s.materializeMediaRef(ctx, userID, *payload.Avatar, cache)
		if err != nil {
			return nil, err
		}
		if avatar.IsZero() {
			payload.Avatar = nil
		} else {
			payload.Avatar = &avatar
		}
	}
	if payload.Banner != nil && !payload.Banner.IsZero() {
		banner, err := s.materializeMediaRef(ctx, userID, *payload.Banner, cache)
		if err != nil {
			return nil, err
		}
		if banner.IsZero() {
			payload.Banner = nil
		} else {
			payload.Banner = &banner
		}
	}
	return payload, nil
}

func (s *SocialTaskMediaService) materializeProfileMediaSnapshot(
	ctx context.Context,
	userID int64,
	snapshot *domain.SocialTaskTemplateSnapshot,
	cache map[string]domain.SocialTaskMediaRef,
) (*domain.SocialTaskTemplateSnapshot, error) {
	if snapshot == nil {
		return nil, nil
	}
	if snapshot.Params.Avatar != nil && !snapshot.Params.Avatar.IsZero() {
		avatar, err := s.materializeMediaRef(ctx, userID, *snapshot.Params.Avatar, cache)
		if err != nil {
			return nil, err
		}
		if avatar.IsZero() {
			snapshot.Params.Avatar = nil
		} else {
			snapshot.Params.Avatar = &avatar
		}
	}
	if snapshot.Params.Banner != nil && !snapshot.Params.Banner.IsZero() {
		banner, err := s.materializeMediaRef(ctx, userID, *snapshot.Params.Banner, cache)
		if err != nil {
			return nil, err
		}
		if banner.IsZero() {
			snapshot.Params.Banner = nil
		} else {
			snapshot.Params.Banner = &banner
		}
	}
	return snapshot, nil
}

func (s *SocialTaskMediaService) materializeMediaRefs(
	ctx context.Context,
	userID int64,
	refs []domain.SocialTaskMediaRef,
	cache map[string]domain.SocialTaskMediaRef,
) ([]domain.SocialTaskMediaRef, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	materialized := make([]domain.SocialTaskMediaRef, 0, len(refs))
	for _, ref := range refs {
		next, err := s.materializeMediaRef(ctx, userID, ref, cache)
		if err != nil {
			return nil, err
		}
		if next.IsZero() {
			continue
		}
		materialized = append(materialized, next)
	}
	if len(materialized) == 0 {
		return nil, nil
	}
	return materialized, nil
}

func (s *SocialTaskMediaService) materializeMediaRef(
	ctx context.Context,
	userID int64,
	ref domain.SocialTaskMediaRef,
	cache map[string]domain.SocialTaskMediaRef,
) (domain.SocialTaskMediaRef, error) {
	ref = normalizeSocialTaskMediaRef(ref)
	if ref.IsZero() {
		return domain.SocialTaskMediaRef{}, nil
	}
	source := strings.TrimSpace(ref.Source)
	rawURL := strings.TrimSpace(ref.URL)
	if source == "library" || (!strings.HasPrefix(strings.ToLower(rawURL), "data:") && rawURL == "") {
		return ref, nil
	}
	if source != "" && source != "inline" {
		return ref, nil
	}
	cacheKey := socialTaskInlineMediaCacheKey(ref)
	if cache != nil {
		if existing, ok := cache[cacheKey]; ok && !existing.IsZero() {
			return existing, nil
		}
	}

	asset, err := s.createInlineAsset(ctx, userID, ref)
	if err != nil {
		return domain.SocialTaskMediaRef{}, err
	}
	materialized := domain.SocialTaskMediaRef{
		Source:      "library",
		StorageKey:  asset.StorageKey,
		ContentType: asset.ContentType,
		FileName:    asset.FileName,
		SHA256:      asset.SHA256,
		ByteSize:    asset.ByteSize,
		Width:       asset.Width,
		Height:      asset.Height,
	}
	if cache != nil {
		cache[cacheKey] = materialized
	}
	return materialized, nil
}

func (s *SocialTaskMediaService) createInlineAsset(
	ctx context.Context,
	userID int64,
	ref domain.SocialTaskMediaRef,
) (*SocialTaskMediaAsset, error) {
	contentType, body, err := parseTwitterDataURL(strings.TrimSpace(ref.URL))
	if err != nil {
		return nil, fmt.Errorf("task media is invalid: %w", err)
	}
	if declaredType := strings.TrimSpace(ref.ContentType); declaredType != "" {
		contentType = declaredType
	}
	sum := sha256.Sum256(body)
	shaHex := hex.EncodeToString(sum[:])
	fileName := strings.TrimSpace(ref.FileName)
	if fileName == "" {
		fileName = "task-media" + socialTaskMediaFileExtensionFromContentType(contentType)
	}
	storageKey := buildSocialTaskMediaStorageKey(userID, shaHex, contentType)
	asset := &SocialTaskMediaAsset{
		UserID:          userID,
		StorageProvider: "inline",
		StorageKey:      storageKey,
		URL:             strings.TrimSpace(ref.URL),
		ContentType:     contentType,
		FileName:        fileName,
		SHA256:          shaHex,
		ByteSize:        int64(len(body)),
		Width:           ref.Width,
		Height:          ref.Height,
	}
	if err := s.insertAsset(ctx, asset); err != nil {
		return nil, err
	}
	return asset, nil
}

func (s *SocialTaskMediaService) insertAsset(ctx context.Context, asset *SocialTaskMediaAsset) error {
	if s == nil || s.entClient == nil || asset == nil {
		return fmt.Errorf("social task media service is not configured")
	}
	client := socialTaskMediaClientFromContext(ctx, s.entClient)
	_, err := client.ExecContext(ctx, `
INSERT INTO social_task_media_assets (
	user_id,
	storage_provider,
	storage_key,
	url,
	content_type,
	file_name,
	sha256,
	byte_size,
	width,
	height,
	created_at,
	updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
`,
		asset.UserID,
		strings.TrimSpace(asset.StorageProvider),
		strings.TrimSpace(asset.StorageKey),
		strings.TrimSpace(asset.URL),
		strings.TrimSpace(asset.ContentType),
		strings.TrimSpace(asset.FileName),
		strings.TrimSpace(asset.SHA256),
		asset.ByteSize,
		asset.Width,
		asset.Height,
	)
	return err
}

func (s *SocialTaskMediaService) getAssetByStorageKey(
	ctx context.Context,
	userID int64,
	storageKey string,
) (*SocialTaskMediaAsset, error) {
	if s == nil || s.entClient == nil {
		return nil, fmt.Errorf("%w: service is not configured", errSocialTaskMediaAssetUnavailable)
	}
	client := socialTaskMediaClientFromContext(ctx, s.entClient)
	rows, err := client.QueryContext(ctx, `
SELECT id, user_id, storage_provider, storage_key, url, content_type, file_name, sha256, byte_size, width, height, created_at, updated_at
FROM social_task_media_assets
WHERE user_id = $1 AND storage_key = $2
LIMIT 1
`, userID, strings.TrimSpace(storageKey))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%w: storage key not found", errSocialTaskMediaAssetUnavailable)
	}
	var asset SocialTaskMediaAsset
	if err := rows.Scan(
		&asset.ID,
		&asset.UserID,
		&asset.StorageProvider,
		&asset.StorageKey,
		&asset.URL,
		&asset.ContentType,
		&asset.FileName,
		&asset.SHA256,
		&asset.ByteSize,
		&asset.Width,
		&asset.Height,
		&asset.CreatedAt,
		&asset.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &asset, nil
}

func socialTaskMediaClientFromContext(ctx context.Context, entClient *dbent.Client) *dbent.Client {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return tx.Client()
	}
	return entClient
}

func buildSocialTaskMediaStorageKey(userID int64, shaHex, contentType string) string {
	fragment := strings.TrimSpace(shaHex)
	if len(fragment) > 16 {
		fragment = fragment[:16]
	}
	if fragment == "" {
		fragment = "media"
	}
	return fmt.Sprintf(
		"social-task/%d/%s-%d%s",
		userID,
		fragment,
		time.Now().UTC().UnixNano(),
		socialTaskMediaFileExtensionFromContentType(contentType),
	)
}

func socialTaskMediaFileExtensionFromContentType(contentType string) string {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "video/mp4":
		return ".mp4"
	default:
		return ""
	}
}

func socialTaskInlineMediaCacheKey(ref domain.SocialTaskMediaRef) string {
	return strings.Join([]string{
		strings.TrimSpace(ref.Source),
		strings.TrimSpace(ref.URL),
		strings.TrimSpace(ref.ContentType),
		strings.TrimSpace(ref.FileName),
		fmt.Sprintf("%d", ref.Width),
		fmt.Sprintf("%d", ref.Height),
	}, "|")
}
