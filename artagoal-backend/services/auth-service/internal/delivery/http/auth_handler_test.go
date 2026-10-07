package httphandler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"artagoal/auth-service/internal/delivery/http/middleware"
	"artagoal/auth-service/internal/domain/entity"
	"artagoal/auth-service/internal/repository"
	"artagoal/auth-service/internal/usecase"
)

const testSecret = "test-jwt-secret-minimal-32-karakter"

// memRepo adalah fake in-memory repository.UserRepository.
type memRepo struct {
	repository.UserRepository
	byID    map[string]entity.User
	byEmail map[string]entity.User
	seq     int
}

func newMemRepo() *memRepo {
	return &memRepo{byID: map[string]entity.User{}, byEmail: map[string]entity.User{}}
}

func (s *memRepo) CreateUser(_ context.Context, user entity.User) (entity.User, error) {
	if _, ok := s.byEmail[user.Email]; ok {
		return entity.User{}, entity.ErrEmailAlreadyExists
	}
	s.seq++
	user.ID = "user-test-1"
	s.byID[user.ID] = user
	s.byEmail[user.Email] = user
	return user, nil
}

func (s *memRepo) GetUserByEmail(_ context.Context, email string) (entity.User, error) {
	u, ok := s.byEmail[email]
	if !ok {
		return entity.User{}, entity.ErrUserNotFound
	}
	return u, nil
}

func (s *memRepo) GetUserByID(_ context.Context, id string) (entity.User, error) {
	u, ok := s.byID[id]
	if !ok {
		return entity.User{}, entity.ErrUserNotFound
	}
	return u, nil
}

func (s *memRepo) UpdatePassword(_ context.Context, id, hash string) error {
	u, ok := s.byID[id]
	if !ok {
		return entity.ErrUserNotFound
	}
	u.PasswordHash = hash
	s.byID[id] = u
	s.byEmail[u.Email] = u
	return nil
}

func newTestRouter() chi.Router {
	h := NewAuthHandler(usecase.NewAuthUsecase(newMemRepo(), testSecret))
	r := chi.NewRouter()
	h.RegisterAuthRoutes(r, middleware.AuthMiddleware(testSecret))
	return r
}

func doRequest(r chi.Router, method, path, body, token string) *httptest.ResponseRecorder {
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestRegisterThenLoginThenMe(t *testing.T) {
	r := newTestRouter()

	rec := doRequest(r, http.MethodPost, "/api/v1/auth/register", `{"name":"Budi","email":"budi@x.com","password":"rahasia123"}`, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register ingin 201, dapat %d: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "password_hash") {
		t.Fatal("respons tidak boleh membocorkan password_hash")
	}

	rec = doRequest(r, http.MethodPost, "/api/v1/auth/register", `{"name":"Budi","email":"budi@x.com","password":"rahasia123"}`, "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("email duplikat ingin 409, dapat %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(r, http.MethodPost, "/api/v1/auth/login", `{"email":"budi@x.com","password":"rahasia123"}`, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login ingin 200, dapat %d: %s", rec.Code, rec.Body.String())
	}
	var loginResp entity.AuthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("respons login bukan JSON valid: %v", err)
	}
	if loginResp.Token == "" || loginResp.User.Email != "budi@x.com" {
		t.Fatalf("respons login tidak lengkap: %+v", loginResp)
	}

	rec = doRequest(r, http.MethodGet, "/api/v1/auth/me", "", loginResp.Token)
	if rec.Code != http.StatusOK {
		t.Fatalf("me dengan token valid ingin 200, dapat %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "budi@x.com") {
		t.Fatalf("me harus memuat profil: %s", rec.Body.String())
	}

	rec = doRequest(r, http.MethodGet, "/api/v1/auth/me", "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("me tanpa token ingin 401, dapat %d", rec.Code)
	}

	rec = doRequest(r, http.MethodGet, "/api/v1/auth/me", "", "token.palsu.disini")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("me dengan token palsu ingin 401, dapat %d", rec.Code)
	}
}

func TestLoginFailuresAndRegisterValidation(t *testing.T) {
	r := newTestRouter()

	rec := doRequest(r, http.MethodPost, "/api/v1/auth/login", `{"email":"tidak@ada.com","password":"rahasia123"}`, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("email tak dikenal ingin 401, dapat %d: %s", rec.Code, rec.Body.String())
	}

	_ = doRequest(r, http.MethodPost, "/api/v1/auth/register", `{"name":"A","email":"a@x.com","password":"rahasia123"}`, "")
	rec = doRequest(r, http.MethodPost, "/api/v1/auth/login", `{"email":"a@x.com","password":"salah1234"}`, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("password salah ingin 401, dapat %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(r, http.MethodPost, "/api/v1/auth/register", `{"name":"A","email":"bukan-email","password":"rahasia123"}`, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("email invalid ingin 400, dapat %d: %s", rec.Code, rec.Body.String())
	}
	rec = doRequest(r, http.MethodPost, "/api/v1/auth/register", `{"name":"A","email":"b@x.com","password":"pendek"}`, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("password pendek ingin 400, dapat %d: %s", rec.Code, rec.Body.String())
	}
}

func TestChangePasswordEndpoint(t *testing.T) {
	r := newTestRouter()

	rec := doRequest(r, http.MethodPost, "/api/v1/auth/register", `{"name":"C","email":"c@x.com","password":"lama12345"}`, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register ingin 201, dapat %d: %s", rec.Code, rec.Body.String())
	}
	var reg struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &reg); err != nil {
		t.Fatalf("respons register bukan JSON: %v", err)
	}

	rec = doRequest(r, http.MethodPut, "/api/v1/auth/password", `{"old_password":"lama12345","new_password":"baru12345"}`, reg.Token)
	if rec.Code != http.StatusOK {
		t.Fatalf("ganti password ingin 200, dapat %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(r, http.MethodPost, "/api/v1/auth/login", `{"email":"c@x.com","password":"baru12345"}`, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login password baru ingin 200, dapat %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(r, http.MethodPut, "/api/v1/auth/password", `{"old_password":"salah1234","new_password":"lain12345"}`, reg.Token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("password lama salah ingin 400, dapat %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(r, http.MethodPut, "/api/v1/auth/password", `{"old_password":"baru12345","new_password":"baru12345"}`, reg.Token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("password sama ingin 400, dapat %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(r, http.MethodPut, "/api/v1/auth/password", `{"old_password":"baru12345","new_password":"lain12345"}`, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("tanpa token ingin 401, dapat %d: %s", rec.Code, rec.Body.String())
	}
}
