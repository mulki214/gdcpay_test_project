package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mulki214/gdcpay_test_project/internal/middleware"
	"github.com/mulki214/gdcpay_test_project/internal/response"
	"github.com/mulki214/gdcpay_test_project/internal/task"
)

type TaskHandler struct{ service *task.Service }

func NewTaskHandler(service *task.Service) *TaskHandler { return &TaskHandler{service: service} }

type createTaskRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
}

type updateTaskRequest struct {
	Title       string      `json:"title" binding:"required"`
	Description string      `json:"description"`
	Status      task.Status `json:"status" binding:"required"`
}

func (h *TaskHandler) Create(c *gin.Context) {
	key, ok := idempotencyKey(c)
	if !ok {
		return
	}
	var request createTaskRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.WriteError(c, http.StatusBadRequest, "INVALID_REQUEST", "A task title is required")
		return
	}
	item, err := h.service.CreateIdempotent(c.Request.Context(), authenticatedUserID(c), key, task.CreateInput{Title: request.Title, Description: request.Description})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *TaskHandler) List(c *gin.Context) {
	page, errPage := queryInt(c, "page", 1)
	limit, errLimit := queryInt(c, "limit", 20)
	if errPage != nil || errLimit != nil {
		response.WriteError(c, http.StatusBadRequest, "INVALID_INPUT", "Page and limit must be integers")
		return
	}
	result, err := h.service.List(c.Request.Context(), authenticatedUserID(c), task.ListFilter{
		Page: page, Limit: limit, Status: task.Status(c.Query("status")), Search: c.Query("search"),
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *TaskHandler) Get(c *gin.Context) {
	id, ok := taskID(c)
	if !ok {
		return
	}
	item, err := h.service.Get(c.Request.Context(), authenticatedUserID(c), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *TaskHandler) Update(c *gin.Context) {
	id, ok := taskID(c)
	if !ok {
		return
	}
	var request updateTaskRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.WriteError(c, http.StatusBadRequest, "INVALID_REQUEST", "Title and a valid status are required")
		return
	}
	item, err := h.service.Update(c.Request.Context(), authenticatedUserID(c), id, task.UpdateInput{Title: request.Title, Description: request.Description, Status: request.Status})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *TaskHandler) Delete(c *gin.Context) {
	id, ok := taskID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), authenticatedUserID(c), id); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *TaskHandler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, task.ErrInvalidInput):
		response.WriteError(c, http.StatusBadRequest, "INVALID_INPUT", "The task data or query parameters are invalid")
	case errors.Is(err, task.ErrNotFound):
		response.WriteError(c, http.StatusNotFound, "TASK_NOT_FOUND", "The task was not found")
	case errors.Is(err, task.ErrIdempotencyKeyReused):
		response.WriteError(c, http.StatusConflict, "IDEMPOTENCY_KEY_REUSED", "This idempotency key was already used with a different request")
	default:
		response.Internal(c)
	}
}

func idempotencyKey(c *gin.Context) (uuid.UUID, bool) {
	value := c.GetHeader("Idempotency-Key")
	key, err := uuid.Parse(value)
	if err != nil || len(value) != 36 || key.String() != strings.ToLower(value) {
		response.WriteError(c, http.StatusBadRequest, "INVALID_IDEMPOTENCY_KEY", "Idempotency-Key must be a UUID")
		return uuid.Nil, false
	}
	return key, true
}

func authenticatedUserID(c *gin.Context) uuid.UUID {
	id, _ := c.Get(middleware.AuthenticatedUserIDKey)
	return id.(uuid.UUID)
}

func taskID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.WriteError(c, http.StatusBadRequest, "INVALID_TASK_ID", "Task ID must be a UUID")
		return uuid.Nil, false
	}
	return id, true
}

func queryInt(c *gin.Context, key string, fallback int) (int, error) {
	value := c.Query(key)
	if value == "" {
		return fallback, nil
	}
	return strconv.Atoi(value)
}
