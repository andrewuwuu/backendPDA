package handler

import (
	"encoding/json"
	"net/http"

	"pda-monitor/internal/domain"
	"pda-monitor/internal/httpx"
)

func (h *APIHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req domain.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.userRepo.GetByUsername(ctx, req.Username)
	if err != nil || user == nil {
		httpx.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if !h.userRepo.ValidatePassword(user, req.Password) {
		httpx.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := h.jwtManager.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	httpx.JSON(w, http.StatusOK, domain.LoginResponse{
		Token: token,
		User:  *user,
	})
}

func (h *APIHandler) Logout(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || len(authHeader) < 8 {
		httpx.Error(w, http.StatusBadRequest, "missing authorization header")
		return
	}

	tokenString := authHeader[7:]

	if err := h.jwtManager.InvalidateToken(tokenString); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to invalidate token")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]string{
		"status":  "logged_out",
		"message": "token has been invalidated",
	})
}

func (h *APIHandler) GetJWTInfo(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, h.jwtManager.GetKeyInfo())
}
