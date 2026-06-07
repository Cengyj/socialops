package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Wei-Shaw/socialops/internal/config"
	middleware2 "github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
)

func TestCleanPageImageRelativePath(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{name: "single filename", in: "logo.png", want: "logo.png", ok: true},
		{name: "nested path", in: "images/logo.png", want: filepath.Join("images", "logo.png"), ok: true},
		{name: "dot prefix", in: "./logo.png", want: "logo.png", ok: true},
		{name: "url escaped slash", in: "images%2Flogo.png", want: filepath.Join("images", "logo.png"), ok: true},
		{name: "parent traversal", in: "../secret.png", ok: false},
		{name: "encoded parent traversal", in: "%2e%2e/secret.png", ok: false},
		{name: "backslash traversal", in: `images\secret.png`, ok: false},
		{name: "absolute path", in: "/etc/passwd", ok: false},
		{name: "encoded absolute path", in: "%2fetc/passwd", ok: false},
		{name: "encoded nul byte", in: "logo.png%00", ok: false},
		{name: "invalid escape", in: "logo.png%zz", ok: false},
		{name: "empty path", in: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := cleanPageImageRelativePath(tt.in)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("path = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCleanPageSlugPath(t *testing.T) {
	tests := []struct {
		name           string
		in             string
		wantRelPath    string
		wantNormalized string
		ok             bool
	}{
		{name: "single slug", in: "guide", wantRelPath: "guide", wantNormalized: "guide", ok: true},
		{name: "nested slug", in: "help/intro", wantRelPath: filepath.Join("help", "intro"), wantNormalized: "help/intro", ok: true},
		{name: "escaped nested slug", in: "help%2Fintro", wantRelPath: filepath.Join("help", "intro"), wantNormalized: "help/intro", ok: true},
		{name: "dot in filename", in: "legal/privacy.v1", wantRelPath: filepath.Join("legal", "privacy.v1"), wantNormalized: "legal/privacy.v1", ok: true},
		{name: "parent traversal", in: "../admin", ok: false},
		{name: "encoded parent traversal", in: "%2e%2e/admin", ok: false},
		{name: "backslash traversal", in: `help\intro`, ok: false},
		{name: "empty segment", in: "help//intro", ok: false},
		{name: "dot segment", in: "help/./intro", ok: false},
		{name: "encoded nul byte", in: "help%00intro", ok: false},
		{name: "invalid escape", in: "help%zz", ok: false},
		{name: "empty slug", in: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRelPath, gotNormalized, ok := cleanPageSlugPath(tt.in)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if gotRelPath != tt.wantRelPath {
				t.Fatalf("rel path = %q, want %q", gotRelPath, tt.wantRelPath)
			}
			if gotNormalized != tt.wantNormalized {
				t.Fatalf("normalized slug = %q, want %q", gotNormalized, tt.wantNormalized)
			}
		})
	}
}

func TestPageMarkdownPathWithinBaseRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	pagesDir := filepath.Join(root, "pages")
	outside := filepath.Join(root, "outside")

	if err := os.MkdirAll(pagesDir, 0755); err != nil {
		t.Fatalf("create pages dir: %v", err)
	}
	if err := os.MkdirAll(outside, 0755); err != nil {
		t.Fatalf("create outside dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.md"), []byte("secret"), 0644); err != nil {
		t.Fatalf("create outside markdown: %v", err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret.md"), filepath.Join(pagesDir, "guide.md")); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	if pageMarkdownPathWithinBase(pagesDir, filepath.Join(pagesDir, "guide.md")) {
		t.Fatal("expected markdown symlink escape to be rejected")
	}
}

func TestPageRouteReceivesDoubleEscapedNestedSlug(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/pages/:slug", func(c *gin.Context) {
		_, normalizedSlug, ok := cleanPageSlugPath(c.Param("slug"))
		if !ok {
			c.Status(http.StatusBadRequest)
			return
		}
		c.String(http.StatusOK, normalizedSlug)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pages/help%252Fintro", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "help/intro" {
		t.Fatalf("slug = %q, want %q", rec.Body.String(), "help/intro")
	}
}

