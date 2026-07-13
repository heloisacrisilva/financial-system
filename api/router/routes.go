package router

import (
	account "financial/system/api/handler/account"
	client "financial/system/api/handler/client"
	docs "financial/system/api/handler/docs"
	health "financial/system/api/handler/health"
	operation "financial/system/api/handler/operation"

	"github.com/gin-gonic/gin"
)

const basePath = "/api"

func InitializeRoutes(router *gin.Engine) {
	router.GET("/swagger", docs.SwaggerUI)
	router.StaticFile("/openapi.yaml", "api/docs/openapi.yaml")
	router.GET("/health", health.GetHealth)

	api := router.Group(basePath)

	accounts := api.Group("/accounts")
	{
		accounts.POST("/create", account.CreateAccountHandler)
		accounts.GET("/:id", account.GetAccountByID)
	}
	operations := api.Group("/operations")
	{
		operations.POST("/:type", operation.CreateOperation)
		operations.GET("/:id", operation.GetOperationByID)
	}
	clients := api.Group("/clients")
	{
		clients.GET("/:id", client.GetClientByID)
	}
}
