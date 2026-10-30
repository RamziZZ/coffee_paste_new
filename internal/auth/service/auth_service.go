package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"coffee_paste_backend/config"
	"coffee_paste_backend/internal/auth/dto"
	"coffee_paste_backend/internal/auth/model"
	"coffee_paste_backend/internal/auth/passwordreset/repository"
	authrepository "coffee_paste_backend/internal/auth/repository"
	"coffee_paste_backend/pkg/jwt"
	"coffee_paste_backend/pkg/password"

	"google.golang.org/api/idtoken"
)

var (
	ErrInvalidCredentials = errors.New("email atau password salah")
	ErrUserInactive       = errors.New("akun tidak aktif")
	ErrPasswordResetToken = errors.New("token reset password tidak valid atau sudah kedaluwarsa")
	ErrUserNotFound       = errors.New("user tidak ditemukan")
	ErrEmailAlreadyExists = errors.New("email sudah digunakan")
	ErrInvalidGoogleToken = errors.New("token Google tidak valid")
	ErrGoogleNotConfigured = errors.New("login Google belum dikonfigurasi")
)

type AuthService struct {
	repository              *authrepository.AuthRepository
	passwordResetRepository *repository.PasswordResetRepository
	config                  config.Config
}

func NewAuthService(
	repository *authrepository.AuthRepository,
	passwordResetRepository *repository.PasswordResetRepository,
	config config.Config,
) *AuthService {
	return &AuthService{
		repository:              repository,
		passwordResetRepository: passwordResetRepository,
		config:                  config,
	}
}

func (s *AuthService) Login(
	ctx context.Context,
	req dto.LoginRequest,
) (*dto.LoginResponse, error) {

	email := strings.TrimSpace(
		strings.ToLower(req.Email),
	)

	user, err := s.repository.FindByEmail(
		ctx,
		email,
	)

	if err != nil {
		if errors.Is(err, authrepository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}

		return nil, err
	}

	if !user.IsActive {
		return nil, ErrUserInactive
	}

	if !password.Verify(
		req.Password,
		user.PasswordHash,
	) {
		return nil, ErrInvalidCredentials
	}

	const tokenDuration = 24 * time.Hour

	accessToken, err := jwt.GenerateToken(
		user.ID,
		user.Role,
		s.config.JWTSecret,
		tokenDuration,
	)

	if err != nil {
		return nil, err
	}

	return &dto.LoginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(tokenDuration.Seconds()),
		User:        mapUserResponse(user),
	}, nil
}

func mapUserResponse(
	user *model.User,
) dto.UserResponse {
	return dto.UserResponse{
		ID:       user.ID,
		Role:     user.Role,
		Name:     user.Name,
		Email:    user.Email,
		Phone:    user.Phone,
		IsActive: user.IsActive,
	}
}

func (s *AuthService) Me(
	ctx context.Context,
	userID string,
) (*dto.UserResponse, error) {

	user, err := s.repository.FindByID(
		ctx,
		userID,
	)

	if err != nil {
		if errors.Is(err, authrepository.ErrUserNotFound) {
			return nil, authrepository.ErrUserNotFound
		}

		return nil, err
	}

	if !user.IsActive {
		return nil, ErrUserInactive
	}

	response := mapUserResponse(user)

	return &response, nil
}

func (s *AuthService) ForgotPassword(
	ctx context.Context,
	req dto.ForgotPasswordRequest,
) (string, error) {

	email := strings.TrimSpace(
		strings.ToLower(req.Email),
	)

	user, err := s.repository.FindByEmail(
		ctx,
		email,
	)

	if err != nil {
		if errors.Is(err, authrepository.ErrUserNotFound) {
			return "", authrepository.ErrUserNotFound
		}

		return "", err
	}

	if !user.IsActive {
		return "", ErrUserInactive
	}

	// Membuat token random 32 byte
	tokenBytes := make([]byte, 32)

	_, err = rand.Read(tokenBytes)
	if err != nil {
		return "", err
	}

	token := hex.EncodeToString(tokenBytes)

	// Hash token sebelum disimpan ke database
	tokenHashBytes := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(tokenHashBytes[:])

	// Token berlaku selama 15 menit
	expiresAt := time.Now().Add(15 * time.Minute)

	err = s.passwordResetRepository.Create(
		ctx,
		user.ID,
		tokenHash,
		expiresAt,
	)

	if err != nil {
		return "", err
	}

	// Untuk sementara token dikembalikan agar bisa
	// kita test melalui Postman.
	// Nanti diganti dengan email reset password.
	return token, nil
}

