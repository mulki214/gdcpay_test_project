package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mulki214/gdcpay_test_project/internal/auth"
	"github.com/mulki214/gdcpay_test_project/internal/response"
)

type AuthHandler struct {
	service *auth.Service
}

func NewAuthHandler(service *auth.Service) *AuthHandler {
	return &AuthHandler{service: service}
}

type credentialsRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type authResponse struct {
	User        userResponse `json:"user"`
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresAt   string       `json:"expires_at"`
}

type userResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var request credentialsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.WriteError(c, http.StatusBadRequest, "INVALID_REQUEST", "Email and password are required")
		return
	}
	result, err := h.service.Register(c.Request.Context(), request.Email, request.Password)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidInput):
			response.WriteError(c, http.StatusBadRequest, "INVALID_INPUT", "Provide a valid email and a password between 8 and 72 bytes")
		case errors.Is(err, auth.ErrEmailAlreadyRegistered):
			response.WriteError(c, http.StatusConflict, "EMAIL_ALREADY_REGISTERED", "An account with this email already exists")
		default:
			response.Internal(c)
		}
		return
	}
	c.JSON(http.StatusCreated, toAuthResponse(result))
}

func (h *AuthHandler) Login(c *gin.Context) {
	var request credentialsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.WriteError(c, http.StatusBadRequest, "INVALID_REQUEST", "Email and password are required")
		return
	}
	result, err := h.service.Login(c.Request.Context(), request.Email, request.Password)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidCredentials):
			response.WriteError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Email or password is incorrect")
		default:
			response.Internal(c)
		}
		return
	}
	c.JSON(http.StatusOK, toAuthResponse(result))
}

func toAuthResponse(result auth.AuthResult) authResponse {
	return authResponse{
		User:        userResponse{ID: result.User.ID.String(), Email: result.User.Email},
		AccessToken: result.AccessToken,
		TokenType:   "Bearer",
		ExpiresAt:   result.ExpiresAt.UTC().Format(time.RFC3339),
	}
}
