package client

import (
	"errors"
	"financial/system/api/config"
	"financial/system/api/handler"
	repository "financial/system/api/repository/client"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var getClientDbFunc = repository.GetClientRepository

func GetClientByID(ctx *gin.Context) {
	logger := config.GetLogger()

	clientID := ctx.Param("id")
	if clientID == "" {
		logger.Warn("Missing required route parameter: ID")
		handler.SendError(ctx, http.StatusBadRequest, handler.ErrParamIsRequired("ID", "routeParameter").Error())
		return
	}

	client, err := getClientDbFunc(clientID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warnf("Client with ID %s not found.", clientID)
			handler.SendError(ctx, http.StatusNotFound, "Client not found.")
			return
		}

		logger.Errorf("Error to get client: %v", err)
		handler.SendError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	handler.SendSuccess(ctx, "get-client", "client", client)
}
