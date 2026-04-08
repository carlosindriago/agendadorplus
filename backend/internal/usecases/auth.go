package usecases

import (
	"context"
	"log/slog"

	"github.com/carlosindriago/agendadorplus/internal/domain"
	"github.com/carlosindriago/agendadorplus/internal/ports"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"time"
)

// AuthUseCase handles authentication operations.
// It implements ports.AuthService.
type AuthUseCase struct {
	userRepo  ports.UserRepository
	jwtSecret []byte
	logger    *slog.Logger
}

// Compile-time check that AuthUseCase implements ports.AuthService.
var _ ports.AuthService = (*AuthUseCase)(nil)

// NewAuthUseCase creates a new AuthUseCase.
func NewAuthUseCase(userRepo ports.UserRepository, jwtSecret string, logger *slog.Logger) *AuthUseCase {
	return &AuthUseCase{
		userRepo:  userRepo,
		jwtSecret: []byte(jwtSecret),
		logger:    logger,
	}
}

// Login validates credentials and returns a JWT token.
func (uc *AuthUseCase) Login(ctx context.Context, email, password string) (string, *domain.User, error) {
	if email == "" || password == "" {
		return "", nil, domain.NewValidationError("email and password are required")
	}

	user, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil {
		// Don't leak whether the email exists
		uc.logger.WarnContext(ctx, "login attempt with unknown email", "email", email)
		return "", nil, domain.ErrInvalidCredentials
	}

	// Compare password with bcrypt hash
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		uc.logger.WarnContext(ctx, "login attempt with wrong password", "email", email)
		return "", nil, domain.ErrInvalidCredentials
	}

	// Generate JWT
	token, err := generateJWT(user.ID, user.TenantID, uc.jwtSecret)
	if err != nil {
		uc.logger.ErrorContext(ctx, "failed to generate JWT", "error", err)
		return "", nil, err
	}

	uc.logger.InfoContext(ctx, "user logged in successfully", "user_id", user.ID, "email", email)

	return token, user, nil
}

// generateJWT creates a signed JWT token for the given user.
func generateJWT(userID, tenantID uuid.UUID, secret []byte) (string, error) {
	claims := jwt.MapClaims{
		"sub":       userID.String(),
		"tenant_id": tenantID.String(),
		"iat":       time.Now().Unix(),
		"exp":       time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}
