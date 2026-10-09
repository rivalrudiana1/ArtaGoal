package httphandler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

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
// Setiap endpoint tersedia dengan dua prefix: "/auth/..." dan "/api/v1/auth/...".
func (h *AuthHandler) RegisterAuthRoutes(r chi.Router, auth func(http.Handler) http.Handler) {
	for _, prefix := range []string{"", "/api/v1"} {
		r.Post(prefix+"/auth/register", h.Register)
		r.Post(prefix+"/auth/login", h.Login)
		r.With(auth).Get(prefix+"/auth/me", h.Me)
		r.With(auth).Put(prefix+"/auth/profile", h.HandleUpdateProfile)
		r.With(auth).Put(prefix+"/auth/password", h.HandleChangePassword)
	}
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

// Batas ukuran file avatar: 2 MiB.
const maxAvatarSize = 2 << 20

// allowedAvatarExt adalah ekstensi gambar yang diizinkan untuk avatar.
var allowedAvatarExt = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}

// uniqueAvatarName membuat nama file unik dari timestamp + acak,
// mempertahankan ekstensi asli yang sudah tervalidasi.
func uniqueAvatarName(ext string) string {
	var rnd [3]byte
	if _, err := rand.Read(rnd[:]); err != nil {
		return fmt.Sprintf("avatar-%d%s", time.Now().UnixNano(), ext)
	}
	return fmt.Sprintf("avatar-%d-%s%s", time.Now().UnixNano(), hex.EncodeToString(rnd[:]), ext)
}

// HandleUpdateProfile menangani PUT /api/v1/auth/profile -> 200 OK + profil terbaru.
// Menerima multipart/form-data dengan field teks "name" dan file opsional "avatar".
// File divalidasi maksimal 2MB dengan ekstensi jpg/jpeg/png/webp, disimpan ke
// uploads/avatars dengan nama unik, lalu path statisnya disimpan ke database.
func (h *AuthHandler) HandleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized: token tidak ada atau tidak valid")
		return
	}

	// Batasi total body agar file raksasa ditolak sejak awal.
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	if err := r.ParseMultipartForm(maxAvatarSize); err != nil {
		writeError(w, http.StatusBadRequest, "body multipart tidak valid: "+err.Error())
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))

	avatarURL := ""
	file, header, err := r.FormFile("avatar")
	switch {
	case err == nil:
		defer file.Close()
		saved, ferr := saveAvatarFile(file, header.Filename, header.Size)
		if ferr != nil {
			writeError(w, http.StatusBadRequest, ferr.Error())
			return
		}
		avatarURL = saved
	case errors.Is(err, http.ErrMissingFile):
		// Tanpa file baru: pertahankan avatar yang sudah ada.
		current, cerr := h.uc.Me(r.Context(), userID)
		if cerr != nil {
			status := statusForError(cerr)
			if status == http.StatusInternalServerError {
				writeError(w, status, "gagal mengambil profil")
				return
			}
			writeError(w, status, cerr.Error())
			return
		}
		avatarURL = current.AvatarURL
	default:
		writeError(w, http.StatusBadRequest, "field avatar tidak valid: "+err.Error())
		return
	}

	updated, err := h.uc.UpdateProfile(r.Context(), userID, name, avatarURL)
	if err != nil {
		status := statusForError(err)
		if status == http.StatusInternalServerError {
			writeError(w, status, "gagal memperbarui profil")
			return
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, updated.ToInfo())
}

// saveAvatarFile memvalidasi ukuran (maks 2MB), ekstensi (jpg/jpeg/png/webp),
// dan isi gambar (jpeg/png/webp), lalu menyimpannya ke uploads/avatars dengan
// nama unik. Mengembalikan path statis untuk disimpan ke database.
func saveAvatarFile(src multipart.File, filename string, size int64) (string, error) {
	if size > maxAvatarSize {
		return "", errors.New("ukuran avatar maksimal 2MB")
	}
	ext := strings.ToLower(filepath.Ext(filepath.Base(filename)))
	if !allowedAvatarExt[ext] {
		return "", errors.New("ekstensi avatar hanya boleh jpg/jpeg/png/webp")
	}

	// Sniff 512 byte pertama agar isi benar-benar gambar jpeg/png/webp.
	head := make([]byte, 512)
	n, _ := io.ReadFull(src, head)
	switch http.DetectContentType(head[:n]) {
	case "image/jpeg", "image/png", "image/webp":
	default:
		return "", errors.New("isi file avatar harus gambar jpeg/png/webp")
	}

	if err := os.MkdirAll("uploads/avatars", 0o755); err != nil {
		return "", errors.New("gagal menyiapkan direktori avatar")
	}

	name := uniqueAvatarName(ext)
	dst, err := os.Create(filepath.Join("uploads", "avatars", name))
	if err != nil {
		return "", errors.New("gagal menyimpan file avatar")
	}
	defer dst.Close()

	if _, err := dst.Write(head[:n]); err != nil {
		_ = os.Remove(dst.Name())
		return "", errors.New("gagal menyimpan file avatar")
	}
	written, err := io.Copy(dst, src)
	if err != nil {
		_ = os.Remove(dst.Name())
		return "", errors.New("gagal menyimpan file avatar")
	}
	if int64(n)+written > maxAvatarSize {
		_ = os.Remove(dst.Name())
		return "", errors.New("ukuran avatar maksimal 2MB")
	}

	return "/uploads/avatars/" + name, nil
}

// ChangePasswordRequest adalah payload PUT /api/v1/auth/password.
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// HandleChangePassword menangani PUT /api/v1/auth/password -> 200 OK.
// Memverifikasi password lama sebelum menyimpan hash password baru.
func (h *AuthHandler) HandleChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized: token tidak ada atau tidak valid")
		return
	}

	var req ChangePasswordRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if err := h.uc.ChangePassword(r.Context(), userID, req.OldPassword, req.NewPassword); err != nil {
		status := statusForError(err)
		if status == http.StatusInternalServerError {
			writeError(w, status, "gagal mengganti password")
			return
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "password berhasil diganti"})
}