func TestResolvePageImagePath(t *testing.T) {
	root := t.TempDir()
	pagesDir := filepath.Join(root, "pages")
	base := filepath.Join(pagesDir, "guide")
	if err := os.MkdirAll(filepath.Join(base, "images"), 0755); err != nil {
		t.Fatalf("create images dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "logo.png"), []byte("fake"), 0644); err != nil {
		t.Fatalf("create direct image: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "images", "logo.png"), []byte("fake"), 0644); err != nil {
		t.Fatalf("create image: %v", err)
	}

	got, ok := resolvePageImagePath(pagesDir, base, "logo.png")
	if !ok {
		t.Fatal("expected direct image path to be accepted")
	}
	want := mustEvalSymlinks(t, filepath.Join(base, "logo.png"))
	if got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}

	got, ok = resolvePageImagePath(pagesDir, base, "images/logo.png")
	if !ok {
		t.Fatal("expected nested image path to be accepted")
	}
	want = mustEvalSymlinks(t, filepath.Join(base, "images", "logo.png"))
	if got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}

	if got, ok := resolvePageImagePath(pagesDir, base, "../guide.md"); ok {
		t.Fatalf("expected traversal to be rejected, got %q", got)
	}
}

func TestResolvePageImagePathRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	pagesDir := filepath.Join(root, "pages")
	base := filepath.Join(pagesDir, "guide")
	outside := filepath.Join(root, "outside")

	if err := os.MkdirAll(base, 0755); err != nil {
		t.Fatalf("create page dir: %v", err)
	}
	if err := os.MkdirAll(outside, 0755); err != nil {
		t.Fatalf("create outside dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.png"), []byte("secret"), 0644); err != nil {
		t.Fatalf("create outside file: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(base, "images")); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	if got, ok := resolvePageImagePath(pagesDir, base, "images/secret.png"); ok {
		t.Fatalf("expected symlink escape to be rejected, got %q", got)
	}
}

func TestPageHandlerCheckSlugVisibilityOnlyAllowsExplicitUserOrAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newPageHandlerWithCustomMenuRaw(t, `[
		{"id":"public-page","url":"md:public","visibility":"user"},
		{"id":"admin-page","url":"md:admin","visibility":"admin"},
		{"id":"invalid-page","url":"md:invalid","visibility":"partner"},
		{"id":"missing-page","url":"md:missing"}
	]`)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/pages/public", nil)

	if !h.checkSlugVisibility(c, "public") {
		t.Fatal("expected explicit user page to be visible")
	}
	if h.checkSlugVisibility(c, "invalid") {
		t.Fatal("expected invalid visibility to be hidden")
	}
	if h.checkSlugVisibility(c, "missing") {
		t.Fatal("expected missing visibility to be hidden")
	}
	if h.checkImageSlugVisibility(c, "invalid") {
		t.Fatal("expected invalid image visibility to be hidden")
	}

	c.Set(string(middleware2.ContextKeyUserRole), "admin")
	if !h.checkSlugVisibility(c, "admin") {
		t.Fatal("expected explicit admin page to be visible to admins")
	}
	if h.checkImageSlugVisibility(c, "admin") {
		t.Fatal("expected admin page images to remain hidden from unauthenticated image requests")
	}
}

func TestPageHandlerCheckSlugVisibilityAllowsUserDuplicateSlugAfterAdminEntry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newPageHandlerWithCustomMenuRaw(t, `[
		{"id":"admin-help","url":"md:help","visibility":"admin"},
		{"id":"user-help","url":"md:help","visibility":"user"}
	]`)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/pages/help", nil)

	if !h.checkSlugVisibility(c, "help") {
		t.Fatal("expected user-visible duplicate slug to remain visible regardless of menu order")
	}
	if !h.checkImageSlugVisibility(c, "help") {
		t.Fatal("expected images for user-visible duplicate slug to remain visible regardless of menu order")
	}
}

