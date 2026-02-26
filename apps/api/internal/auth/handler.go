package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.con/falasefemi2/taskflow/api/db/generated"
	"github.con/falasefemi2/taskflow/api/internal/config"
	"golang.org/x/crypto/bcrypt"
)

const (
	accessTokenTTL   = 15 * time.Minute
	refreshTokenTTL  = 7 * 24 * time.Hour
	resetPasswordTTL = time.Hour
	refreshCookieKey = "refresh_token"
)

type Handler struct {
	queries *db.Queries
	cfg     *config.Config
}

func NewHandler(queries *db.Queries, cfg *config.Config) *Handler {
	return &Handler{queries: queries, cfg: cfg}
}

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

type authResponse struct {
	AccessToken string       `json:"access_token"`
	User        userResponse `json:"user"`
}

type userResponse struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	Email      string     `json:"email"`
	AvatarURL  *string    `json:"avatar_url"`
	IsVerified bool       `json:"is_verified"`
	Status     string     `json:"status"`
	LastLogin  *time.Time `json:"last_login_at"`
	CreatedAt  *time.Time `json:"created_at"`
	UpdatedAt  *time.Time `json:"updated_at"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Name == "" || req.Email == "" || len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "name, email and password(min 8 chars) are required")
		return
	}

	existing, err := h.queries.GetUserByEmail(r.Context(), strings.ToLower(req.Email))
	if err == nil && existing.ID != uuid.Nil {
		writeError(w, http.StatusConflict, "email already in use")
		return
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusInternalServerError, "failed to check existing user")
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user, err := h.queries.CreateUser(r.Context(), db.CreateUserParams{
		Name:         req.Name,
		Email:        strings.ToLower(req.Email),
		PasswordHash: string(passwordHash),
		AvatarUrl:    pgtype.Text{},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	resp, refreshToken, err := h.issueSession(r.Context(), user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}
	h.setRefreshCookie(w, refreshToken)

	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	user, err := h.queries.GetUserByEmail(r.Context(), strings.ToLower(req.Email))
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := h.queries.UpdateUserLastLogin(r.Context(), user.ID); err == nil {
		user.LastLoginAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	}

	resp, refreshToken, err := h.issueSession(r.Context(), user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}
	h.setRefreshCookie(w, refreshToken)

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	rawRefreshToken, err := h.getRefreshCookie(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing refresh token")
		return
	}

	tokenHash := hashToken(rawRefreshToken)
	session, err := h.queries.GetRefreshTokenByHash(r.Context(), tokenHash)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	if !session.ExpiresAt.Valid || session.ExpiresAt.Time.Before(time.Now().UTC()) {
		_ = h.queries.DeleteRefreshTokenByHash(r.Context(), tokenHash)
		h.clearRefreshCookie(w)
		writeError(w, http.StatusUnauthorized, "refresh token expired")
		return
	}

	user, err := h.queries.GetUserByID(r.Context(), session.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "user not found")
		return
	}

	_ = h.queries.DeleteRefreshTokenByHash(r.Context(), tokenHash)

	resp, newRefreshToken, err := h.issueSession(r.Context(), user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to refresh session")
		return
	}
	h.setRefreshCookie(w, newRefreshToken)

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	rawRefreshToken, err := h.getRefreshCookie(r)
	if err == nil && rawRefreshToken != "" {
		_ = h.queries.DeleteRefreshTokenByHash(r.Context(), hashToken(rawRefreshToken))
	}

	h.clearRefreshCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	user, err := h.queries.GetUserByEmail(r.Context(), strings.ToLower(req.Email))
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{
			"message": "if an account exists for this email, a reset link has been sent",
		})
		return
	}

	resetToken, err := h.generateJWT(user.ID, "reset", resetPasswordTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create reset token")
		return
	}

	err = h.queries.SetUserResetToken(r.Context(), db.SetUserResetTokenParams{
		ID: user.ID,
		ResetToken: pgtype.Text{
			String: hashToken(resetToken),
			Valid:  true,
		},
		ResetTokenExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().UTC().Add(resetPasswordTTL),
			Valid: true,
		},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to set reset token")
		return
	}

	resp := map[string]string{
		"message": "if an account exists for this email, a reset link has been sent",
	}
	if h.cfg.Primary.Env != "production" {
		resp["reset_token"] = resetToken
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Token == "" || len(req.NewPassword) < 8 {
		writeError(w, http.StatusBadRequest, "token and new_password(min 8 chars) are required")
		return
	}

	userID, tokenType, err := h.parseJWT(req.Token)
	if err != nil || tokenType != "reset" {
		writeError(w, http.StatusBadRequest, "invalid or expired token")
		return
	}

	user, err := h.queries.GetUserByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid or expired token")
		return
	}

	if !user.ResetToken.Valid || user.ResetToken.String != hashToken(req.Token) {
		writeError(w, http.StatusBadRequest, "invalid or expired token")
		return
	}
	if !user.ResetTokenExpiresAt.Valid || user.ResetTokenExpiresAt.Time.Before(time.Now().UTC()) {
		writeError(w, http.StatusBadRequest, "invalid or expired token")
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	if err := h.queries.UpdateUserPassword(r.Context(), db.UpdateUserPasswordParams{
		ID:           user.ID,
		PasswordHash: string(passwordHash),
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reset password")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "password reset successful"})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		writeError(w, http.StatusUnauthorized, "missing bearer token")
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	userID, tokenType, err := h.parseJWT(token)
	if err != nil || tokenType != "access" {
		writeError(w, http.StatusUnauthorized, "invalid access token")
		return
	}

	user, err := h.queries.GetUserByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]userResponse{"user": toUserResponse(user)})
}

func (h *Handler) issueSession(ctx context.Context, user db.User) (authResponse, string, error) {
	accessToken, err := h.generateJWT(user.ID, "access", accessTokenTTL)
	if err != nil {
		return authResponse{}, "", err
	}

	refreshToken, err := generateOpaqueToken(32)
	if err != nil {
		return authResponse{}, "", err
	}

	if _, err := h.queries.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID:    user.ID,
		TokenHash: hashToken(refreshToken),
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().UTC().Add(refreshTokenTTL), Valid: true},
	}); err != nil {
		return authResponse{}, "", err
	}

	return authResponse{
		AccessToken: accessToken,
		User:        toUserResponse(user),
	}, refreshToken, nil
}

func (h *Handler) setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieKey,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.Primary.Env == "production",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(refreshTokenTTL.Seconds()),
	})
}

func (h *Handler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieKey,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.Primary.Env == "production",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func (h *Handler) getRefreshCookie(r *http.Request) (string, error) {
	c, err := r.Cookie(refreshCookieKey)
	if err != nil || c.Value == "" {
		return "", errors.New("missing cookie")
	}
	return c.Value, nil
}

func (h *Handler) generateJWT(userID uuid.UUID, tokenType string, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}
	payload := map[string]any{
		"sub": userID.String(),
		"typ": tokenType,
		"iat": now.Unix(),
		"exp": now.Add(ttl).Unix(),
	}

	headerBytes, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerBytes)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	signingInput := encodedHeader + "." + encodedPayload

	mac := hmac.New(sha256.New, []byte(h.cfg.Auth.JWTSecret))
	_, _ = mac.Write([]byte(signingInput))
	signature := mac.Sum(nil)

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (h *Handler) parseJWT(token string) (uuid.UUID, string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return uuid.Nil, "", errors.New("invalid token")
	}

	signingInput := parts[0] + "." + parts[1]
	expectedMAC := hmac.New(sha256.New, []byte(h.cfg.Auth.JWTSecret))
	_, _ = expectedMAC.Write([]byte(signingInput))
	expectedSig := expectedMAC.Sum(nil)

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(signature, expectedSig) {
		return uuid.Nil, "", errors.New("invalid token")
	}

	var payload struct {
		Sub string `json:"sub"`
		Typ string `json:"typ"`
		Exp int64  `json:"exp"`
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return uuid.Nil, "", err
	}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return uuid.Nil, "", err
	}

	if payload.Exp < time.Now().UTC().Unix() {
		return uuid.Nil, "", errors.New("token expired")
	}
	userID, err := uuid.Parse(payload.Sub)
	if err != nil {
		return uuid.Nil, "", err
	}
	return userID, payload.Typ, nil
}

func generateOpaqueToken(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func toUserResponse(user db.User) userResponse {
	return userResponse{
		ID:         user.ID,
		Name:       user.Name,
		Email:      user.Email,
		AvatarURL:  textPtr(user.AvatarUrl),
		IsVerified: user.IsVerified,
		Status:     user.Status,
		LastLogin:  timePtr(user.LastLoginAt),
		CreatedAt:  timePtr(user.CreatedAt),
		UpdatedAt:  timePtr(user.UpdatedAt),
	}
}

func textPtr(v pgtype.Text) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func timePtr(v pgtype.Timestamptz) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time
	return &t
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid request body")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
