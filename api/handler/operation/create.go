package operation

import (
	"errors"
	"financial/system/api/config"
	"financial/system/api/entities"
	"financial/system/api/handler"
	"financial/system/api/helpers"
	repository "financial/system/api/repository/operation"
	"net/http"
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
	Timestamp        time.Time `json:"timestamp"`
	ErrorMessage     *string   `json:"error_message"`
}

func CreateOperation(ctx *gin.Context) {
	logger := config.GetLogger()
	opType := ctx.Param("type")

	//FIXME:
	if opType != "credit" {
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

	for attempt := 1; attempt < maxRetries; attempt++ {
		if opType == "credit" {
			account, history, err = repository.CreditOperation(req.AccountID, req.Currency, req.Value, req.ReferenceID)
		}

		if err == nil {
			break
		}

		if !helpers.IsRetryable(err) {
			break
		}
		logger.Warnf("Retryable error on credit ref=%s, attempt %d/%d: %v", req.ReferenceID, attempt, maxRetries, delay)
		time.Sleep(delay)
		delay *= 2
	}

	if err != nil {
		switch {
		case errors.Is(err, repository.ErrDuplicateRef):
			handler.SendError(ctx, http.StatusConflict, "Transaction reference already processed")
			return
		case errors.Is(err, repository.ErrAccountNotFound):
			handler.SendError(ctx, http.StatusNotFound, "Account not found")
			return
		case errors.Is(err, repository.ErrAccountNotActive),
			errors.Is(err, repository.ErrInvalidCurrency):
			repository.RecordFailedCredit(req.AccountID, req.ReferenceID, req.Value, req.Currency, err)
			handler.SendError(ctx, http.StatusUnprocessableEntity, err.Error())
			return
		default:
			logger.Errorf("Credit operation failed after retries: %v", err)
			handler.SendError(ctx, http.StatusInternalServerError, "Internal error processing operation")
			return
		}
	}

	resp := OperationResponse{
		TransactionID:    history.ReferenceID + "-PROCESSED",
		Status:           history.Status,
		Balance:          account.AvailableBalance + account.ReservedBalance,
		ReservedBalance:  account.ReservedBalance,
		AvailableBalance: account.AvailableBalance,
		Timestamp:        history.CreatedAt,
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
