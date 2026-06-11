package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"taskmanager/internal/auth"
	"taskmanager/internal/httpx"
	"taskmanager/internal/middleware"
	"taskmanager/internal/repository"
	"taskmanager/internal/validation"
)

const tokenTTL = 7 * 24 * time.Hour

type AuthHandler struct {
	users     *repository.UserRepository
	jwtSecret string
}

func NewAuthHandler(users *repository.UserRepository, jwtSecret string) *AuthHandler {
	return &AuthHandler{users: users, jwtSecret: jwtSecret}
}

type signupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string      `json:"token"`
	User  userPayload `json:"user"`
}

type userPayload struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	errs := validation.Errors{}
	if !validation.IsValidEmail(req.Email) {
		errs.Add("email", "must be a valid email address")
	}
	if len(req.Password) < 8 {
		errs.Add("password", "must be at least 8 characters")
	}
	if errs.HasErrors() {
		httpx.ValidationError(w, errs)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to process password")
		return
	}

	user, err := h.users.Create(r.Context(), req.Email, hash)
	if err != nil {
		if err == repository.ErrDuplicate {
			httpx.Error(w, http.StatusConflict, "an account with this email already exists")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "failed to create account")
		return
	}

	token, err := auth.GenerateToken(h.jwtSecret, user.ID, user.Email, user.Role, tokenTTL)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	httpx.JSON(w, http.StatusCreated, authResponse{
		Token: token,
		User:  userPayload{ID: user.ID, Email: user.Email, Role: user.Role},
	})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if validation.IsBlank(req.Email) || validation.IsBlank(req.Password) {
		httpx.Error(w, http.StatusBadRequest, "email and password are required")
		return
	}

	user, err := h.users.GetByEmail(r.Context(), req.Email)
	if err != nil {
		if err == repository.ErrNotFound {
			httpx.Error(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "failed to log in")
		return
	}

	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		httpx.Error(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, err := auth.GenerateToken(h.jwtSecret, user.ID, user.Email, user.Role, tokenTTL)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	httpx.JSON(w, http.StatusOK, authResponse{
		Token: token,
		User:  userPayload{ID: user.ID, Email: user.Email, Role: user.Role},
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	user, err := h.users.GetByID(r.Context(), userID)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "user not found")
		return
	}

	httpx.JSON(w, http.StatusOK, userPayload{ID: user.ID, Email: user.Email, Role: user.Role})
}