func TestPageHandlerListPagesIncludesNestedMarkdownSlugs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dataDir := t.TempDir()
	pagesDir := filepath.Join(dataDir, "pages")
	if err := os.MkdirAll(filepath.Join(pagesDir, "help"), 0755); err != nil {
		t.Fatalf("create nested page dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pagesDir, "public.md"), []byte("# Public"), 0644); err != nil {
		t.Fatalf("create public page: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pagesDir, "help", "intro.md"), []byte("# Intro"), 0644); err != nil {
		t.Fatalf("create nested page: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pagesDir, "help", ".hidden.md"), []byte("# Hidden"), 0644); err != nil {
		t.Fatalf("create invalid page: %v", err)
	}

	router := gin.New()
	h := NewPageHandler(dataDir, nil)
	router.GET("/api/v1/pages", h.ListPages)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pages", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var envelope struct {
		Data []string `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !pageTestContainsString(envelope.Data, "public") {
		t.Fatalf("expected top-level slug in %v", envelope.Data)
	}
	if !pageTestContainsString(envelope.Data, "help/intro") {
		t.Fatalf("expected nested slug in %v", envelope.Data)
	}
	if pageTestContainsString(envelope.Data, "help/.hidden") {
		t.Fatalf("expected invalid slug to be omitted from %v", envelope.Data)
	}
}

func pageTestContainsString(items []string, needle string) bool {
	for _, item := range items {
		if item == needle {
			return true
		}
	}
	return false
}

type pageSettingRepoStub struct {
	values map[string]string
}

func (s *pageSettingRepoStub) Get(context.Context, string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (s *pageSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	return s.values[key], nil
}

func (s *pageSettingRepoStub) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (s *pageSettingRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *pageSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *pageSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *pageSettingRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

func newPageHandlerWithCustomMenuRaw(t *testing.T, raw string) *PageHandler {
	t.Helper()

	repo := &pageSettingRepoStub{values: map[string]string{
		service.SettingKeyCustomMenuItems: raw,
	}}
	return &PageHandler{
		pagesDir:       t.TempDir(),
		settingService: service.NewSettingService(repo, &config.Config{}),
	}
}

type pageRouteSettingRepoStub struct {
	values map[string]string
}

func (s *pageRouteSettingRepoStub) Get(context.Context, string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (s *pageRouteSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	return s.values[key], nil
}

func (s *pageRouteSettingRepoStub) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (s *pageRouteSettingRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *pageRouteSettingRepoStub) SetMultiple(_ context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = make(map[string]string, len(settings))
	}
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *pageRouteSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *pageRouteSettingRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

func TestRegisterPageRoutesBackendModeBlocksNonAdminPageContent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dataDir := t.TempDir()
	pagesDir := filepath.Join(dataDir, "pages")
	if err := os.MkdirAll(pagesDir, 0755); err != nil {
		t.Fatalf("create pages dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pagesDir, "public.md"), []byte("# Public"), 0644); err != nil {
		t.Fatalf("create markdown page: %v", err)
	}

	repo := &pageRouteSettingRepoStub{values: map[string]string{
		service.SettingKeyCustomMenuItems:    `[{"id":"public-page","url":"md:public","visibility":"user"}]`,
		service.SettingKeyBackendModeEnabled: "false",
	}}
	settingSvc := service.NewSettingService(repo, &config.Config{})
	if err := settingSvc.UpdateSettings(context.Background(), &service.SystemSettings{BackendModeEnabled: true}); err != nil {
		t.Fatalf("enable backend mode: %v", err)
	}
	t.Cleanup(func() {
		_ = settingSvc.UpdateSettings(context.Background(), &service.SystemSettings{BackendModeEnabled: false})
	})
	repo.values[service.SettingKeyCustomMenuItems] = `[{"id":"public-page","url":"md:public","visibility":"user"}]`

	router := gin.New()
	v1 := router.Group("/api/v1")
	RegisterPageRoutes(
		v1,
		dataDir,
		func(c *gin.Context) {
			c.Set(string(middleware2.ContextKeyUserRole), "user")
			c.Next()
		},
		func(c *gin.Context) {
			c.Set(string(middleware2.ContextKeyUserRole), "admin")
			c.Next()
		},
		settingSvc,
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pages/public", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestPageImagesBackendModeDoesNotExposeUserVisibleAssets(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dataDir := t.TempDir()
	imageDir := filepath.Join(dataDir, "pages", "public")
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		t.Fatalf("create image dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(imageDir, "logo.png"), []byte("fake"), 0644); err != nil {
		t.Fatalf("create image: %v", err)
	}

	repo := &pageRouteSettingRepoStub{values: map[string]string{
		service.SettingKeyCustomMenuItems:    `[{"id":"public-page","url":"md:public","visibility":"user"}]`,
		service.SettingKeyBackendModeEnabled: "false",
	}}
	settingSvc := service.NewSettingService(repo, &config.Config{})
	if err := settingSvc.UpdateSettings(context.Background(), &service.SystemSettings{BackendModeEnabled: true}); err != nil {
		t.Fatalf("enable backend mode: %v", err)
	}
	t.Cleanup(func() {
		_ = settingSvc.UpdateSettings(context.Background(), &service.SystemSettings{BackendModeEnabled: false})
	})
	repo.values[service.SettingKeyCustomMenuItems] = `[{"id":"public-page","url":"md:public","visibility":"user"}]`

	router := gin.New()
	v1 := router.Group("/api/v1")
	RegisterPageRoutes(
		v1,
		dataDir,
		func(c *gin.Context) {
			c.Next()
		},
		func(c *gin.Context) {
			c.Set(string(middleware2.ContextKeyUserRole), "admin")
			c.Next()
		},
		settingSvc,
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pages/public/images/logo.png", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func mustEvalSymlinks(t *testing.T, path string) string {
	t.Helper()

	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("eval symlinks for %q: %v", path, err)
	}
	return realPath
}
