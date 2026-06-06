package setup

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTestDatabaseTrimsConnectionInputBeforeValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/setup/test-db", bytes.NewReader([]byte(`{
		"host":" 127.0.0.1 ",
		"port":5432,
		"user":" postgres ",
		"password":"secret",
		"dbname":" socialops ",
		"sslmode":"bogus"
	}`)))
	c.Request.Header.Set("Content-Type", "application/json")

	testDatabase(c)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", recorder.Code)
	}
	message := setupTestResponseMessage(t, recorder)
	if strings.Contains(message, "hostname") || strings.Contains(message, "username") || strings.Contains(message, "database name") {
		t.Fatalf("expected trimmed input to pass string validation and fail on port, got message %q", message)
	}
	if !strings.Contains(message, "SSL mode") {
		t.Fatalf("expected SSL mode validation error after trimming string fields, got message %q", message)
	}
}

func TestTestRedisTrimsConnectionInputBeforeValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/setup/test-redis", bytes.NewReader([]byte(`{
		"host":" 127.0.0.1 ",
		"port":6379,
		"password":"secret",
		"db":16,
		"enable_tls":false
	}`)))
	c.Request.Header.Set("Content-Type", "application/json")

	testRedis(c)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", recorder.Code)
	}
	message := setupTestResponseMessage(t, recorder)
	if strings.Contains(message, "hostname") {
		t.Fatalf("expected trimmed host to pass hostname validation and fail on port, got message %q", message)
	}
	if !strings.Contains(message, "database number") {
		t.Fatalf("expected Redis database validation error after trimming host, got message %q", message)
	}
}

func TestInstallTrimsDatabaseSSLModeBeforeValidation(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/setup/install", bytes.NewReader([]byte(`{
		"database":{
			"host":"127.0.0.1",
			"port":5432,
			"user":"postgres",
			"password":"secret",
			"dbname":"socialops",
			"sslmode":" disable "
		},
		"redis":{
			"host":"127.0.0.1",
			"port":6379,
			"password":"",
			"db":0,
			"enable_tls":false
		},
		"admin":{
			"email":"admin@example.com",
			"password":"password123"
		},
		"server":{
			"host":"0.0.0.0",
			"port":8080,
			"mode":"bogus"
		}
	}`)))
	c.Request.Header.Set("Content-Type", "application/json")

	install(c)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", recorder.Code)
	}
	message := setupTestResponseMessage(t, recorder)
	if strings.Contains(message, "SSL mode") {
		t.Fatalf("expected sslmode to be trimmed before validation, got message %q", message)
	}
	if !strings.Contains(message, "server mode") {
		t.Fatalf("expected request to reach server mode validation after trimming sslmode, got message %q", message)
	}
}

func setupTestResponseMessage(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var payload struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, recorder.Body.String())
	}
	return payload.Message
}
