package v1

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/myavarih/sre-hands-on/pkg/exceptions"
	authv1 "github.com/myavarih/sre-hands-on/services/gen/go/auth/v1"
	"github.com/myavarih/sre-hands-on/services/storefront-gateway/internal/presentation/controllers"
)

type AuthController struct {
	authClient authv1.AuthServiceClient
	logger     *slog.Logger
}

func NewAuthController(authClient authv1.AuthServiceClient, logger *slog.Logger) *AuthController {
	return &AuthController{
		authClient: authClient,
		logger:     logger,
	}
}

type registerRequestParams struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

func (c *AuthController) Register(ctx *gin.Context) {
	params, ok := controllers.Receive[registerRequestParams](ctx)
	if !ok {
		return
	}

	reqCtx, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancel()

	resp, err := c.authClient.Register(reqCtx, &authv1.RegisterRequest{
		Username: params.Username,
		Email:    params.Email,
		Password: params.Password,
	})
	if err != nil {
		c.logger.Warn("Register RPC failed", "error", err)
		dErr := exceptions.FromGRPCError(err)
		controllers.RespondError(ctx, exceptions.HTTPStatusCode(dErr), dErr.Error(), "")
		return
	}

	controllers.Respond(ctx, http.StatusCreated, "User registered successfully", gin.H{
		"user_id":  resp.GetUserId(),
		"username": resp.GetUsername(),
		"email":    resp.GetEmail(),
	})
}

type loginRequestParams struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (c *AuthController) Login(ctx *gin.Context) {
	params, ok := controllers.Receive[loginRequestParams](ctx)
	if !ok {
		return
	}

	reqCtx, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancel()

	resp, err := c.authClient.Login(reqCtx, &authv1.LoginRequest{
		Username: params.Username,
		Password: params.Password,
	})
	if err != nil {
		c.logger.Warn("Login RPC failed", "error", err)
		controllers.RespondError(ctx, http.StatusUnauthorized, "Invalid credentials", "")
		return
	}

	controllers.Respond(ctx, http.StatusOK, "Login successful", gin.H{
		"token":      resp.GetToken(),
		"expires_in": resp.GetExpiresIn(),
		"user_id":    resp.GetUserId(),
		"username":   resp.GetUsername(),
	})
}
