package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/socialops/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type userAttributeDefinitionRepoStub struct {
	defs map[int64]service.UserAttributeDefinition
}

func (r *userAttributeDefinitionRepoStub) Create(_ context.Context, def *service.UserAttributeDefinition) error {
	r.defs[def.ID] = *def
	return nil
}

func (r *userAttributeDefinitionRepoStub) GetByID(_ context.Context, id int64) (*service.UserAttributeDefinition, error) {
	def, ok := r.defs[id]
	if !ok {
		return nil, service.ErrAttributeDefinitionNotFound
	}
	return &def, nil
}

func (r *userAttributeDefinitionRepoStub) GetByKey(_ context.Context, key string) (*service.UserAttributeDefinition, error) {
	for _, def := range r.defs {
		if def.Key == key {
			copied := def
			return &copied, nil
		}
	}
	return nil, service.ErrAttributeDefinitionNotFound
}

func (r *userAttributeDefinitionRepoStub) Update(_ context.Context, def *service.UserAttributeDefinition) error {
	r.defs[def.ID] = *def
	return nil
}

func (r *userAttributeDefinitionRepoStub) Delete(_ context.Context, id int64) error {
	delete(r.defs, id)
	return nil
}

func (r *userAttributeDefinitionRepoStub) List(_ context.Context, enabledOnly bool) ([]service.UserAttributeDefinition, error) {
	out := make([]service.UserAttributeDefinition, 0, len(r.defs))
	for _, def := range r.defs {
		if enabledOnly && !def.Enabled {
			continue
		}
		out = append(out, def)
	}
	return out, nil
}

func (r *userAttributeDefinitionRepoStub) UpdateDisplayOrders(_ context.Context, orders map[int64]int) error {
	for id, order := range orders {
		def := r.defs[id]
		def.DisplayOrder = order
		r.defs[id] = def
	}
	return nil
}

func (r *userAttributeDefinitionRepoStub) ExistsByKey(_ context.Context, key string) (bool, error) {
	for _, def := range r.defs {
		if def.Key == key {
			return true, nil
		}
	}
	return false, nil
}

type userAttributeValueRepoStub struct {
	values map[int64]map[int64]service.UserAttributeValue
}

func (r *userAttributeValueRepoStub) GetByUserID(_ context.Context, userID int64) ([]service.UserAttributeValue, error) {
	userValues := r.values[userID]
	out := make([]service.UserAttributeValue, 0, len(userValues))
	for _, value := range userValues {
		out = append(out, value)
	}
	return out, nil
}

func (r *userAttributeValueRepoStub) GetByUserIDs(_ context.Context, userIDs []int64) ([]service.UserAttributeValue, error) {
	out := make([]service.UserAttributeValue, 0)
	for _, userID := range userIDs {
		for _, value := range r.values[userID] {
			out = append(out, value)
		}
	}
	return out, nil
}

func (r *userAttributeValueRepoStub) UpsertBatch(_ context.Context, userID int64, values []service.UpdateUserAttributeInput) error {
	if r.values[userID] == nil {
		r.values[userID] = make(map[int64]service.UserAttributeValue)
	}
	now := time.Now().UTC()
	for _, input := range values {
		r.values[userID][input.AttributeID] = service.UserAttributeValue{
			ID:          input.AttributeID,
			UserID:      userID,
			AttributeID: input.AttributeID,
			Value:       input.Value,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
	}
	return nil
}

func (r *userAttributeValueRepoStub) DeleteByAttributeID(_ context.Context, attributeID int64) error {
	for userID, userValues := range r.values {
		delete(userValues, attributeID)
		r.values[userID] = userValues
	}
	return nil
}

func (r *userAttributeValueRepoStub) DeleteByUserID(_ context.Context, userID int64) error {
	delete(r.values, userID)
	return nil
}

func TestUserAttributesBatchCacheInvalidatedAfterUserAttributeUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalCache := userAttributesBatchCache
	t.Cleanup(func() { userAttributesBatchCache = originalCache })
	userAttributesBatchCache = newSnapshotCache(time.Hour)

	router := newUserAttributeHandlerTestRouter()

	first := performUserAttributeJSONRequest(t, router, http.MethodPost, "/admin/user-attributes/batch", gin.H{"user_ids": []int64{7}})
	require.Equal(t, http.StatusOK, first.Code)
	require.Equal(t, "miss", first.Header().Get("X-Snapshot-Cache"))
	require.Equal(t, "old", decodeUserAttributesBatchResponse(t, first)["7"]["9"])

	update := performUserAttributeJSONRequest(t, router, http.MethodPut, "/admin/users/7/attributes", gin.H{"values": map[int64]string{9: "new"}})
	require.Equal(t, http.StatusOK, update.Code)

	second := performUserAttributeJSONRequest(t, router, http.MethodPost, "/admin/user-attributes/batch", gin.H{"user_ids": []int64{7}})
	require.Equal(t, http.StatusOK, second.Code)
	require.Equal(t, "new", decodeUserAttributesBatchResponse(t, second)["7"]["9"])
}

func TestUserAttributesBatchCacheInvalidatedAfterDefinitionDelete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalCache := userAttributesBatchCache
	t.Cleanup(func() { userAttributesBatchCache = originalCache })
	userAttributesBatchCache = newSnapshotCache(time.Hour)

	router := newUserAttributeHandlerTestRouter()

	first := performUserAttributeJSONRequest(t, router, http.MethodPost, "/admin/user-attributes/batch", gin.H{"user_ids": []int64{7}})
	require.Equal(t, http.StatusOK, first.Code)
	require.Equal(t, "old", decodeUserAttributesBatchResponse(t, first)["7"]["9"])

	deleteReq := httptest.NewRequest(http.MethodDelete, "/admin/user-attributes/9", nil)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)
	require.Equal(t, http.StatusOK, deleteRec.Code)

	second := performUserAttributeJSONRequest(t, router, http.MethodPost, "/admin/user-attributes/batch", gin.H{"user_ids": []int64{7}})
	require.Equal(t, http.StatusOK, second.Code)
	require.NotContains(t, decodeUserAttributesBatchResponse(t, second)["7"], "9")
}

func newUserAttributeHandlerTestRouter() *gin.Engine {
	defRepo := &userAttributeDefinitionRepoStub{
		defs: map[int64]service.UserAttributeDefinition{
			9: {
				ID:      9,
				Key:     "tier",
				Name:    "Tier",
				Type:    service.AttributeTypeText,
				Enabled: true,
			},
		},
	}
	valueRepo := &userAttributeValueRepoStub{
		values: map[int64]map[int64]service.UserAttributeValue{
			7: {
				9: {
					ID:          99,
					UserID:      7,
					AttributeID: 9,
					Value:       "old",
					CreatedAt:   time.Now().UTC(),
					UpdatedAt:   time.Now().UTC(),
				},
			},
		},
	}
	handler := NewUserAttributeHandler(service.NewUserAttributeService(defRepo, valueRepo))

	router := gin.New()
	router.POST("/admin/user-attributes/batch", handler.GetBatchUserAttributes)
	router.PUT("/admin/users/:id/attributes", handler.UpdateUserAttributes)
	router.DELETE("/admin/user-attributes/:id", handler.DeleteDefinition)
	return router
}

func performUserAttributeJSONRequest(t *testing.T, router *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeUserAttributesBatchResponse(t *testing.T, rec *httptest.ResponseRecorder) map[string]map[string]string {
	t.Helper()
	var envelope struct {
		Data struct {
			Attributes map[string]map[string]string `json:"attributes"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	return envelope.Data.Attributes
}
