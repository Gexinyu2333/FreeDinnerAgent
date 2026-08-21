package api

import (
	"errors"
	"net/http"

	"freedinner/backend/internal/auth"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	auth *auth.Service
}

func NewAuthHandler(authService *auth.Service) *AuthHandler {
	return &AuthHandler{auth: authService}
}

type registerRequest struct {
	Username    string  `json:"username" binding:"required"`
	Password    string  `json:"password" binding:"required"`
	DisplayName *string `json:"display_name"`
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	result, err := h.auth.Register(c.Request.Context(), auth.RegisterInput{
		Username:    req.Username,
		Password:    req.Password,
		DisplayName: req.DisplayName,
		UserAgent:   c.GetHeader("User-Agent"),
	})
	if err != nil {
		writeAuthError(c, err)
		return
	}

	OK(c, result)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	result, err := h.auth.Login(c.Request.Context(), auth.LoginInput{
		Username:  req.Username,
		Password:  req.Password,
		UserAgent: c.GetHeader("User-Agent"),
	})
	if err != nil {
		writeAuthError(c, err)
		return
	}

	OK(c, result)
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	user, err := h.auth.CurrentUser(c.Request.Context(), userID)
	if err != nil {
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load user")
		return
	}

	OK(c, user)
}

func writeAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, auth.ErrInvalidInput):
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "username must be 3-64 characters and password at least 8 characters")
	case errors.Is(err, auth.ErrUsernameTaken):
		Error(c, http.StatusConflict, "USERNAME_TAKEN", "username already exists")
	case errors.Is(err, auth.ErrInvalidCredential):
		Error(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid username or password")
	case errors.Is(err, auth.ErrUserDisabled):
		Error(c, http.StatusForbidden, "USER_DISABLED", "user is disabled")
	default:
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "authentication failed")
	}
}
