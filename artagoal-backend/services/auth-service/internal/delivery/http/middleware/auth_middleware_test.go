package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-jwt-secret-untuk-unit-test"

// signToken menandatangani token HMAC untuk testing.
func signToken(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("gagal sign token: %v", err)
	}
	return signed
}

// nextOK adalah handler hilir yang hanya bisa dicapai bila auth lolos.
// Ia menulis kembali user_id dari context agar bisa diverifikasi.
func nextOK() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := UserIDFromContext(r.Context())
		if !ok {
			http.Error(w, "user_id hilang di context", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(userID))
	})
}

func TestNoAuthorizationHeader401(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()

	AuthMiddleware(testSecret)(nextOK()).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("ingin 401, dapat %d: %s", rec.Code, rec.Body.String())
	}
}

func TestInvalidAndExpiredToken401(t *testing.T) {
	cases := map[string]string{
		"format acak": "bukan-token-jwt",
		"signature beda": signToken(t, "secret-yang-salah", jwt.MapClaims{
			"user_id": "550e8400-e29b-41d4-a716-446655440000",
			"exp":     time.Now().Add(time.Hour).Unix(),
		}),
		"expired": signToken(t, testSecret, jwt.MapClaims{
			"user_id": "550e8400-e29b-41d4-a716-446655440000",
			"exp":     time.Now().Add(-time.Hour).Unix(),
		}),
		"tanpa klaim user": signToken(t, testSecret, jwt.MapClaims{
			"exp": time.Now().Add(time.Hour).Unix(),
		}),
	}

	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()

			AuthMiddleware(testSecret)(nextOK()).ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("ingin 401, dapat %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestValidToken200AndUserIDExtracted(t *testing.T) {
	const wantUserID = "550e8400-e29b-41d4-a716-446655440000"
	token := signToken(t, testSecret, jwt.MapClaims{
		"user_id": wantUserID,
		"exp":     time.Now().Add(time.Hour).Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	AuthMiddleware(testSecret)(nextOK()).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ingin 200, dapat %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != wantUserID {
		t.Fatalf("user_id di context = %q, ingin %q", rec.Body.String(), wantUserID)
	}
}
