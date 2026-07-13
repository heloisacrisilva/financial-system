package health

import (
	"encoding/json"
	"financial/system/api/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetHealthHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config.InitLogger("[Test GET]")

	type testCase struct {
		name              string
		mockDbHealthy     bool
		expectedStatus    int
		expectedStatusStr string
	}

	tests := []testCase{
		{
			name:              "Sucesso - API e Banco de dados operacionais",
			mockDbHealthy:     true,
			expectedStatus:    http.StatusOK,
			expectedStatusStr: "ok",
		},
		{
			name:              "Erro - Banco de dados fora do ar",
			mockDbHealthy:     false,
			expectedStatus:    http.StatusServiceUnavailable,
			expectedStatusStr: "unhealthy",
		},
	}

	for _, tc := range tests {
		checkDatabaseHealthFunc = func() bool {
			return tc.mockDbHealthy
		}

		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/health", GetHealth)

			req, err := http.NewRequest(http.MethodGet, "/health", nil)
			if err != nil {
				t.Fatalf("Falha ao criar requisição: %v", err)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Status Code incorreto para '%s'. Obtido: %d, Esperado: %d. Resposta: %s",
					tc.name, w.Code, tc.expectedStatus, w.Body.String())
			}

			var response HealthResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Falha ao desserializar resposta: %v", err)
			}

			if response.Status != tc.expectedStatusStr {
				t.Errorf("Status interno do JSON incorreto para '%s'. Obtido: %s, Esperado: %s",
					tc.name, response.Status, tc.expectedStatusStr)
			}
		})
	}

	checkDatabaseHealthFunc = func() bool {
		db := config.GetPostgres()
		if db == nil {
			return false
		}
		sqlDB, err := db.DB()
		if err != nil {
			return false
		}
		return sqlDB.Ping() == nil
	}
}
