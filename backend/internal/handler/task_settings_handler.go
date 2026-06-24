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
	subject, ok := taskSettingsAuthSubject(c)
	if !ok {
		return
	}
	svc, ok := h.taskSettingsService(c)
	if !ok {
		return
	}
	items, err := svc.ListTemplates(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *TaskSettingsHandler) SaveTemplate(c *gin.Context) {
	subject, ok := taskSettingsAuthSubject(c)
	if !ok {
		return
	}
	var req service.TaskTemplateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, taskTemplateInputRequiredError())
		return
	}
	svc, ok := h.taskSettingsService(c)
	if !ok {
		return
	}
	tmpl, err := svc.SaveTemplate(c.Request.Context(), subject.UserID, &req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, tmpl)
}

func (h *TaskSettingsHandler) DeleteTemplate(c *gin.Context) {
	subject, ok := taskSettingsAuthSubject(c)
	if !ok {
		return
	}
	svc, ok := h.taskSettingsService(c)
	if !ok {
		return
	}
	if err := svc.DeleteTemplate(c.Request.Context(), subject.UserID, c.Param("id")); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, nil)
}

func (h *TaskSettingsHandler) CopyTemplate(c *gin.Context) {
	subject, ok := taskSettingsAuthSubject(c)
	if !ok {
		return
	}
	svc, ok := h.taskSettingsService(c)
	if !ok {
		return
	}
	tmpl, err := svc.CopyTemplate(c.Request.Context(), subject.UserID, c.Param("id"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, tmpl)
}

func (h *TaskSettingsHandler) SetDefaultTemplate(c *gin.Context) {
	subject, ok := taskSettingsAuthSubject(c)
	if !ok {
		return
	}
	svc, ok := h.taskSettingsService(c)
	if !ok {
		return
	}
	tmpl, err := svc.SetDefaultTemplate(c.Request.Context(), subject.UserID, c.Param("id"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, tmpl)
}

func (h *TaskSettingsHandler) ValidateTemplate(c *gin.Context) {
	if _, ok := taskSettingsAuthSubject(c); !ok {
		return
	}
	var req service.TaskTemplateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, taskTemplateInputRequiredError())
		return
	}
	svc, ok := h.taskSettingsService(c)
	if !ok {
		return
	}
	response.Success(c, svc.ValidateTemplateInput(&req))
}

func (h *TaskSettingsHandler) PreviewTemplateMedia(c *gin.Context) {
	subject, ok := taskSettingsAuthSubject(c)
	if !ok {
		return
	}
	svc, ok := h.taskSettingsService(c)
	if !ok {
		return
	}
	resolved, err := svc.PreviewTemplateMedia(c.Request.Context(), subject.UserID, c.Query("storage_key"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Content-Disposition", inlineMediaContentDisposition(resolved.FileName))
	c.Data(http.StatusOK, resolved.ContentType, resolved.Body)
}

func taskSettingsAuthSubject(c *gin.Context) (middleware2.AuthSubject, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return middleware2.AuthSubject{}, false
	}
	return subject, true
}

func (h *TaskSettingsHandler) taskSettingsService(c *gin.Context) (*service.TaskSettingsService, bool) {
	if h == nil || h.svc == nil {
		response.ErrorFrom(c, taskTemplateServiceUnavailableError())
		return nil, false
	}
	return h.svc, true
}

func taskTemplateServiceUnavailableError() error {
	return infraerrors.ServiceUnavailable("TASK_TEMPLATE_SERVICE_UNAVAILABLE", "task template service is unavailable")
}

func taskTemplateInputRequiredError() error {
	return infraerrors.BadRequest("TASK_TEMPLATE_INPUT_REQUIRED", "task template input is required")
}
