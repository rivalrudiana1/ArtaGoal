package httphandler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"artagoal/auth-service/internal/delivery/http/middleware"
	"artagoal/auth-service/internal/domain/entity"
	"artagoal/auth-service/internal/usecase"
)

// AuthHandler melayani endpoint REST autentikasi.
// Bergantung pada interface usecase (bukan implementasi konkret).
type AuthHandler struct {
	uc usecase.AuthUsecase
}

// NewAuthHandler membuat handler baru.
func NewAuthHandler(uc usecase.AuthUsecase) *AuthHandler {
	return &AuthHandler{uc: uc}
}

// RegisterAuthRoutes mendaftarkan endpoint auth pada router chi yang diberikan.
// Endpoint register & login bersifat publik; /me dibungkus middleware JWT
// (user_id diambil dari klaim token, bukan dari path/body).
func (h *AuthHandler) RegisterAuthRoutes(r chi.Router, auth func(http.Handler) http.Handler) {
	r.Post("/api/v1/auth/register", h.Register)
	r.Post("/api/v1/auth/login", h.Login)
	r.With(auth).Get("/api/v1/auth/me", h.Me)
}

// ---- helpers ----

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// statusForError memetakan error domain/usecase ke HTTP status.
func statusForError(err error) int {
	switch {
	case errors.Is(err, entity.ErrUserNotFound):
		return http.StatusNotFound
	case errors.Is(err, entity.ErrInvalidCredentials):
		return http.StatusUnauthorized
	case errors.Is(err, entity.ErrEmailAlreadyExists):
		return http.StatusConflict
	case errors.Is(err, usecase.ErrValidation):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// decodeJSON membaca body (maks 1 MiB) menjadi v.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "body JSON tidak valid: "+err.Error())
		return false
	}
	return true
}

// ---- handlers ----

// Register menangani POST /api/v1/auth/register -> 201 Created + token.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req entity.RegisterRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	resp, err := h.uc.Register(r.Context(), req)
	if err != nil {
		status := statusForError(err)
		if status == http.StatusInternalServerError {
			writeError(w, status, "gagal mendaftarkan user")
			return
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// Login menangani POST /api/v1/auth/login -> 200 OK + token.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req entity.LoginRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	resp, err := h.uc.Login(r.Context(), req)
	if err != nil {
		status := statusForError(err)
		if status == http.StatusInternalServerError {
			writeError(w, status, "gagal login")
			return
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// Me menangani GET /api/v1/auth/me -> 200 OK + profil user.
// Wajib melewati AuthMiddleware; user_id diambil dari request context.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized: token tidak ada atau tidak valid")
		return
	}

	user, err := h.uc.Me(r.Context(), userID)
	if err != nil {
		status := statusForError(err)
		if status == http.StatusInternalServerError {
			writeError(w, status, "gagal mengambil profil")
			return
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, user.ToInfo())
}

