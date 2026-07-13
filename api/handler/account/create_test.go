package account

import (
	"bytes"
	"encoding/json"
	"errors"
	"financial/system/api/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCreateAccountHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.InitLogger("[Test POST]")

	type testCase struct {
		name           string
		inputPayload   interface{}
		mockDbBehavior func(name, email, cpf string) ([]map[string]interface{}, error)
		expectedStatus int
	}

	tests := []testCase{
		{
			name: "Sucesso ao criar conta",
			inputPayload: CreateAccountRequest{
				Name:  "Pessoa 1",
				Email: "teste1@mail.com",
				CPF:   "000.000.000-01",
			},
			mockDbBehavior: func(name, email, cpf string) ([]map[string]interface{}, error) {
				accountMock := []map[string]interface{}{
					{
						"name":  name,
						"email": email,
						"cpf":   cpf,
					},
				}
				return accountMock, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Erro de validação - Falta name",
			inputPayload: CreateAccountRequest{
				Email: "teste1@mail.com",
				CPF:   "000.000.000-01",
			},
			mockDbBehavior: nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Erro de validação - Email inválido",
			inputPayload: CreateAccountRequest{
				Name:  "Pessoa 1",
				Email: "email-invalido",
				CPF:   "000.000.000-01",
			},
			mockDbBehavior: nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Erro de conflito - Email já cadastrado",
			inputPayload: CreateAccountRequest{
				Name:  "Pessoa 1",
				Email: "teste1@mail.com",
				CPF:   "000.000.000-01",
			},
			mockDbBehavior: func(name, email, cpf string) ([]map[string]interface{}, error) {
				return nil, errors.New("This email is already in use by another user.")
			},
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			oldDbFunc := createAccountDbFunc
			defer func() { createAccountDbFunc = oldDbFunc }()

			if tc.mockDbBehavior != nil {
				createAccountDbFunc = tc.mockDbBehavior
			} else {
				createAccountDbFunc = func(name, email, cpf string) ([]map[string]interface{}, error) {
					return nil, nil
				}
			}

			router := gin.New()
			router.POST("/accounts", CreateAccountHandler)

			body, _ := json.Marshal(tc.inputPayload)
			req, err := http.NewRequest(http.MethodPost, "/accounts", bytes.NewBuffer(body))
			if err != nil {
				t.Fatalf("Falha ao criar requisição: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			createAccountDbFunc = oldDbFunc

			if w.Code != tc.expectedStatus {
				t.Errorf("Status Code incorreto para '%s'. Obtido: %d, Esperado: %d. Resposta: %s",
					tc.name, w.Code, tc.expectedStatus, w.Body.String())
			}
		})
	}
}