func (s *AuthService) ResetPassword(
	ctx context.Context,
	req dto.ResetPasswordRequest,
) error {

	// Hash token dari request
	tokenHashBytes := sha256.Sum256([]byte(req.Token))
	tokenHash := hex.EncodeToString(tokenHashBytes[:])

	// Cari token yang masih valid
	resetData, err := s.passwordResetRepository.FindValidByTokenHash(
		ctx,
		tokenHash,
	)

	if err != nil {
		if errors.Is(
			err,
			repository.ErrPasswordResetNotFound,
		) {
			return ErrPasswordResetToken
		}

		return err
	}

	// Hash password baru
	passwordHash, err := password.Hash(req.NewPassword)
	if err != nil {
		return err
	}

	// Update password user
	err = s.repository.UpdatePassword(
		ctx,
		resetData.UserID,
		passwordHash,
	)

	if err != nil {
		return err
	}

	// Tandai token sudah digunakan
	err = s.passwordResetRepository.MarkAsUsed(
		ctx,
		resetData.ID,
	)

	if err != nil {
		return err
	}

	return nil
}

func (s *AuthService) RegisterCustomer(
	ctx context.Context,
	req dto.RegisterCustomerRequest,
) (*dto.UserResponse, error) {

	// Bersihkan input
	name := strings.TrimSpace(req.Name)
	email := strings.TrimSpace(strings.ToLower(req.Email))
	phone := strings.TrimSpace(req.Phone)

	// Cek apakah email sudah digunakan
	_, err := s.repository.FindByEmail(ctx, email)

	if err == nil {
		return nil, ErrEmailAlreadyExists
	}

	if !errors.Is(err, authrepository.ErrUserNotFound) {
		return nil, err
	}

	// Hash password
	passwordHash, err := password.Hash(req.Password)
	if err != nil {
		return nil, err
	}

	// Buat customer
	user, err := s.repository.CreateCustomer(
		ctx,
		name,
		email,
		passwordHash,
		phone,
	)
	if err != nil {
		return nil, err
	}

	response := mapUserResponse(user)

	return &response, nil
}

// LoginWithGoogle memverifikasi ID token dari Google Sign-In,
// lalu mencocokkan/membuat user dan menerbitkan JWT.
// Jika email belum terdaftar, user customer baru dibuat otomatis.
func (s *AuthService) LoginWithGoogle(
	ctx context.Context,
	req dto.GoogleLoginRequest,
) (*dto.LoginResponse, error) {

	if s.config.GoogleClientID == "" {
		return nil, ErrGoogleNotConfigured
	}

	// Verifikasi ID token terhadap client ID kita
	payload, err := idtoken.Validate(
		ctx,
		req.IDToken,
		s.config.GoogleClientID,
	)
	if err != nil {
		return nil, ErrInvalidGoogleToken
	}

	// Ambil klaim standar Google
	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)
	picture, _ := payload.Claims["picture"].(string)
	emailVerified, _ := payload.Claims["email_verified"].(bool)

	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || !emailVerified {
		return nil, ErrInvalidGoogleToken
	}

	if name == "" {
		name = strings.Split(email, "@")[0]
	}

	// Cari user berdasarkan email dulu
	user, err := s.repository.FindByEmail(ctx, email)

	if err != nil && !errors.Is(err, authrepository.ErrUserNotFound) {
		return nil, err
	}

	if errors.Is(err, authrepository.ErrUserNotFound) {
		// User belum ada: buat customer baru dari data Google
		user, err = s.repository.CreateGoogleCustomer(
			ctx,
			name,
			email,
			payload.Subject,
			picture,
		)
		if err != nil {
			return nil, err
		}
	} else {
		// User sudah ada: cek aktif & link Google ID jika belum
		if !user.IsActive {
			return nil, ErrUserInactive
		}

		if user.GoogleID == nil || *user.GoogleID == "" {
			if err := s.repository.LinkGoogleID(
				ctx,
				user.ID,
				payload.Subject,
			); err != nil {
				return nil, err
			}
			user.GoogleID = &payload.Subject
		}
	}

	const tokenDuration = 24 * time.Hour

	accessToken, err := jwt.GenerateToken(
		user.ID,
		user.Role,
		s.config.JWTSecret,
		tokenDuration,
	)
	if err != nil {
		return nil, err
	}

	return &dto.LoginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(tokenDuration.Seconds()),
		User:        mapUserResponse(user),
	}, nil
}
