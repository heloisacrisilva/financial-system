package operation

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"financial/system/api/config"
	"financial/system/api/entities"
	repositoryErrors "financial/system/api/repository"
	repository "financial/system/api/repository/operation"

	"github.com/gin-gonic/gin"
)

func TestCreateOperationHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config.InitLogger("[Test OPERATION]")

	strPtr := func(s string) *string { return &s }

	type testCase struct {
		name           string
		opTypeParam    string
		inputPayload   OperationRequest
		setupMocks     func()
		expectedStatus int
	}

	tests := []testCase{
		{
			name:        "Sucesso - Operacao de Credito",
			opTypeParam: "credit",
			inputPayload: OperationRequest{
				AccountID: 1,
				Value:     1500,
				Currency:  "BRL",
			},
			setupMocks: func() {
				creditOperationDbFunc = func(accountID uint64, currency string, value int64, referenceID string) (*entities.Account, *entities.TransactionHistory, error) {
					return &entities.Account{ID: 1, AvailableBalance: 2000, ReservedBalance: 0, CreditLimit: 1000},
						&entities.TransactionHistory{ID: 101, ReferenceID: "ref-123", Status: "completed", CreatedAt: time.Now()},
						nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Erro Validação - Valor Negativo",
			opTypeParam: "credit",
			inputPayload: OperationRequest{
				AccountID: 1,
				Value:     -500,
				Currency:  "BRL",
			},
			setupMocks:     func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "Erro Negocio - Saldo Insuficiente com Gravacao de Falha",
			opTypeParam: "debit",
			inputPayload: OperationRequest{
				AccountID: 1,
				Value:     999999,
				Currency:  "BRL",
			},
			setupMocks: func() {
				debitOperationDbFunc = func(accountID uint64, currency string, value int64, referenceID string) (*entities.Account, *entities.TransactionHistory, error) {
					return nil, nil, repositoryErrors.ErrInsufficientFunds
				}
				recordFailedOperationDbFunc = func(opType string, accountID uint64, referenceID string, value int64, currency string, err error) {}
			},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:        "Sucesso - Transferencia entre Contas",
			opTypeParam: "transfer",
			inputPayload: OperationRequest{
				AccountID:     1,
				AccountDestID: 2,
				Value:         500,
				Currency:      "BRL",
			},
			setupMocks: func() {
				transferOperationDbFunc = func(accountID, accountDestID uint64, currency string, value int64, referenceID string) ([]entities.TransactionHistory, *entities.Account, *entities.Account, error) {
					histories := []entities.TransactionHistory{
						{ID: 201, ReferenceID: "ref-deb", Status: "completed", TransferGroupID: strPtr("group-tg"), CreatedAt: time.Now()},
						{ID: 202, ReferenceID: "ref-cred", Status: "completed", TransferGroupID: strPtr("group-tg"), CreatedAt: time.Now()},
					}
					origin := &entities.Account{ID: 1, AvailableBalance: 500, ReservedBalance: 0, CreditLimit: 500}
					dest := &entities.Account{ID: 2, AvailableBalance: 1500, ReservedBalance: 0, CreditLimit: 0}
					return histories, origin, dest, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Sucesso - Estorno de Transacao (Reversal)",
			opTypeParam: "reversal",
			inputPayload: OperationRequest{
				OriginalReferenceID: "ref-original-existente",
			},
			setupMocks: func() {
				reversalOperationDbFunc = func(origRefID, refID string, value int64) ([]*entities.Account, []entities.TransactionHistory, error) {
					accs := []*entities.Account{
						{ID: 1, AvailableBalance: 1000, ReservedBalance: 0},
					}
					hists := []entities.TransactionHistory{
						{ID: 301, ReferenceID: "ref-rev", Status: "reversed", CreatedAt: time.Now()},
					}
					return accs, hists, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		tc.setupMocks()

		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.POST("/operations/:type", CreateOperation)

			body, _ := json.Marshal(tc.inputPayload)
			req, err := http.NewRequest(http.MethodPost, "/operations/"+tc.opTypeParam, bytes.NewBuffer(body))
			if err != nil {
				t.Fatalf("Falha ao criar requisição: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Status Code incorreto para '%s'. Obtido: %d, Esperado: %d. Resposta: %s",
					tc.name, w.Code, tc.expectedStatus, w.Body.String())
			}
		})

		creditOperationDbFunc = repository.CreditOperation
		debitOperationDbFunc = repository.DebitOperation
		reserveOperationDbFunc = repository.ReserveOperation
		captureOperationDbFunc = repository.CaptureOperation
		transferOperationDbFunc = repository.TransferOperation
		reversalOperationDbFunc = repository.ReversalOperation
		recordFailedOperationDbFunc = repository.RecordFailedOperation
	}
}
