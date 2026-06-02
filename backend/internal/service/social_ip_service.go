package service

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/socialaccount"
	"github.com/Wei-Shaw/socialops/ent/socialip"
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
)

// SocialIP represents a user-owned execution proxy.
type SocialIP struct {
	ID          int64      `json:"id"`
	UserID      int64      `json:"user_id"`
	Name        string     `json:"name"`
	IPType      string     `json:"ip_type"`
	Endpoint    *string    `json:"endpoint,omitempty"`
	Status      string     `json:"status"`
	LatencyMs   *int       `json:"latency_ms,omitempty"`
	LastCheckAt *time.Time `json:"last_check_at,omitempty"`
	Remark      *string    `json:"remark,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CreateSocialIPInput is the input for creating a social IP.
type CreateSocialIPInput struct {
	UserID   int64   `json:"-"`
	Name     string  `json:"name" binding:"required"`
	IPType   string  `json:"ip_type"`
	Endpoint *string `json:"endpoint"`
	Remark   *string `json:"remark"`
}

// UpdateSocialIPInput is the input for updating a social IP.
type UpdateSocialIPInput struct {
	Name     *string `json:"name"`
	IPType   *string `json:"ip_type"`
	Endpoint *string `json:"endpoint"`
	Remark   *string `json:"remark"`
}

// SocialIPListFilters contains admin filters for execution proxies.
type SocialIPListFilters struct {
	UserID *int64
	Status string
	IPType string
	Search string
}

// SocialIPService handles social IP/proxy management.
type SocialIPService struct {
	entClient *dbent.Client
}

// NewSocialIPService creates a new SocialIPService.
func NewSocialIPService(entClient *dbent.Client) *SocialIPService {
	return &SocialIPService{entClient: entClient}
}

// Create creates a new social IP entry.
func (s *SocialIPService) Create(ctx context.Context, input *CreateSocialIPInput) (*SocialIP, error) {
	q := s.entClient.SocialIP.Create().
		SetUserID(input.UserID).
		SetName(input.Name).
		SetIPType(input.IPType).
		SetStatus("unknown")

	if input.Endpoint != nil {
		endpoint, err := normalizeSocialIPEndpoint(input.Endpoint)
		if err != nil {
			return nil, err
		}
		q.SetEndpoint(endpoint)
	}
	if input.Remark != nil {
		q.SetRemark(*input.Remark)
	}

	ent, err := q.Save(ctx)
	if err != nil {
		return nil, err
	}
	return socialIPFromEnt(ent), nil
}

// GetByID returns a social IP by ID.
func (s *SocialIPService) GetByID(ctx context.Context, id int64) (*SocialIP, error) {
	ent, err := s.entClient.SocialIP.Get(ctx, id)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("SOCIAL_IP_NOT_FOUND", "social IP not found")
		}
		return nil, err
	}
	return socialIPFromEnt(ent), nil
}

// GetByIDForUser returns a social IP by ID, verifying it belongs to the given user.
func (s *SocialIPService) GetByIDForUser(ctx context.Context, id, userID int64) (*SocialIP, error) {
	ip, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if ip.UserID != userID {
		return nil, infraerrors.NotFound("SOCIAL_IP_NOT_FOUND", "social IP not found")
	}
	return ip, nil
}

// ListByUser returns social IPs for a specific user.
func (s *SocialIPService) ListByUser(ctx context.Context, userID int64, params pagination.PaginationParams) ([]*SocialIP, *pagination.PaginationResult, error) {
	q := s.entClient.SocialIP.Query().
		Where(socialip.UserIDEQ(userID))

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	ents, err := q.
		Offset(params.Offset()).
		Limit(params.Limit()).
		Order(dbent.Desc(socialip.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	ips := make([]*SocialIP, len(ents))
	for i, e := range ents {
		ips[i] = socialIPFromEnt(e)
	}

	result := &pagination.PaginationResult{Total: int64(total), Page: params.Page, PageSize: params.PageSize}
	return ips, result, nil
}

// ListForAdmin returns execution proxies across users for admin management.
func (s *SocialIPService) ListForAdmin(ctx context.Context, params pagination.PaginationParams, filters SocialIPListFilters) ([]*SocialIP, *pagination.PaginationResult, error) {
	q := s.entClient.SocialIP.Query()
	if filters.UserID != nil && *filters.UserID > 0 {
		q = q.Where(socialip.UserIDEQ(*filters.UserID))
	}
	if filters.Status != "" {
		q = q.Where(socialip.StatusEQ(filters.Status))
	}
	if filters.IPType != "" {
		q = q.Where(socialip.IPTypeEQ(filters.IPType))
	}
	if search := strings.TrimSpace(filters.Search); search != "" {
		q = q.Where(socialip.Or(
			socialip.NameContainsFold(search),
			socialip.EndpointContainsFold(search),
			socialip.RemarkContainsFold(search),
		))
	}

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	ents, err := q.
		Offset(params.Offset()).
		Limit(params.Limit()).
		Order(dbent.Desc(socialip.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	ips := make([]*SocialIP, len(ents))
	for i, e := range ents {
		ips[i] = socialIPFromEnt(e)
	}
	result := &pagination.PaginationResult{Total: int64(total), Page: params.Page, PageSize: params.PageSize}
	return ips, result, nil
}

// Update updates a social IP entry.
func (s *SocialIPService) Update(ctx context.Context, id int64, input *UpdateSocialIPInput) (*SocialIP, error) {
	q := s.entClient.SocialIP.UpdateOneID(id)

	if input.Name != nil {
		q.SetName(*input.Name)
	}
	if input.IPType != nil {
		q.SetIPType(*input.IPType)
	}
	if input.Endpoint != nil {
		endpoint, err := normalizeSocialIPEndpoint(input.Endpoint)
		if err != nil {
			return nil, err
		}
		q.SetEndpoint(endpoint).
			SetStatus(SocialIPStatusUnknown).
			ClearLatencyMs().
			ClearLastCheckAt()
	}
	if input.Remark != nil {
		q.SetRemark(*input.Remark)
	}

	ent, err := q.Save(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("SOCIAL_IP_NOT_FOUND", "social IP not found")
		}
		return nil, err
	}
	return socialIPFromEnt(ent), nil
}

// Delete soft-deletes a social IP entry.
func (s *SocialIPService) Delete(ctx context.Context, id int64) error {
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.SocialIP.Get(ctx, id); err != nil {
		if dbent.IsNotFound(err) {
			return infraerrors.NotFound("SOCIAL_IP_NOT_FOUND", "social IP not found")
		}
		return err
	}
	if err := clearDefaultProxySnapshotsForDeletedIP(ctx, tx.Client(), id); err != nil {
		return err
	}
	if err := tx.SocialIP.DeleteOneID(id).Exec(ctx); err != nil {
		if dbent.IsNotFound(err) {
			return infraerrors.NotFound("SOCIAL_IP_NOT_FOUND", "social IP not found")
		}
		return err
	}
	return tx.Commit()
}

func clearDefaultProxySnapshotsForDeletedIP(ctx context.Context, client *dbent.Client, id int64) error {
	accounts, err := client.SocialAccount.Query().
		Where(socialaccount.BoundIPNotNil()).
		All(ctx)
	if err != nil {
		return err
	}

	accountIDs := make([]int64, 0)
	for _, account := range accounts {
		if account.BoundIP == nil {
			continue
		}
		snapshotID, ok := SocialIPIDFromSnapshot(strings.TrimSpace(*account.BoundIP))
		if ok && snapshotID == id {
			accountIDs = append(accountIDs, account.ID)
		}
	}
	if len(accountIDs) == 0 {
		return nil
	}
	_, err = client.SocialAccount.Update().
		Where(socialaccount.IDIn(accountIDs...)).
		ClearBoundIP().
		Save(ctx)
	return err
}

func normalizeSocialIPEndpoint(endpoint *string) (string, error) {
	if endpoint == nil {
		return "", nil
	}
	normalized := strings.TrimSpace(*endpoint)
	if normalized == "" {
		return normalized, nil
	}
	parsed, err := url.Parse(normalized)
	if err != nil {
		return "", infraerrors.BadRequest("INVALID_PROXY_ENDPOINT", "invalid proxy endpoint URL")
	}
	if err := validateProxyEndpoint(parsed); err != nil {
		return "", infraerrors.BadRequest("INVALID_PROXY_ENDPOINT", err.Error())
	}
	return normalized, nil
}

func socialIPFromEnt(e *dbent.SocialIP) *SocialIP {
	ip := &SocialIP{
		ID:        int64(e.ID),
		UserID:    int64(e.UserID),
		Name:      e.Name,
		IPType:    e.IPType,
		Status:    e.Status,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
	if e.Endpoint != nil {
		ip.Endpoint = e.Endpoint
	}
	if e.LatencyMs != nil {
		ip.LatencyMs = e.LatencyMs
	}
	if e.LastCheckAt != nil {
		ip.LastCheckAt = e.LastCheckAt
	}
	if e.Remark != nil {
		ip.Remark = e.Remark
	}
	return ip
}

func SocialIPTaskSnapshot(ip *SocialIP) string {
	if ip == nil {
		return "{}"
	}
	payload := map[string]any{
		"id":       ip.ID,
		"name":     ip.Name,
		"ip_type":  ip.IPType,
		"endpoint": stringValue(ip.Endpoint),
		"status":   ip.Status,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func SocialIPIDFromSnapshot(snapshot string) (int64, bool) {
	var payload struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal([]byte(snapshot), &payload); err != nil || payload.ID <= 0 {
		return 0, false
	}
	return payload.ID, true
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
