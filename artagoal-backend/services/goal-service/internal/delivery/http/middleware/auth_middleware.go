// Package middleware menyediakan middleware HTTP untuk goal-service,
// termasuk autentikasi JWT yang menyuntikkan user_id ke request context.
package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// contextKey adalah tipe unexported sehingga tidak akan bentrok
// dengan key context dari package lain.
type contextKey struct{}

// userIDKey adalah satu-satunya key untuk menyimpan user_id terautentikasi.
var userIDKey = contextKey{}

// ContextWithUserID menyisipkan user_id ke dalam context.
// Diekspos terutama untuk kebutuhan testing handler tanpa token asli.
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFromContext mengambil user_id dari context.
// Mengembalikan ("", false) bila request belum terautentikasi.
func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userIDKey).(string)
	if !ok || v == "" {
		return "", false
	}
	return v, true
}

// AuthMiddleware memverifikasi header "Authorization: Bearer <token>"
// memakai HMAC secret yang diberikan, mengekstrak klaim "user_id"
// (fallback ke "sub" untuk kompatibilitas OIDC/Supabase), lalu
// menyimpannya ke request context sebelum meneruskan ke handler berikutnya.
//
// Respons 401 JSON dikembalikan bila header tidak ada, format salah,
// secret belum dikonfigurasi, atau token tidak sah/kedaluwarsa.
func AuthMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr, ok := bearerToken(r.Header.Get("Authorization"))
			if !ok {
				unauthorized(w, "header Authorization tidak ada atau format invalid (gunakan Bearer <token>)")
				return
			}
			userID, err := parseAndValidate(tokenStr, secret)
			if err != nil {
				unauthorized(w, "token tidak sah atau kedaluwarsa")
				return
			}
			next.ServeHTTP(w, r.WithContext(ContextWithUserID(r.Context(), userID)))
		})
	}
}

// bearerToken memisahkan skema "Bearer" (case-insensitive) dari token.
func bearerToken(header string) (string, bool) {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", false
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}
	return token, true
}

// parseAndValidate mem-parsing token, memverifikasi signature HMAC + masa
// berlaku, lalu mengembalikan klaim user_id.
func parseAndValidate(tokenStr, secret string) (string, error) {
	if secret == "" {
		// Fail-closed: tanpa secret yang terkonfigurasi, tidak ada token
		// yang boleh dianggap valid.
		return "", errors.New("jwt secret belum dikonfigurasi")
	}
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("metode signing token tak didukung")
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return "", errors.New("token tidak valid")
	}
	if s, _ := claims["user_id"].(string); s != "" {
		return s, nil
	}
	// Fallback standar: klaim subject (mis. token Supabase/Auth0).
	if s, _ := claims["sub"].(string); s != "" {
		return s, nil
	}
	return "", errors.New("klaim user_id tidak ditemukan")
}

// unauthorized menulis respons 401 JSON yang konsisten.
func unauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
