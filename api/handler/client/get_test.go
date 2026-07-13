package client

import (
	"errors"
	"financial/system/api/config"
	"financial/system/api/entities"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestGetClientHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config.InitLogger("[Test GET]")

	type testCase struct {
		name           string
		clientID       uint64
		mockDbBehavior func(clientID string) (entities.Client, error)
		expectedStatus int
	}

	tests := []testCase{
		{
			name:     "Sucesso ao buscar cliente existente",
			clientID: 123,
			mockDbBehavior: func(clientID string) (entities.Client, error) {
				idUint, _ := strconv.ParseUint(clientID, 10, 64)
				return entities.Client{
					ID: idUint,
				}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "Erro - Cliente nao encontrado (404)",
			clientID: 999,
			mockDbBehavior: func(clientID string) (entities.Client, error) {
				return entities.Client{}, gorm.ErrRecordNotFound
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:     "Erro - Falha interna no banco de dados (500)",
			clientID: 123,
			mockDbBehavior: func(clientID string) (entities.Client, error) {
				return entities.Client{}, errors.New("database connection broken")
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			oldDbFunc := getClientDbFunc

			if tc.mockDbBehavior != nil {
				getClientDbFunc = tc.mockDbBehavior
			} else {
				getClientDbFunc = func(clientID string) (entities.Client, error) {
					return entities.Client{}, nil
				}
			}

			router := gin.New()
			router.GET("/clients/:id", GetClientByID)

			idStr := strconv.FormatUint(tc.clientID, 10)
			req, err := http.NewRequest(http.MethodGet, "/clients/"+idStr, nil)
			if err != nil {
				t.Fatalf("Falha ao criar requisição: %v", err)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			getClientDbFunc = oldDbFunc
			if w.Code != tc.expectedStatus {
				t.Errorf("Status Code incorreto para '%s'. Obtido: %d, Esperado: %d. Resposta: %s",
					tc.name, w.Code, tc.expectedStatus, w.Body.String())
			}
		})
	}
}
