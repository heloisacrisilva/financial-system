package health

import (
	"financial/system/api/config"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
}

var checkDatabaseHealthFunc = func() bool {
	db := config.GetPostgres()
	if db == nil {
		return false
	}

	sqlDB, err := db.DB()
	if err != nil {
		return false
	}

	if err := sqlDB.Ping(); err != nil {
		return false
	}

	return true
}

func GetHealth(ctx *gin.Context) {
	response := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
		Checks: map[string]string{
			"api":      "ok",
			"database": "ok",
		},
	}

	if !checkDatabaseHealthFunc() {
		response.Status = "unhealthy"
		response.Checks["database"] = "unavailable"
		ctx.JSON(http.StatusServiceUnavailable, response)
		return
	}

	ctx.JSON(http.StatusOK, response)
}
