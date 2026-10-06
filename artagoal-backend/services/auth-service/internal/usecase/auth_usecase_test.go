package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"artagoal/auth-service/internal/domain/entity"
	"artagoal/auth-service/internal/repository"
)

// stubRepo adalah fake in-memory untuk verifikasi logika usecase
// tanpa membutuhkan Postgres.
type stubRepo struct {
	repository.UserRepository
	byID    map[string]entity.User
	byEmail map[string]entity.User
}

func newStub() *stubRepo {
	return &stubRepo{byID: map[string]entity.User{}, byEmail: map[string]entity.User{}}
}

func (s *stubRepo) CreateUser(_ context.Context, user entity.User) (entity.User, error) {
	if _, ok := s.byEmail[user.Email]; ok {
		return entity.User{}, entity.ErrEmailAlreadyExists
	}
	user.ID = "user-123"
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	s.byID[user.ID] = user
	s.byEmail[user.Email] = user
	return user, nil
}

func (s *stubRepo) GetUserByEmail(_ context.Context, email string) (entity.User, error) {
	u, ok := s.byEmail[email]
	if !ok {
		return entity.User{}, entity.ErrUserNotFound
	}
	return u, nil
}

func (s *stubRepo) GetUserByID(_ context.Context, id string) (entity.User, error) {
	u, ok := s.byID[id]
	if !ok {
		return entity.User{}, entity.ErrUserNotFound
	}
	return u, nil
}

const testSecret = "test-jwt-secret-minimal-32-karakter"

func parseClaims(t *testing.T, tokenStr string) jwt.MapClaims {
	t.Helper()
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		return []byte(testSecret), nil
	})
	if err != nil || !token.Valid {
		t.Fatalf("token harus valid: %v", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("klaim harus MapClaims")
	}
	return claims
}

func TestRegisterSuccessIssues24hToken(t *testing.T) {
	uc := NewAuthUsecase(newStub(), testSecret)
	resp, err := uc.Register(context.Background(), entity.RegisterRequest{
		Name: "Budi", Email: "Budi@Example.com", Password: "rahasia123",
	})
	if err != nil {
		t.Fatalf("register gagal: %v", err)
	}
	if resp.Token == "" {
		t.Fatal("token harus terisi")
	}
	if resp.User.Email != "budi@example.com" {
		t.Fatalf("email harus dinormalisasi lowercase, dapat %q", resp.User.Email)
	}
	claims := parseClaims(t, resp.Token)
	if claims["user_id"] != "user-123" {
		t.Fatalf("klaim user_id salah: %v", claims["user_id"])
	}
	if claims["email"] != "budi@example.com" {
		t.Fatalf("klaim email salah: %v", claims["email"])
	}
	exp, _ := claims["exp"].(float64)
	iat, _ := claims["iat"].(float64)
	if exp-iat != float64(24*3600) {
		t.Fatalf("umur token harus 24 jam, dapat %v detik", exp-iat)
	}
}

func TestRegisterValidation(t *testing.T) {
	uc := NewAuthUsecase(newStub(), testSecret)
	ctx := context.Background()
	cases := map[string]entity.RegisterRequest{
		"nama kosong":     {Name: "", Email: "a@b.com", Password: "rahasia123"},
		"email invalid":   {Name: "A", Email: "bukan-email", Password: "rahasia123"},
		"password pendek": {Name: "A", Email: "a@b.com", Password: "pendek"},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := uc.Register(ctx, req); !errors.Is(err, ErrValidation) {
				t.Fatalf("ingin ErrValidation, dapat %v", err)
			}
		})
	}
}

func TestRegisterDuplicateEmail409(t *testing.T) {
	st := newStub()
	uc := NewAuthUsecase(st, testSecret)
	ctx := context.Background()
	req := entity.RegisterRequest{Name: "Budi", Email: "budi@x.com", Password: "rahasia123"}
	if _, err := uc.Register(ctx, req); err != nil {
		t.Fatalf("register pertama gagal: %v", err)
	}
	_, err := uc.Register(ctx, req)
	if !errors.Is(err, entity.ErrEmailAlreadyExists) {
		t.Fatalf("ingin ErrEmailAlreadyExists, dapat %v", err)
	}
}

func TestLoginSuccessAndFailures(t *testing.T) {
	st := newStub()
	uc := NewAuthUsecase(st, testSecret)
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("rahasia123"), bcrypt.MinCost)
	st.byID["u1"] = entity.User{ID: "u1", Name: "Budi", Email: "budi@x.com", PasswordHash: string(hash)}
	st.byEmail["budi@x.com"] = st.byID["u1"]

	resp, err := uc.Login(ctx, entity.LoginRequest{Email: "BUDI@x.com", Password: "rahasia123"})
	if err != nil {
		t.Fatalf("login gagal: %v", err)
	}
	if resp.Token == "" || resp.User.ID != "u1" {
		t.Fatalf("respons login tidak lengkap: %+v", resp)
	}

	if _, err := uc.Login(ctx, entity.LoginRequest{Email: "budi@x.com", Password: "salah1234"}); !errors.Is(err, entity.ErrInvalidCredentials) {
		t.Fatalf("password salah harus ErrInvalidCredentials, dapat %v", err)
	}
	if _, err := uc.Login(ctx, entity.LoginRequest{Email: "tidak@ada.com", Password: "rahasia123"}); !errors.Is(err, entity.ErrInvalidCredentials) {
		t.Fatalf("email tak dikenal harus ErrInvalidCredentials, dapat %v", err)
	}
}

func TestMeNotFound(t *testing.T) {
	uc := NewAuthUsecase(newStub(), testSecret)
	if _, err := uc.Me(context.Background(), "tidak-ada"); !errors.Is(err, entity.ErrUserNotFound) {
		t.Fatalf("ingin ErrUserNotFound, dapat %v", err)
	}
}

func TestGenerateTokenNeedsSecret(t *testing.T) {
	uc := NewAuthUsecase(newStub(), "")
	if _, err := uc.GenerateToken(entity.User{ID: "u1"}); err == nil {
		t.Fatal("secret kosong harus gagal (fail-closed)")
	}
}

