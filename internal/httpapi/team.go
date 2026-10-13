package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mulki214/gdcpay_test_project/internal/response"
	"github.com/mulki214/gdcpay_test_project/internal/team"
)

type TeamHandler struct {
	service *team.Service
}

func NewTeamHandler(service *team.Service) *TeamHandler {
	return &TeamHandler{service: service}
}

type createTeamRequest struct {
	Name string `json:"name" binding:"required"`
}

type addTeamMemberRequest struct {
	Email string `json:"email" binding:"required"`
}

func (h *TeamHandler) Create(c *gin.Context) {
	var request createTeamRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.WriteError(c, http.StatusBadRequest, "INVALID_REQUEST", "A team name is required")
		return
	}
	created, err := h.service.Create(c.Request.Context(), authenticatedUserID(c), team.CreateInput{Name: request.Name})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *TeamHandler) AddMember(c *gin.Context) {
	teamID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.WriteError(c, http.StatusBadRequest, "INVALID_TEAM_ID", "Team ID must be a UUID")
		return
	}
	var request addTeamMemberRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.WriteError(c, http.StatusBadRequest, "INVALID_REQUEST", "A member email is required")
		return
	}
	member, err := h.service.AddMember(c.Request.Context(), authenticatedUserID(c), teamID, team.AddMemberInput{Email: request.Email})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, member)
}

func (h *TeamHandler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, team.ErrInvalidInput):
		response.WriteError(c, http.StatusBadRequest, "INVALID_INPUT", "The team data is invalid")
	case errors.Is(err, team.ErrNotFound):
		response.WriteError(c, http.StatusNotFound, "TEAM_NOT_FOUND", "The team was not found")
	case errors.Is(err, team.ErrForbidden):
		response.WriteError(c, http.StatusForbidden, "TEAM_OWNER_REQUIRED", "Only the team owner can add members")
	case errors.Is(err, team.ErrUserNotFound):
		response.WriteError(c, http.StatusNotFound, "USER_NOT_FOUND", "The registered user was not found")
	default:
		response.Internal(c)
	}
}
