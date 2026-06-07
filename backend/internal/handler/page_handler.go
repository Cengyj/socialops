package handler

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Wei-Shaw/socialops/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
)

var validSlugPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*(/[a-zA-Z0-9][a-zA-Z0-9._-]*)*$`)

const (
	maxPageSlugLength = 64
	maxPageFileSize   = 1 << 20 // 1MB
)

type PageHandler struct {
	pagesDir       string
	settingService *service.SettingService
}

func NewPageHandler(dataDir string, settingService *service.SettingService) *PageHandler {
	pagesDir := filepath.Join(dataDir, "pages")
	_ = os.MkdirAll(pagesDir, 0755)
	return &PageHandler{pagesDir: pagesDir, settingService: settingService}
}

// GetPageContent serves raw markdown content for a given slug.
// GET /api/v1/pages/:slug
func (h *PageHandler) GetPageContent(c *gin.Context) {
	slugRelPath, normalizedSlug, ok := cleanPageSlugPath(c.Param("slug"))
	if !ok {
		response.BadRequest(c, "Invalid page slug")
		return
	}

	// Visibility check: slug must be configured in custom_menu_items
	// and the user must have permission based on visibility setting
	if !h.checkSlugVisibility(c, normalizedSlug) {
		c.JSON(http.StatusNotFound, gin.H{"error": "page not found"})
		return
	}

	cleanedPagesDir := filepath.Clean(h.pagesDir)
	cleaned := filepath.Clean(filepath.Join(cleanedPagesDir, slugRelPath+".md"))
	if !isPathWithinBase(cleaned, cleanedPagesDir) {
		response.BadRequest(c, "Invalid page slug")
		return
	}

	info, err := os.Stat(cleaned)
	if err != nil || info.IsDir() {
		c.JSON(http.StatusNotFound, gin.H{"error": "page not found"})
		return
	}
	if !pageMarkdownPathWithinBase(h.pagesDir, cleaned) {
		c.JSON(http.StatusNotFound, gin.H{"error": "page not found"})
		return
	}
	if info.Size() > maxPageFileSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "page too large"})
		return
	}

	content, err := os.ReadFile(cleaned)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read page"})
		return
	}

	c.Data(http.StatusOK, "text/markdown; charset=utf-8", content)
}

// ListPages returns available page slugs.
// GET /api/v1/pages
func (h *PageHandler) ListPages(c *gin.Context) {
	slugs := make([]string, 0)
	err := filepath.WalkDir(h.pagesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".md") {
			return nil
		}
		rel, err := filepath.Rel(h.pagesDir, path)
		if err != nil {
			return err
		}
		slug := strings.TrimSuffix(filepath.ToSlash(rel), ".md")
		if _, _, ok := cleanPageSlugPath(slug); ok {
			slugs = append(slugs, slug)
		}
		return nil
	})
	if err != nil {
		response.Success(c, []string{})
		return
	}

	response.Success(c, slugs)
}

// ServePageImage serves images from data/pages/{slug}/ directory.
// GET /api/v1/pages/:slug/images/*filename
// No JWT required (browser img tags can't carry tokens), but visibility is checked.
func (h *PageHandler) ServePageImage(c *gin.Context) {
	slugRelPath, normalizedSlug, ok := cleanPageSlugPath(c.Param("slug"))
	filename := c.Param("filename")
	filename = strings.TrimPrefix(filename, "/")

	if !ok {
		c.Status(http.StatusNotFound)
		return
	}

	if !h.checkImageSlugVisibility(c, normalizedSlug) {
		c.Status(http.StatusNotFound)
		return
	}

	imagesDir := filepath.Join(h.pagesDir, slugRelPath)
	cleaned, ok := resolvePageImagePath(h.pagesDir, imagesDir, filename)
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}

	info, err := os.Stat(cleaned)
	if err != nil || info.IsDir() {
		c.Status(http.StatusNotFound)
		return
	}

	c.File(cleaned)
}

func resolvePageImagePath(pagesDir, imagesDir, filename string) (string, bool) {
	relPath, ok := cleanPageImageRelativePath(filename)
	if !ok {
		return "", false
	}

	cleanedPagesDir := filepath.Clean(pagesDir)
	cleanedImagesDir := filepath.Clean(imagesDir)
	cleanedTarget := filepath.Clean(filepath.Join(cleanedImagesDir, relPath))
	if !isPathWithinBase(cleanedTarget, cleanedImagesDir) {
		return "", false
	}

	realPagesDir, err := filepath.EvalSymlinks(cleanedPagesDir)
	if err != nil {
		return "", false
	}
	realImagesDir, err := filepath.EvalSymlinks(cleanedImagesDir)
	if err != nil || !isPathWithinBase(realImagesDir, realPagesDir) {
		return "", false
	}
	realTarget, err := filepath.EvalSymlinks(cleanedTarget)
	if err != nil || !isPathWithinBase(realTarget, realImagesDir) {
		return "", false
	}
	return realTarget, true
}

func cleanPageImageRelativePath(filename string) (string, bool) {
	if filename == "" {
		return "", false
	}
	if strings.HasPrefix(filename, "/") {
		return "", false
	}
	decoded, err := url.PathUnescape(filename)
	if err != nil {
		return "", false
	}
	if decoded == "" || strings.HasPrefix(decoded, "/") || strings.Contains(decoded, "\\") || strings.ContainsRune(decoded, 0) {
		return "", false
	}

	parts := make([]string, 0)
	for _, part := range strings.Split(decoded, "/") {
		switch part {
		case "", ".":
			continue
		case "..":
			return "", false
		default:
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return "", false
	}

	relPath := filepath.Join(parts...)
	if filepath.IsAbs(relPath) || filepath.VolumeName(relPath) != "" {
		return "", false
	}
	return relPath, true
}

func cleanPageSlugPath(slug string) (string, string, bool) {
	if slug == "" || len(slug) > maxPageSlugLength {
		return "", "", false
	}
	decoded, err := url.PathUnescape(slug)
	if err != nil {
		return "", "", false
	}
	if decoded == "" ||
		len(decoded) > maxPageSlugLength ||
		strings.HasPrefix(decoded, "/") ||
		strings.Contains(decoded, "\\") ||
		strings.ContainsRune(decoded, 0) ||
		strings.Contains(decoded, "..") ||
		!validSlugPattern.MatchString(decoded) {
		return "", "", false
	}

	parts := strings.Split(decoded, "/")
	relPath := filepath.Join(parts...)
	if filepath.IsAbs(relPath) || filepath.VolumeName(relPath) != "" {
		return "", "", false
	}
	return relPath, decoded, true
}

func pageMarkdownPathWithinBase(pagesDir, filePath string) bool {
	realPagesDir, err := filepath.EvalSymlinks(filepath.Clean(pagesDir))
	if err != nil {
		return false
	}
	realFilePath, err := filepath.EvalSymlinks(filepath.Clean(filePath))
	if err != nil {
		return false
	}
	return isPathWithinBase(realFilePath, realPagesDir)
}

func isPathWithinBase(path, base string) bool {
	rel, err := filepath.Rel(filepath.Clean(base), filepath.Clean(path))
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// findSlugVisibility looks up the slug in custom_menu_items and returns (visibility, found).
func (h *PageHandler) findSlugVisibility(c *gin.Context, slug string) (string, bool) {
	if h.settingService == nil {
		return "", false
	}

	raw := h.settingService.GetCustomMenuItemsRaw(c.Request.Context())
	if raw == "" || raw == "[]" {
		return "", false
	}

	var items []struct {
		URL        string `json:"url"`
		PageSlug   string `json:"page_slug"`
		Visibility string `json:"visibility"`
	}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return "", false
	}

	adminVisible := false
	for _, item := range items {
		itemSlug := item.PageSlug
		if itemSlug == "" && strings.HasPrefix(item.URL, "md:") {
			itemSlug = strings.TrimPrefix(item.URL, "md:")
		}
		if itemSlug != slug {
			continue
		}
		if item.Visibility == "user" {
			return "user", true
		}
		if item.Visibility == "admin" {
			adminVisible = true
		}
	}
	if adminVisible {
		return "admin", true
	}
	return "", false
}

// checkSlugVisibility verifies the slug is configured in custom_menu_items
// and the authenticated user has permission to view it.
func (h *PageHandler) checkSlugVisibility(c *gin.Context, slug string) bool {
	visibility, found := h.findSlugVisibility(c, slug)
	if !found {
		return false
	}
	switch visibility {
	case "user":
		return true
	case "admin":
		role, _ := middleware2.GetUserRoleFromContext(c)
		return role == "admin"
	default:
		return false
	}
}

// checkImageSlugVisibility checks visibility for image requests (no JWT available).
// Only allows user-visible pages; admin-only pages are blocked.
func (h *PageHandler) checkImageSlugVisibility(c *gin.Context, slug string) bool {
	if h.settingService != nil && h.settingService.IsBackendModeEnabled(c.Request.Context()) {
		return false
	}
	visibility, found := h.findSlugVisibility(c, slug)
	if !found {
		return false
	}
	return visibility == "user"
}

// RegisterPageRoutes registers page routes on a router group.
func RegisterPageRoutes(v1 *gin.RouterGroup, dataDir string, jwtAuth gin.HandlerFunc, adminAuth gin.HandlerFunc, settingService *service.SettingService) {
	h := NewPageHandler(dataDir, settingService)

	// Authenticated page content (JWT required + visibility check)
	pages := v1.Group("/pages")
	pages.Use(jwtAuth)
	pages.Use(middleware2.BackendModeUserGuard(settingService))
	{
		pages.GET("/:slug", h.GetPageContent)
	}

	// Images: no JWT (browser img tags can't carry tokens), visibility check in handler
	pageImages := v1.Group("/pages")
	{
		pageImages.GET("/:slug/images/*filename", h.ServePageImage)
	}

	// Admin-only: list all available pages
	adminPages := v1.Group("/pages")
	adminPages.Use(adminAuth)
	{
		adminPages.GET("", h.ListPages)
	}
}
