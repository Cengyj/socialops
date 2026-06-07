package handler

import (
	"net/http"

	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
)

type TaskSettingsHandler struct {
	svc *service.TaskSettingsService
}

func NewTaskSettingsHandler(svc *service.TaskSettingsService) *TaskSettingsHandler {
	return &TaskSettingsHandler{svc: svc}
}

func (h *TaskSettingsHandler) ListTemplates(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	items, err := h.svc.ListTemplates(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *TaskSettingsHandler) SaveTemplate(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	var req service.TaskTemplateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	tmpl, err := h.svc.SaveTemplate(c.Request.Context(), subject.UserID, &req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, tmpl)
}

func (h *TaskSettingsHandler) DeleteTemplate(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	if err := h.svc.DeleteTemplate(c.Request.Context(), subject.UserID, c.Param("id")); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, nil)
}

func (h *TaskSettingsHandler) CopyTemplate(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	tmpl, err := h.svc.CopyTemplate(c.Request.Context(), subject.UserID, c.Param("id"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, tmpl)
}

func (h *TaskSettingsHandler) SetDefaultTemplate(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	tmpl, err := h.svc.SetDefaultTemplate(c.Request.Context(), subject.UserID, c.Param("id"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, tmpl)
}

func (h *TaskSettingsHandler) ValidateTemplate(c *gin.Context) {
	if _, ok := middleware2.GetAuthSubjectFromContext(c); !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	var req service.TaskTemplateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, service.ValidateTaskTemplateInput(&req))
}

func (h *TaskSettingsHandler) PreviewTemplateMedia(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	if h.svc == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("TASK_TEMPLATE_SERVICE_UNAVAILABLE", "task template service is unavailable"))
		return
	}
	resolved, err := h.svc.PreviewTemplateMedia(c.Request.Context(), subject.UserID, c.Query("storage_key"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Content-Disposition", `inline; filename="`+resolved.FileName+`"`)
	c.Data(http.StatusOK, resolved.ContentType, resolved.Body)
}
