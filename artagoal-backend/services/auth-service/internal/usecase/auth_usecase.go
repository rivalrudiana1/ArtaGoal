package usecase

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"artagoal/auth-service/internal/domain/entity"
	"artagoal/auth-service/internal/repository"
)

// ErrValidation menandai input tidak valid.
// Delivery layer memetakannya ke HTTP 400 Bad Request via errors.Is.
var ErrValidation = errors.New("validation error")

// TokenTTL adalah umur token JWT yang diterbitkan saat register/login.
const TokenTTL = 24 * time.Hour

// AuthUsecase adalah kontrak bisnis autentikasi.
type AuthUsecase interface {
	Register(ctx context.Context, req entity.RegisterRequest) (entity.AuthResponse, error)
	Login(ctx context.Context, req entity.LoginRequest) (entity.AuthResponse, error)
	// Me mengambil profil user terautentikasi untuk endpoint GET /api/v1/auth/me.
	Me(ctx context.Context, userID string) (entity.User, error)
	// GenerateToken membuat JWT 24 jam dengan klaim user_id, email, exp.
	GenerateToken(user entity.User) (string, error)
}

// authUsecase implementasi AuthUsecase.
type authUsecase struct {
	repo      repository.UserRepository
	jwtSecret string
	now       func() time.Time
}

// NewAuthUsecase membuat usecase baru dengan HMAC secret untuk signing JWT.
func NewAuthUsecase(repo repository.UserRepository, jwtSecret string) AuthUsecase {
	return &authUsecase{repo: repo, jwtSecret: jwtSecret, now: time.Now}
}

// validationError membungkus pesan validasi agar terdeteksi errors.Is(err, ErrValidation).
func validationError(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrValidation, fmt.Sprintf(format, args...))
}

// normalizeEmail memangkas spasi dan menurunkan huruf agar unik per email.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// validEmail memastikan format email benar (bukan sekadar mengandung "@").
func validEmail(email string) bool {
	if email == "" || len(email) > 255 || strings.Contains(email, " ") {
		return false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	return addr.Address == email
}

// validatePassword memastikan panjang 8-72 karakter.
// Batas atas 72 byte mengikuti limit bcrypt agar tidak terpotong diam-diam.
func validatePassword(password string) error {
	n := utf8.RuneCountInString(password)
	if n < 8 {
		return validationError("password minimal 8 karakter")
	}
	if len(password) > 72 {
		return validationError("password maksimal 72 karakter")
	}
	return nil
}

// Register memvalidasi input, menolak email duplikat, menyimpan hash bcrypt,
// lalu menerbitkan token JWT 24 jam.
func (u *authUsecase) Register(ctx context.Context, req entity.RegisterRequest) (entity.AuthResponse, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Email = normalizeEmail(req.Email)

	if req.Name == "" || utf8.RuneCountInString(req.Name) > 100 {
		return entity.AuthResponse{}, validationError("name wajib diisi (maks 100 karakter)")
	}
	if !validEmail(req.Email) {
		return entity.AuthResponse{}, validationError("format email tidak valid")
	}
	if err := validatePassword(req.Password); err != nil {
		return entity.AuthResponse{}, err
	}

	if _, err := u.repo.GetUserByEmail(ctx, req.Email); err == nil {
		return entity.AuthResponse{}, fmt.Errorf("usecase: Register: %w", entity.ErrEmailAlreadyExists)
	} else if !errors.Is(err, entity.ErrUserNotFound) {
		return entity.AuthResponse{}, fmt.Errorf("usecase: Register check email: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return entity.AuthResponse{}, fmt.Errorf("usecase: Register hash password: %w", err)
	}

	created, err := u.repo.CreateUser(ctx, entity.User{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hash),
	})
	if err != nil {
		return entity.AuthResponse{}, fmt.Errorf("usecase: Register: %w", err)
	}

	token, err := u.GenerateToken(created)
	if err != nil {
		return entity.AuthResponse{}, err
	}
	return entity.AuthResponse{Token: token, User: created.ToInfo()}, nil
}

// Login memverifikasi kredensial lalu menerbitkan token JWT 24 jam.
// Email tak dikenal dan password salah sama-sama dipetakan ke
// ErrInvalidCredentials agar tidak membocorkan keberadaan akun.
func (u *authUsecase) Login(ctx context.Context, req entity.LoginRequest) (entity.AuthResponse, error) {
	req.Email = normalizeEmail(req.Email)
	if req.Email == "" || req.Password == "" {
		return entity.AuthResponse{}, validationError("email dan password wajib diisi")
	}

	user, err := u.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) {
			return entity.AuthResponse{}, fmt.Errorf("usecase: Login: %w", entity.ErrInvalidCredentials)
		}
		return entity.AuthResponse{}, fmt.Errorf("usecase: Login load user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return entity.AuthResponse{}, fmt.Errorf("usecase: Login: %w", entity.ErrInvalidCredentials)
	}

	token, err := u.GenerateToken(user)
	if err != nil {
		return entity.AuthResponse{}, err
	}
	return entity.AuthResponse{Token: token, User: user.ToInfo()}, nil
}

// Me mengambil profil satu user berdasarkan id (untuk /auth/me).
func (u *authUsecase) Me(ctx context.Context, userID string) (entity.User, error) {
	if userID == "" {
		return entity.User{}, validationError("user_id wajib diisi")
	}
	user, err := u.repo.GetUserByID(ctx, userID)
	if err != nil {
		return entity.User{}, fmt.Errorf("usecase: Me: %w", err)
	}
	return user, nil
}

// GenerateToken membuat JWT HS256 berumur 24 jam dengan klaim
// user_id (UUID), email, iat, dan exp.
func (u *authUsecase) GenerateToken(user entity.User) (string, error) {
	if u.jwtSecret == "" {
		return "", errors.New("usecase: GenerateToken: JWT secret belum dikonfigurasi")
	}
	if user.ID == "" {
		return "", validationError("user id wajib diisi untuk token")
	}
	now := u.now()
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"iat":     now.Unix(),
		"exp":     now.Add(TokenTTL).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(u.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("usecase: GenerateToken sign: %w", err)
	}
	return signed, nil
}

