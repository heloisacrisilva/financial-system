package operation

import (
	"errors"
	"financial/system/api/config"
	"financial/system/api/entities"
	"financial/system/api/handler"
	"financial/system/api/helpers"
	repositoryErrors "financial/system/api/repository"
	repository "financial/system/api/repository/operation"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type OperationRequest struct {
	AccountID   string `json:"account_id" binding:"required"`
	Value       int64  `json:"value" binding:"required,gt=0"`
	Currency    string `json:"currency"`
	ReferenceID string `json:"reference_id" binding:"required"`
}

type OperationResponse struct {
	TransactionID    string    `json:"transaction_id"`
	Status           string    `json:"status"`
	Balance          int64     `json:"balance"`
	ReservedBalance  int64     `json:"reserved_balance"`
	AvailableBalance int64     `json:"available_balance"`
	CreditUsed       int64     `json:"credit_used"`
	CreditAvailable  int64     `json:"credit_available"`
	Timestamp        time.Time `json:"timestamp"`
	ErrorMessage     *string   `json:"error_message"`
	Type             string    `json:"type"`
}

func CreateOperation(ctx *gin.Context) {
	logger := config.GetLogger()
	opType := ctx.Param("type")
	availableTypes := []string{"debit", "credit", "reserve", "capture"}

	if !slices.Contains(availableTypes, opType) {
		handler.SendError(ctx, http.StatusBadRequest, "Invalid operation type.")
		return
	}

	var req OperationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logger.Errorf("Invalid JSON payload: %v", err)
		handler.SendError(ctx, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := req.OperationValidate(); err != nil {
		logger.Errorf("Validation failed: %v", err)
		handler.SendError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	const maxRetries = 3
	delay := 100 * time.Millisecond

	var account *entities.Account
	var history *entities.TransactionHistory
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {

		switch opType {
		case "credit":
			account, history, err = repository.CreditOperation(req.AccountID, req.Currency, req.Value, req.ReferenceID)
		case "debit":
			account, history, err = repository.DebitOperation(req.AccountID, req.Currency, req.Value, req.ReferenceID)
		case "reserve":
			account, history, err = repository.ReserveOperation(req.AccountID, req.Currency, req.Value, req.ReferenceID)

		case "capture":
			account, history, err = repository.CaptureOperation(req.AccountID, req.Currency, req.Value, req.ReferenceID)
		}

		if err == nil {
			break
		}

		if !helpers.IsRetryable(err) {
			break
		}
		logger.Warnf("Retryable error on operation %s ref=%s, attempt %d/%d: %v - delay ", opType, req.ReferenceID, attempt, maxRetries, err, delay)
		time.Sleep(delay)
		delay *= 2
	}

	if err != nil {
		switch {
		case errors.Is(err, repositoryErrors.ErrDuplicateRef):
			handler.SendError(ctx, http.StatusConflict, "Transaction reference already processed")
			return
		case errors.Is(err, repositoryErrors.ErrAccountNotFound):
			handler.SendError(ctx, http.StatusNotFound, "Account not found")
			return
		case errors.Is(err, repositoryErrors.ErrInsufficientFunds):
			repository.RecordFailedOperation(opType, req.AccountID, req.ReferenceID, req.Value, req.Currency, err)
			handler.SendError(ctx, http.StatusUnprocessableEntity, err.Error())
			return
		case errors.Is(err, repositoryErrors.ErrAccountNotActive),
			errors.Is(err, repositoryErrors.ErrInvalidCurrency):
			repository.RecordFailedOperation(opType, req.AccountID, req.ReferenceID, req.Value, req.Currency, err)
			handler.SendError(ctx, http.StatusUnprocessableEntity, err.Error())
			return
		default:
			logger.Errorf("Operation %s failed after retries: %v", opType, err)
			handler.SendError(ctx, http.StatusInternalServerError, "Internal error processing operation")
			return
		}
	}

	usedCredit := int64(0)
	if account.AvailableBalance < 0 {
		usedCredit = -account.AvailableBalance
	}

	resp := OperationResponse{
		TransactionID:    history.ReferenceID + "-PROCESSED",
		Status:           history.Status,
		Balance:          account.AvailableBalance + account.ReservedBalance,
		ReservedBalance:  account.ReservedBalance,
		AvailableBalance: account.AvailableBalance,
		CreditUsed:       usedCredit,
		CreditAvailable:  account.CreditLimit - usedCredit,
		Timestamp:        history.CreatedAt,
		Type:             opType,
		ErrorMessage:     nil,
	}
	handler.SendSuccess(ctx, "create-operation", "data", resp)
}

func (r *OperationRequest) OperationValidate() error {
	r.AccountID = strings.TrimSpace(r.AccountID)
	if r.AccountID == "" {
		return handler.ErrParamIsRequired("account_id", "string")
	}

	if r.Value <= 0 {
		return handler.ErrInvalidParam("value", "int64")
	}

	r.ReferenceID = strings.TrimSpace(r.ReferenceID)
	if r.ReferenceID == "" {
		return handler.ErrParamIsRequired("reference_id", "string")
	}

	if r.Currency != "" && len(r.Currency) != 3 {
		return handler.ErrInvalidParam("currency", "string")
	}

	return nil
}
