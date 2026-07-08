package handlers

import (
	"bus-schedule-lounge/internal/middleware"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type AuthHandler struct {
	jwtSecret                 string
	jwtRefreshSecret          string
	accessTokenExpirySeconds  int64
	refreshTokenExpirySeconds int64
	adminUser                 string
	adminPass                 string
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewAuthHandler(jwtSecret, jwtRefreshSecret, adminUser, adminPass string, accessTokenExpirySeconds, refreshTokenExpirySeconds int64) *AuthHandler {
	return &AuthHandler{
		jwtSecret:                 jwtSecret,
		jwtRefreshSecret:          jwtRefreshSecret,
		accessTokenExpirySeconds:  accessTokenExpirySeconds,
		refreshTokenExpirySeconds: refreshTokenExpirySeconds,
		adminUser:                 adminUser,
		adminPass:                 adminPass,
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)

	if username == "" || password == "" {
		respondWithError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	if username != h.adminUser || password != h.adminPass {
		respondWithError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	accessTokenTTL := time.Duration(h.accessTokenExpirySeconds) * time.Second
	refreshTokenTTL := time.Duration(h.refreshTokenExpirySeconds) * time.Second

	accessToken, err := middleware.GenerateAuthToken(username, "super_admin", "access", h.jwtSecret, accessTokenTTL)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate access token")
		return
	}

	refreshToken, err := middleware.GenerateAuthToken(username, "super_admin", "refresh", h.jwtRefreshSecret, refreshTokenTTL)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate refresh token")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"token":              accessToken,
		"refresh_token":      refreshToken,
		"token_type":         "Bearer",
		"expires_in":         h.accessTokenExpirySeconds,
		"refresh_expires_in": h.refreshTokenExpirySeconds,
		"username":           username,
		"role":               "super_admin",
		"authenticatedAt":    time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(authHeader, "Bearer ") {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	claims, err := middleware.ValidateAuthToken(token, h.jwtSecret, "access")
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	exp := int64(0)
	if claims.ExpiresAt != nil {
		exp = claims.ExpiresAt.Unix()
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"username": claims.Username,
		"role":     claims.Role,
		"exp":      exp,
	})
}
