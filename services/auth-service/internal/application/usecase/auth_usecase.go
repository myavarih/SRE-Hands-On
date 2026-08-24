package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/myavarih/sre-hands-on/pkg/entities"
	"github.com/myavarih/sre-hands-on/pkg/exceptions"
	"github.com/myavarih/sre-hands-on/pkg/ports"
	"github.com/myavarih/sre-hands-on/services/auth-service/internal/application/dto"
	"golang.org/x/crypto/bcrypt"
)

type AuthUseCase interface {
	Register(ctx context.Context, input dto.RegisterInput) (*dto.UserOutput, error)
	Login(ctx context.Context, input dto.LoginInput) (*dto.AuthTokenOutput, error)
	ValidateToken(ctx context.Context, input dto.ValidateTokenInput) (*dto.ValidateTokenOutput, error)
	GetUserByID(ctx context.Context, userID string) (*dto.UserOutput, error)
}

type authUseCase struct {
	uow       ports.UnitOfWork
	jwtSecret []byte
	logger    *slog.Logger
}

func NewAuthUseCase(uow ports.UnitOfWork, jwtSecret string, logger *slog.Logger) AuthUseCase {
	return &authUseCase{
		uow:       uow,
		jwtSecret: []byte(jwtSecret),
		logger:    logger,
	}
}

type tokenClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func (uc *authUseCase) Register(ctx context.Context, input dto.RegisterInput) (*dto.UserOutput, error) {
	if input.Username == "" || input.Email == "" || input.Password == "" {
		return nil, exceptions.NewValidationError("username, email, and password are required")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		uc.logger.Error("Failed to hash password", "error", err)
		return nil, exceptions.NewInternalError("failed to process password", err)
	}

	user := &entities.User{
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: string(hashedPassword),
		Role:         "customer",
	}

	if err := uc.uow.Users().Create(ctx, user); err != nil {
		uc.logger.Warn("Failed to create user in repository", "error", err, "username", input.Username)
		return nil, err
	}

	uc.logger.Info("User registered successfully", "user_id", user.ID, "username", user.Username)

	return &dto.UserOutput{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}, nil
}

func (uc *authUseCase) Login(ctx context.Context, input dto.LoginInput) (*dto.AuthTokenOutput, error) {
	if input.Username == "" || input.Password == "" {
		return nil, exceptions.NewValidationError("username and password are required")
	}

	user, err := uc.uow.Users().FindByUsername(ctx, input.Username)
	if err != nil {
		user, err = uc.uow.Users().FindByEmail(ctx, input.Username)
		if err != nil {
			return nil, exceptions.NewUnauthorizedError("invalid credentials")
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		uc.logger.Warn("Invalid password attempt", "username", input.Username)
		return nil, exceptions.NewUnauthorizedError("invalid credentials")
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &tokenClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "sre-auth-service",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(uc.jwtSecret)
	if err != nil {
		uc.logger.Error("Failed to sign JWT token", "error", err)
		return nil, exceptions.NewInternalError("failed to generate token", err)
	}

	uc.logger.Info("User logged in successfully", "user_id", user.ID, "username", user.Username)

	return &dto.AuthTokenOutput{
		Token:     tokenString,
		ExpiresIn: int64(24 * 3600),
		UserID:    user.ID,
		Username:  user.Username,
	}, nil
}

func (uc *authUseCase) ValidateToken(ctx context.Context, input dto.ValidateTokenInput) (*dto.ValidateTokenOutput, error) {
	if input.Token == "" {
		return nil, exceptions.NewValidationError("token is required")
	}

	claims := &tokenClaims{}
	token, err := jwt.ParseWithClaims(input.Token, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return uc.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return &dto.ValidateTokenOutput{Valid: false}, nil
	}

	return &dto.ValidateTokenOutput{
		Valid:    true,
		UserID:   claims.UserID,
		Username: claims.Username,
		Role:     claims.Role,
	}, nil
}

func (uc *authUseCase) GetUserByID(ctx context.Context, userID string) (*dto.UserOutput, error) {
	if userID == "" {
		return nil, exceptions.NewValidationError("user_id is required")
	}

	user, err := uc.uow.Users().FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &dto.UserOutput{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}, nil
}
