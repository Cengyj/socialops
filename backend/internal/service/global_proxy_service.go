package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/globalproxy"
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/pkg/proxyurl"
)

// GlobalProxy is an administrator-managed fallback proxy for platform execution.
type GlobalProxy struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	IPType      string     `json:"ip_type"`
	Endpoint    *string    `json:"endpoint,omitempty"`
	Status      string     `json:"status"`
	LatencyMs   *int       `json:"latency_ms,omitempty"`
	LastCheckAt *time.Time `json:"last_check_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	Remark      *string    `json:"remark,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type CreateGlobalProxyInput struct {
	Name     string  `json:"name" binding:"required"`
	IPType   string  `json:"ip_type"`
	Endpoint *string `json:"endpoint"`
	Remark   *string `json:"remark"`
}

type UpdateGlobalProxyInput struct {
	Name     *string `json:"name"`
	IPType   *string `json:"ip_type"`
	Endpoint *string `json:"endpoint"`
	Remark   *string `json:"remark"`
}

type GlobalProxyListFilters struct {
	Status string
	IPType string
	Search string
}

type GlobalProxyCheckResult struct {
	ID        int64  `json:"id"`
	Status    string `json:"status"`
	LatencyMs int    `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

type GlobalProxyService struct {
	entClient *dbent.Client
}

func NewGlobalProxyService(entClient *dbent.Client) *GlobalProxyService {
	return &GlobalProxyService{entClient: entClient}
}

func (s *GlobalProxyService) Create(ctx context.Context, input *CreateGlobalProxyInput) (*GlobalProxy, error) {
	if input == nil {
		return nil, infraerrors.BadRequest("GLOBAL_PROXY_INPUT_REQUIRED", "global proxy input is required")
	}
	ipType, err := normalizeSocialIPType(input.IPType, true)
	if err != nil {
		return nil, err
	}
	name, err := normalizeSocialIPName(input.Name)
	if err != nil {
		return nil, err
	}
	q := s.entClient.GlobalProxy.Create().
		SetName(name).
		SetIPType(ipType).
		SetStatus(SocialIPStatusUnknown)
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
	return globalProxyFromEnt(ent), nil
}

func (s *GlobalProxyService) List(ctx context.Context, params pagination.PaginationParams, filters GlobalProxyListFilters) ([]*GlobalProxy, *pagination.PaginationResult, error) {
	filters = normalizeGlobalProxyListFilters(filters)
	q := s.entClient.GlobalProxy.Query()
	q = applyGlobalProxyListFilters(q, filters)
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}
	ents, err := q.
		Offset(params.Offset()).
		Limit(params.Limit()).
		Order(dbent.Asc(globalproxy.FieldID)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}
	proxies := make([]*GlobalProxy, len(ents))
	for i, e := range ents {
		proxies[i] = globalProxyFromEnt(e)
	}
	result := &pagination.PaginationResult{Total: int64(total), Page: params.Page, PageSize: params.PageSize}
	return proxies, result, nil
}

func (s *GlobalProxyService) GetByID(ctx context.Context, id int64) (*GlobalProxy, error) {
	ent, err := s.entClient.GlobalProxy.Get(ctx, id)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("GLOBAL_PROXY_NOT_FOUND", "global proxy not found")
		}
		return nil, err
	}
	return globalProxyFromEnt(ent), nil
}

func (s *GlobalProxyService) Update(ctx context.Context, id int64, input *UpdateGlobalProxyInput) (*GlobalProxy, error) {
	if input == nil {
		return nil, infraerrors.BadRequest("GLOBAL_PROXY_INPUT_REQUIRED", "global proxy input is required")
	}
	q := s.entClient.GlobalProxy.UpdateOneID(id)
	if input.Name != nil {
		name, err := normalizeSocialIPName(*input.Name)
		if err != nil {
			return nil, err
		}
		q.SetName(name)
	}
	if input.IPType != nil {
		ipType, err := normalizeSocialIPType(*input.IPType, false)
		if err != nil {
			return nil, err
		}
		q.SetIPType(ipType)
	}
	if input.Endpoint != nil {
		current, err := s.entClient.GlobalProxy.Get(ctx, id)
		if err != nil {
			if dbent.IsNotFound(err) {
				return nil, infraerrors.NotFound("GLOBAL_PROXY_NOT_FOUND", "global proxy not found")
			}
			return nil, err
		}
		endpoint, err := normalizeSocialIPEndpoint(input.Endpoint)
		if err != nil {
			return nil, err
		}
		if endpoint == "" {
			q.ClearEndpoint()
		} else {
			q.SetEndpoint(endpoint)
		}
		if endpoint != stringValue(current.Endpoint) {
			q.SetStatus(SocialIPStatusUnknown).ClearLatencyMs().ClearLastCheckAt().ClearLastUsedAt()
		}
	}
	if input.Remark != nil {
		q.SetRemark(*input.Remark)
	}
	ent, err := q.Save(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("GLOBAL_PROXY_NOT_FOUND", "global proxy not found")
		}
		return nil, err
	}
	return globalProxyFromEnt(ent), nil
}

func (s *GlobalProxyService) Delete(ctx context.Context, id int64) error {
	if err := s.entClient.GlobalProxy.DeleteOneID(id).Exec(ctx); err != nil {
		if dbent.IsNotFound(err) {
			return infraerrors.NotFound("GLOBAL_PROXY_NOT_FOUND", "global proxy not found")
		}
		return err
	}
	return nil
}

func (s *GlobalProxyService) Test(ctx context.Context, id int64) (*GlobalProxyCheckResult, error) {
	ip, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	result := checkGlobalProxyConnectivity(ctx, ip)
	if err := s.updateCheckResult(ctx, id, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *GlobalProxyService) TestAll(ctx context.Context) ([]*GlobalProxyCheckResult, error) {
	ents, err := s.entClient.GlobalProxy.Query().
		Order(dbent.Asc(globalproxy.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]*GlobalProxyCheckResult, 0, len(ents))
	for _, ent := range ents {
		ip := globalProxyFromEnt(ent)
		result := checkGlobalProxyConnectivity(ctx, ip)
		if err := s.updateCheckResult(ctx, ip.ID, result); err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func (s *GlobalProxyService) NextAvailable(ctx context.Context) (*GlobalProxy, error) {
	ent, err := s.entClient.GlobalProxy.Query().
		Where(
			globalproxy.StatusEQ(SocialIPStatusOnline),
			globalproxy.EndpointNotNil(),
			globalproxy.EndpointNEQ(""),
		).
		Order(
			globalproxy.ByLastUsedAt(),
			dbent.Asc(globalproxy.FieldID),
		).
		First(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.BadRequest("GLOBAL_PROXY_NOT_AVAILABLE", "global proxy is not available")
		}
		return nil, err
	}
	updated, err := s.entClient.GlobalProxy.UpdateOneID(ent.ID).
		SetLastUsedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return globalProxyFromEnt(updated), nil
}

func checkGlobalProxyConnectivity(ctx context.Context, ip *GlobalProxy) *GlobalProxyCheckResult {
	result := &GlobalProxyCheckResult{ID: ip.ID}
	if ip.Endpoint == nil || strings.TrimSpace(*ip.Endpoint) == "" {
		result.Status = SocialIPStatusUnknown
		result.Error = safeSocialIPCheckErrorMessage(result.Status)
		return result
	}
	endpoint, err := ResolveSocialIPExecutionEndpoint(ctx, *ip.Endpoint)
	if err != nil {
		result.Status = SocialIPStatusOffline
		result.Error = safeSocialIPCheckErrorMessage(result.Status)
		return result
	}
	_, parsed, _ := proxyurl.Parse(endpoint)
	checker := NewSocialIPChecker(nil)
	start := time.Now()
	var connErr error
	switch parsed.Scheme {
	case "socks5", "socks5h":
		connErr = checker.testSOCKS5(parsed)
	case "http", "https":
		connErr = checker.testHTTPProxy(parsed)
	default:
		result.Status = SocialIPStatusUnknown
		result.Error = safeSocialIPCheckErrorMessage(result.Status)
		return result
	}
	result.LatencyMs = int(time.Since(start).Milliseconds())
	if connErr != nil {
		result.Status = SocialIPStatusOffline
		result.Error = safeSocialIPCheckErrorMessage(result.Status)
	} else {
		result.Status = SocialIPStatusOnline
	}
	return result
}

func (s *GlobalProxyService) updateCheckResult(ctx context.Context, id int64, result *GlobalProxyCheckResult) error {
	q := s.entClient.GlobalProxy.UpdateOneID(id).
		SetStatus(result.Status).
		SetLastCheckAt(time.Now())
	if result.LatencyMs > 0 {
		q.SetLatencyMs(result.LatencyMs)
	} else {
		q.ClearLatencyMs()
	}
	if _, err := q.Save(ctx); err != nil {
		if dbent.IsNotFound(err) {
			return infraerrors.NotFound("GLOBAL_PROXY_NOT_FOUND", "global proxy not found")
		}
		return err
	}
	return nil
}

func GlobalProxyTaskSnapshot(ip *GlobalProxy) string {
	if ip == nil {
		return "{}"
	}
	payload := map[string]any{
		"scope":    "global",
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

func globalProxyFromEnt(e *dbent.GlobalProxy) *GlobalProxy {
	ip := &GlobalProxy{
		ID:        int64(e.ID),
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
	if e.LastUsedAt != nil {
		ip.LastUsedAt = e.LastUsedAt
	}
	if e.Remark != nil {
		ip.Remark = e.Remark
	}
	return ip
}

func applyGlobalProxyListFilters(q *dbent.GlobalProxyQuery, filters GlobalProxyListFilters) *dbent.GlobalProxyQuery {
	if filters.Status != "" {
		q = q.Where(globalproxy.StatusEQ(filters.Status))
	}
	if filters.IPType != "" {
		q = q.Where(globalproxy.IPTypeEQ(filters.IPType))
	}
	if search := strings.TrimSpace(filters.Search); search != "" {
		q = q.Where(globalproxy.Or(
			globalproxy.NameContainsFold(search),
			globalproxy.EndpointContainsFold(search),
			globalproxy.RemarkContainsFold(search),
		))
	}
	return q
}

func normalizeGlobalProxyListFilters(filters GlobalProxyListFilters) GlobalProxyListFilters {
	filters.Status = strings.TrimSpace(filters.Status)
	filters.IPType = strings.TrimSpace(filters.IPType)
	filters.Search = strings.TrimSpace(filters.Search)
	return filters
}
