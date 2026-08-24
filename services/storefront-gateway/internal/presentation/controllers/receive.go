package controllers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Receive binds JSON body, URI parameters, and Query parameters into struct T.
func Receive[T any](ctx *gin.Context) (T, bool) {
	var params T

	if ctx.Request.ContentLength > 0 {
		if err := ctx.ShouldBindJSON(&params); err != nil {
			RespondError(ctx, http.StatusBadRequest, "Invalid JSON payload format", err.Error())
			return params, false
		}
	}

	if err := ctx.ShouldBindUri(&params); err != nil {
		RespondError(ctx, http.StatusBadRequest, "Invalid URI parameters", err.Error())
		return params, false
	}

	if err := ctx.ShouldBindQuery(&params); err != nil {
		RespondError(ctx, http.StatusBadRequest, "Invalid query parameters", err.Error())
		return params, false
	}

	return params, true
}

func GetUserID(ctx *gin.Context) (string, error) {
	val, exists := ctx.Get("user_id")
	if !exists {
		return "", errors.New("unauthorized: missing user context")
	}
	userID, ok := val.(string)
	if !ok || userID == "" {
		return "", errors.New("unauthorized: invalid user context")
	}
	return userID, nil
}
