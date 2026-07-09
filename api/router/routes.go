package router

import (
	account "financial/system/api/handler/account"
	operation "financial/system/api/handler/operation"

	"github.com/gin-gonic/gin"
)

const basePath = "/api"

func InitializeRoutes(router *gin.Engine) {
	api := router.Group(basePath)

	accounts := api.Group("/accounts")
	{
		accounts.POST("", account.CreateAccount)
		accounts.GET("/:id", account.GetAccountByID)
	}
	operations := api.Group("/operations")
	{
		operations.POST("/:type", operation.CreateOperation)
		operations.GET("/:id", operation.GetOperationByID)
	}
}
