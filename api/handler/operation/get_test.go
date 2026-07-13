package operation

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"financial/system/api/entities"
	repository "financial/system/api/repository/operation"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestGetOperationByIDHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type testCase struct {
		name           string
		operationIDStr string
		mockDbBehavior func(id uint64) (entities.TransactionHistory, error)
		expectedStatus int
	}

	tests := []testCase{
		{
			name:           "Sucesso ao buscar historico de operacao",
			operationIDStr: "50",
			mockDbBehavior: func(id uint64) (entities.TransactionHistory, error) {
				return entities.TransactionHistory{
					ID:        id,
					AccountID: 10,
					Type:      "credit",
					Value:     5000,
					Currency:  "BRL",
					Status:    "completed",
					CreatedAt: time.Now(),
				}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Erro - ID invalido passado na URL",
			operationIDStr: "abc",
			mockDbBehavior: nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Erro - ID igual a zero",
			operationIDStr: "0",
			mockDbBehavior: nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Erro - Historico nao encontrado (404)",
			operationIDStr: "999",
			mockDbBehavior: func(id uint64) (entities.TransactionHistory, error) {
				return entities.TransactionHistory{}, gorm.ErrRecordNotFound
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Erro - Falha interna no banco de dados (500)",
			operationIDStr: "50",
			mockDbBehavior: func(id uint64) (entities.TransactionHistory, error) {
				return entities.TransactionHistory{}, errors.New("database failure")
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		if tc.mockDbBehavior != nil {
			getOperationHistoryByIDDbFunc = tc.mockDbBehavior
		} else {
			getOperationHistoryByIDDbFunc = func(id uint64) (entities.TransactionHistory, error) {
				return entities.TransactionHistory{}, nil
			}
		}

		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/operations/:id", GetOperationByID)

			req, err := http.NewRequest(http.MethodGet, "/operations/"+tc.operationIDStr, nil)
			if err != nil {
				t.Fatalf("Falha ao criar requisicao: %v", err)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Status Code incorreto para '%s'. Obtido: %d, Esperado: %d. Resposta: %s",
					tc.name, w.Code, tc.expectedStatus, w.Body.String())
			}
		})

		getOperationHistoryByIDDbFunc = repository.GetOperationHistoryByID
	}
}
