package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	UserIDKey   contextKey = "userID"
	UserRoleKey contextKey = "userRole"
)

func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Get token from cookie first
			cookie, err := r.Cookie("access_token")
			if err != nil {
				// fallback to Authorization header
				authHeader := r.Header.Get("Authorization")
				if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
					http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
					return
				}
				cookie = &http.Cookie{Value: strings.TrimPrefix(authHeader, "Bearer ")}
			}

			// 2. Parse and validate token
			token, err := jwt.Parse(cookie.Value, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			// 3. Extract claims and put in context
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, claims["user_id"])
			ctx = context.WithValue(ctx, UserRoleKey, claims["role"])
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(r *http.Request) (string, bool) {
	userID, ok := r.Context().Value(UserIDKey).(string)
	return userID, ok
}

func GetUserRole(r *http.Request) (string, bool) {
	role, ok := r.Context().Value(UserRoleKey).(string)
	return role, ok
}
