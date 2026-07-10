package account

import (
	"errors"
	"financial/system/api/config"
	"financial/system/api/handler"
	repository "financial/system/api/repository/account"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetAccountByID(ctx *gin.Context) {
	logger := config.GetLogger()

	accountID := ctx.Param("id")
	if accountID == "" {
		logger.Warn("Missing required route parameter: ID")
		handler.SendError(ctx, http.StatusBadRequest, handler.ErrParamIsRequired("ID", "routeParameter").Error())
		return
	}

	account, err := repository.GetAccountRepository(accountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warnf("Account with ID %s not found.", accountID)
			handler.SendError(ctx, http.StatusNotFound, "Account not found.")
			return
		}

		logger.Errorf("Error to get account: %v", err)
		handler.SendError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	handler.SendSuccess(ctx, "get-account", "account", account)
}
