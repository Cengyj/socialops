package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/socialaccount"
	"github.com/Wei-Shaw/socialops/ent/socialip"
	"github.com/Wei-Shaw/socialops/ent/socialtasklog"
	"github.com/Wei-Shaw/socialops/ent/user"
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/pkg/proxyurl"
)

const (
	SocialIPTypeResidential = "residential"
	SocialIPTypeStatic      = "static"
	SocialIPTypeMobile      = "mobile"
	SocialIPTypeDatacenter  = "datacenter"
	SocialIPTypeDynamic     = "dynamic"
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

// SocialIPListFilters contains filters for execution proxies.
type SocialIPListFilters struct {
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
	if input == nil {
		return nil, infraerrors.BadRequest("SOCIAL_IP_INPUT_REQUIRED", "social IP input is required")
	}
	if input.UserID <= 0 {
		return nil, infraerrors.BadRequest("SOCIAL_IP_OWNER_REQUIRED", "social IP owner is required")
	}
	exists, err := s.entClient.User.Query().
		Where(user.IDEQ(input.UserID)).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, infraerrors.NotFound("SOCIAL_IP_OWNER_NOT_FOUND", "social IP owner not found")
	}

	ipType, err := normalizeSocialIPType(input.IPType, true)
	if err != nil {
		return nil, err
	}
	name, err := normalizeSocialIPName(input.Name)
	if err != nil {
		return nil, err
	}
	q := s.entClient.SocialIP.Create().
		SetUserID(input.UserID).
		SetName(name).
		SetIPType(ipType).
		SetStatus("unknown")

	if input.Endpoint != nil {
		endpoint, err := normalizeSocialIPEndpoint(input.Endpoint)
		if err != nil {
			return nil, err
		}
		if endpoint != "" {
			q.SetEndpoint(endpoint)
		}
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
func (s *SocialIPService) ListByUser(ctx context.Context, userID int64, params pagination.PaginationParams, filters SocialIPListFilters) ([]*SocialIP, *pagination.PaginationResult, error) {
	filters = normalizeSocialIPListFilters(filters)
	q := s.entClient.SocialIP.Query().
		Where(socialip.UserIDEQ(userID))
	q = applySocialIPListFilters(q, filters)

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

// ListUsableByUser returns online proxies with a configured endpoint for assignment and execution.
func (s *SocialIPService) ListUsableByUser(ctx context.Context, userID int64) ([]*SocialIP, error) {
	ents, err := s.entClient.SocialIP.Query().
		Where(
			socialip.UserIDEQ(userID),
			socialip.StatusEQ(SocialIPStatusOnline),
			socialip.EndpointNotNil(),
			socialip.EndpointNEQ(""),
		).
		Order(dbent.Asc(socialip.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	ips := make([]*SocialIP, 0, len(ents))
	for _, e := range ents {
		ip := socialIPFromEnt(e)
		if strings.TrimSpace(stringValue(ip.Endpoint)) == "" {
			continue
		}
		ips = append(ips, ip)
	}
	return ips, nil
}

// EnsureSocialIPUsableForExecution verifies that a proxy can be used to execute social tasks.
func EnsureSocialIPUsableForExecution(ip *SocialIP) error {
	if ip == nil {
		return infraerrors.BadRequest("SOCIAL_IP_NOT_AVAILABLE", "social IP is not available")
	}
	if ip.Status != SocialIPStatusOnline {
		return infraerrors.BadRequest("SOCIAL_IP_NOT_AVAILABLE", "social IP must pass a connectivity test before execution")
	}
	if strings.TrimSpace(stringValue(ip.Endpoint)) == "" {
		return infraerrors.BadRequest("SOCIAL_IP_NOT_AVAILABLE", "social IP endpoint is required for execution")
	}
	return nil
}

func applySocialIPListFilters(q *dbent.SocialIPQuery, filters SocialIPListFilters) *dbent.SocialIPQuery {
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
	return q
}

func normalizeSocialIPListFilters(filters SocialIPListFilters) SocialIPListFilters {
	filters.Status = strings.TrimSpace(filters.Status)
	filters.IPType = strings.TrimSpace(filters.IPType)
	filters.Search = strings.TrimSpace(filters.Search)
	return filters
}

// Update updates a social IP entry.
func (s *SocialIPService) Update(ctx context.Context, id int64, input *UpdateSocialIPInput) (*SocialIP, error) {
	return s.update(ctx, id, 0, input)
}

func (s *SocialIPService) update(ctx context.Context, id, userID int64, input *UpdateSocialIPInput) (*SocialIP, error) {
	if input == nil {
		return nil, infraerrors.BadRequest("SOCIAL_IP_INPUT_REQUIRED", "social IP input is required")
	}
	var normalizedName string
	if input.Name != nil {
		name, err := normalizeSocialIPName(*input.Name)
		if err != nil {
			return nil, err
		}
		normalizedName = name
	}
	var normalizedIPType string
	if input.IPType != nil {
		ipType, err := normalizeSocialIPType(*input.IPType, false)
		if err != nil {
			return nil, err
		}
		normalizedIPType = ipType
	}
	var normalizedEndpoint string
	if input.Endpoint != nil {
		endpoint, err := normalizeSocialIPEndpoint(input.Endpoint)
		if err != nil {
			return nil, err
		}
		normalizedEndpoint = endpoint
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	q := tx.SocialIP.UpdateOneID(id)
	if userID > 0 {
		q.Where(socialip.UserIDEQ(userID))
	}

	if input.Name != nil {
		q.SetName(normalizedName)
	}
	if input.IPType != nil {
		q.SetIPType(normalizedIPType)
	}
	if input.Endpoint != nil {
		currentQuery := tx.SocialIP.Query().Where(socialip.IDEQ(id))
		if userID > 0 {
			currentQuery = currentQuery.Where(socialip.UserIDEQ(userID))
		}
		current, err := currentQuery.Only(ctx)
		if err != nil {
			if dbent.IsNotFound(err) {
				return nil, infraerrors.NotFound("SOCIAL_IP_NOT_FOUND", "social IP not found")
			}
			return nil, err
		}
		if normalizedEndpoint == "" {
			q.ClearEndpoint()
		} else {
			q.SetEndpoint(normalizedEndpoint)
		}
		if normalizedEndpoint != stringValue(current.Endpoint) {
			q.SetStatus(SocialIPStatusUnknown).
				ClearLatencyMs().
				ClearLastCheckAt()
		}
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
	ip := socialIPFromEnt(ent)
	if err := syncDefaultProxySnapshotsForIP(ctx, tx.Client(), ip); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return ip, nil
}

// UpdateForUser updates a proxy only when it belongs to the current user.
func (s *SocialIPService) UpdateForUser(ctx context.Context, id, userID int64, input *UpdateSocialIPInput) (*SocialIP, error) {
	return s.update(ctx, id, userID, input)
}

// MarkExecutionReachable records that an execution request reached the platform
// through this proxy. It intentionally does not set latency because execution
// requests are not proxy benchmark probes.
func (s *SocialIPService) MarkExecutionReachable(ctx context.Context, id int64) error {
	if s == nil || s.entClient == nil || id <= 0 {
		return nil
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	ent, err := tx.SocialIP.UpdateOneID(id).
		SetStatus(SocialIPStatusOnline).
		SetLastCheckAt(time.Now()).
		Save(ctx)
	if err != nil {
		return err
	}
	if err := syncDefaultProxySnapshotsForIP(ctx, tx.Client(), socialIPFromEnt(ent)); err != nil {
		return err
	}
	return tx.Commit()
}

// Delete physically removes a social IP entry and clears references that point to it.
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
	if err := clearTaskLogProxyReferencesForDeletedIP(ctx, tx.Client(), id); err != nil {
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

// DeleteForUser deletes a proxy only when it belongs to the current user.
func (s *SocialIPService) DeleteForUser(ctx context.Context, id, userID int64) error {
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.SocialIP.Query().
		Where(socialip.IDEQ(id), socialip.UserIDEQ(userID)).
		Only(ctx); err != nil {
		if dbent.IsNotFound(err) {
			return infraerrors.NotFound("SOCIAL_IP_NOT_FOUND", "social IP not found")
		}
		return err
	}
	if err := clearDefaultProxySnapshotsForDeletedIPForUser(ctx, tx.Client(), id, userID); err != nil {
		return err
	}
	if err := clearTaskLogProxyReferencesForDeletedIP(ctx, tx.Client(), id); err != nil {
		return err
	}
	affected, err := tx.SocialIP.Delete().
		Where(socialip.IDEQ(id), socialip.UserIDEQ(userID)).
		Exec(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return infraerrors.NotFound("SOCIAL_IP_NOT_FOUND", "social IP not found")
	}
	return tx.Commit()
}

func clearDefaultProxySnapshotsForDeletedIP(ctx context.Context, client *dbent.Client, id int64) error {
	accounts, err := defaultProxySnapshotAccounts(ctx, client, id)
	if err != nil {
		return err
	}
	return clearDefaultProxySnapshotsPreservingAccountUpdatedAt(ctx, client, accounts)
}

func clearDefaultProxySnapshotsForDeletedIPForUser(ctx context.Context, client *dbent.Client, id, userID int64) error {
	accounts, err := defaultProxySnapshotAccountsForAssignedUser(ctx, client, id, userID)
	if err != nil {
		return err
	}
	return clearDefaultProxySnapshotsPreservingAccountUpdatedAt(ctx, client, accounts)
}

func clearTaskLogProxyReferencesForDeletedIP(ctx context.Context, client *dbent.Client, id int64) error {
	_, err := client.SocialTaskLog.Update().
		Where(socialtasklog.ProxyIDEQ(id)).
		ClearProxyID().
		Save(ctx)
	return err
}

func syncDefaultProxySnapshotsForIP(ctx context.Context, client *dbent.Client, ip *SocialIP) error {
	if ip == nil || ip.ID <= 0 {
		return nil
	}
	accounts, err := defaultProxySnapshotAccountsForAssignedUser(ctx, client, ip.ID, ip.UserID)
	if err != nil {
		return err
	}
	snapshot := SocialIPTaskSnapshot(ip)
	for _, account := range accounts {
		if account == nil || trimPtr(account.DefaultProxySnapshot) == snapshot {
			continue
		}
		if _, err := client.SocialAccount.UpdateOneID(account.ID).
			SetDefaultProxySnapshot(snapshot).
			SetUpdatedAt(account.UpdatedAt).
			Save(ctx); err != nil {
			return err
		}
	}
	return nil
}

func clearDefaultProxySnapshotsPreservingAccountUpdatedAt(ctx context.Context, client *dbent.Client, accounts []*dbent.SocialAccount) error {
	for _, account := range accounts {
		if account == nil || account.DefaultProxySnapshot == nil {
			continue
		}
		if _, err := client.SocialAccount.UpdateOneID(account.ID).
			ClearDefaultProxySnapshot().
			SetUpdatedAt(account.UpdatedAt).
			Save(ctx); err != nil {
			return err
		}
	}
	return nil
}

func defaultProxySnapshotAccountsForAssignedUser(ctx context.Context, client *dbent.Client, id, userID int64) ([]*dbent.SocialAccount, error) {
	accounts, err := client.SocialAccount.Query().
		Where(
			socialaccount.DefaultProxySnapshotNotNil(),
			socialaccount.AssignedUserIDEQ(userID),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return matchingDefaultProxySnapshotAccounts(accounts, id), nil
}

func defaultProxySnapshotAccounts(ctx context.Context, client *dbent.Client, id int64) ([]*dbent.SocialAccount, error) {
	accounts, err := client.SocialAccount.Query().
		Where(socialaccount.DefaultProxySnapshotNotNil()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return matchingDefaultProxySnapshotAccounts(accounts, id), nil
}

func matchingDefaultProxySnapshotAccounts(accounts []*dbent.SocialAccount, id int64) []*dbent.SocialAccount {
	matched := make([]*dbent.SocialAccount, 0, len(accounts))
	for _, account := range accounts {
		snapshot := trimPtr(account.DefaultProxySnapshot)
		if snapshot == "" {
			continue
		}
		snapshotID, ok := SocialIPIDFromSnapshot(snapshot)
		if ok && snapshotID == id {
			matched = append(matched, account)
		}
	}
	return matched
}

func normalizeSocialIPType(raw string, defaultIfEmpty bool) (string, error) {
	ipType := strings.TrimSpace(raw)
	if ipType == "" && defaultIfEmpty {
		return SocialIPTypeResidential, nil
	}
	switch ipType {
	case SocialIPTypeResidential, SocialIPTypeStatic, SocialIPTypeMobile, SocialIPTypeDatacenter, SocialIPTypeDynamic:
		return ipType, nil
	default:
		return "", infraerrors.BadRequest("SOCIAL_IP_TYPE_INVALID", "social IP type is invalid")
	}
}

func normalizeSocialIPName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", infraerrors.BadRequest("SOCIAL_IP_NAME_REQUIRED", "social IP name is required")
	}
	return name, nil
}

func normalizeSocialIPEndpoint(endpoint *string) (string, error) {
	if endpoint == nil {
		return "", nil
	}
	normalized := strings.TrimSpace(*endpoint)
	if normalized == "" {
		return normalized, nil
	}
	normalized, parsed, err := proxyurl.Parse(normalized)
	if err != nil {
		return "", infraerrors.BadRequest("INVALID_PROXY_ENDPOINT", "invalid proxy endpoint URL")
	}
	if isDynamicProxySourceURL(parsed) {
		if err := validateDynamicProxySourceURL(parsed); err != nil {
			return "", infraerrors.BadRequest("INVALID_PROXY_ENDPOINT", err.Error())
		}
		return normalized, nil
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

func SocialIPSnapshotUsable(snapshot string) bool {
	var payload struct {
		ID       int64  `json:"id"`
		Endpoint string `json:"endpoint"`
		Status   string `json:"status"`
	}
	if err := json.Unmarshal([]byte(snapshot), &payload); err != nil {
		return false
	}
	return payload.ID > 0 && strings.TrimSpace(payload.Endpoint) != "" && payload.Status == SocialIPStatusOnline
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
