package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ResponseEnvelope struct {
	StatusCode int         `json:"statusCode"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data,omitempty"`
}

type ErrorEnvelope struct {
	StatusCode int    `json:"statusCode"`
	Error      string `json:"error"`
	Details    string `json:"details,omitempty"`
}

func Respond(ctx *gin.Context, statusCode int, message string, data interface{}) {
	if message == "" {
		message = http.StatusText(statusCode)
	}

	ctx.JSON(statusCode, ResponseEnvelope{
		StatusCode: statusCode,
		Message:    message,
		Data:       data,
	})
}

func RespondError(ctx *gin.Context, statusCode int, errMessage string, details string) {
	ctx.JSON(statusCode, ErrorEnvelope{
		StatusCode: statusCode,
		Error:      errMessage,
		Details:    details,
	})
}

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
