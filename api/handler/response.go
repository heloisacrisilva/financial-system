package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SendSuccess(ctx *gin.Context, operation string, responsekey string, data interface{}) {
	ctx.Header("Content-type", "application/json")
	ctx.JSON(http.StatusOK, gin.H{
		"message":   fmt.Sprintf("Operation '%s' completed successfully", operation),
		responsekey: data,
	})
}

func SendError(ctx *gin.Context, statusCode int, message string) {
	ctx.Header("Content-type", "application/json")
	ctx.JSON(statusCode, gin.H{
		"message":   message,
		"errorCode": statusCode,
	})
}

func ErrParamIsRequired(name, typ string) error {
	return fmt.Errorf("param: %s (type: %s) is required", name, typ)
}

func ErrInvalidParam(name, typ string) error {
	return fmt.Errorf("param: %s (type: %s) is invalid", name, typ)
}
