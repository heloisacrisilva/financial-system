package router

import (
	account "financial/system/api/handler/account"
	client "financial/system/api/handler/client"
	operation "financial/system/api/handler/operation"

	"github.com/gin-gonic/gin"
)

const basePath = "/api"

func InitializeRoutes(router *gin.Engine) {
	api := router.Group(basePath)

	accounts := api.Group("/accounts")
	{
		accounts.POST("/create", account.CreateAccountHandler)
		accounts.GET("/:id", account.GetAccountByID)
	}
	operations := api.Group("/operations")
	{
		operations.POST("/create/:type", operation.CreateOperation)
		operations.GET("/:id", operation.GetOperationByID)
	}
	clients := api.Group("/clients")
	{
		clients.GET("/:id", client.GetClientByID)
	}
}
