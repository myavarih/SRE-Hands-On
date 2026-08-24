package controllers

import (
	"context"
	"time"

	"github.com/myavarih/sre-hands-on/pkg/exceptions"
	authv1 "github.com/myavarih/sre-hands-on/services/gen/go/auth/v1"
	"github.com/myavarih/sre-hands-on/services/auth-service/internal/application/dto"
	"github.com/myavarih/sre-hands-on/services/auth-service/internal/application/usecase"
)

// AuthGrpcController is a thin transport adapter mapping Protobuf to Application DTOs.
type AuthGrpcController struct {
	authv1.UnimplementedAuthServiceServer
	authUC usecase.AuthUseCase
}

func NewAuthGrpcController(authUC usecase.AuthUseCase) *AuthGrpcController {
	return &AuthGrpcController{authUC: authUC}
}

func (c *AuthGrpcController) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	out, err := c.authUC.Register(ctx, dto.RegisterInput{
		Username: req.GetUsername(),
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	})
	if err != nil {
		return nil, exceptions.ToGRPCError(err)
	}

	return &authv1.RegisterResponse{
		UserId:   out.ID,
		Username: out.Username,
		Email:    out.Email,
	}, nil
}

func (c *AuthGrpcController) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	out, err := c.authUC.Login(ctx, dto.LoginInput{
		Username: req.GetUsername(),
		Password: req.GetPassword(),
	})
	if err != nil {
		return nil, exceptions.ToGRPCError(err)
	}

	return &authv1.LoginResponse{
		Token:     out.Token,
		ExpiresIn: out.ExpiresIn,
		UserId:    out.UserID,
		Username:  out.Username,
	}, nil
}

func (c *AuthGrpcController) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	out, err := c.authUC.ValidateToken(ctx, dto.ValidateTokenInput{
		Token: req.GetToken(),
	})
	if err != nil {
		return nil, exceptions.ToGRPCError(err)
	}

	return &authv1.ValidateTokenResponse{
		Valid:    out.Valid,
		UserId:   out.UserID,
		Username: out.Username,
		Role:     out.Role,
	}, nil
}

func (c *AuthGrpcController) GetUser(ctx context.Context, req *authv1.GetUserRequest) (*authv1.GetUserResponse, error) {
	out, err := c.authUC.GetUserByID(ctx, req.GetUserId())
	if err != nil {
		return nil, exceptions.ToGRPCError(err)
	}

	return &authv1.GetUserResponse{
		UserId:    out.ID,
		Username:  out.Username,
		Email:     out.Email,
		Role:      out.Role,
		CreatedAt: out.CreatedAt.Format(time.RFC3339),
	}, nil
}
