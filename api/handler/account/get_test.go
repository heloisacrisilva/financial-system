package account

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"financial/system/api/config"
	"financial/system/api/entities"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestGetAccountHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config.InitLogger("[Test GET]")

	type testCase struct {
		name           string
		accountID      uint64
		mockDbBehavior func(accountID string) (entities.Account, error)
		expectedStatus int
	}

	tests := []testCase{
		{
			name:      "Sucesso ao buscar conta existente",
			accountID: 123,
			mockDbBehavior: func(accountID string) (entities.Account, error) {
				idUint, _ := strconv.ParseUint(accountID, 10, 64)
				return entities.Account{
					ID: idUint,
				}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "Erro - Conta nao encontrada (404)",
			accountID: 999,
			mockDbBehavior: func(accountID string) (entities.Account, error) {
				return entities.Account{}, gorm.ErrRecordNotFound
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:      "Erro - Falha interna no banco de dados (500)",
			accountID: 123,
			mockDbBehavior: func(accountID string) (entities.Account, error) {
				return entities.Account{}, errors.New("database connection broken")
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			oldDbFunc := getAccountDbFunc

			if tc.mockDbBehavior != nil {
				getAccountDbFunc = tc.mockDbBehavior
			} else {
				getAccountDbFunc = func(accountID string) (entities.Account, error) {
					return entities.Account{}, nil
				}
			}

			router := gin.New()
			router.GET("/accounts/:id", GetAccountByID)

			idStr := strconv.FormatUint(tc.accountID, 10)
			req, err := http.NewRequest(http.MethodGet, "/accounts/"+idStr, nil)
			if err != nil {
				t.Fatalf("Falha ao criar requisição: %v", err)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			getAccountDbFunc = oldDbFunc
			if w.Code != tc.expectedStatus {
				t.Errorf("Status Code incorreto para '%s'. Obtido: %d, Esperado: %d. Resposta: %s",
					tc.name, w.Code, tc.expectedStatus, w.Body.String())
			}
		})
	}
}
