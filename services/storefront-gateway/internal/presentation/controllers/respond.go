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
