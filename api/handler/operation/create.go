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
	"github.com/google/uuid"
)

var creditOperationDbFunc = repository.CreditOperation
var debitOperationDbFunc = repository.DebitOperation
var reserveOperationDbFunc = repository.ReserveOperation
var captureOperationDbFunc = repository.CaptureOperation
var transferOperationDbFunc = repository.TransferOperation
var reversalOperationDbFunc = repository.ReversalOperation
var recordFailedOperationDbFunc = repository.RecordFailedOperation

type OperationRequest struct {
	AccountID           uint64 `json:"account_id"`
	AccountDestID       uint64 `json:"account_dest_id"`
	OriginalReferenceID string `json:"original_reference_id"`
	Value               int64  `json:"value"`
	Currency            string `json:"currency"`
	ReferenceID         string `json:"-"`
}

type OperationResponse struct {
	TransactionID    string    `json:"transaction_id"`
	ReferenceID      string    `json:"reference_id"`
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

type TransferOperationResponse struct {
	TransactionID   string    `json:"transaction_id"`
	ReferenceID     string    `json:"reference_id"`
	Status          string    `json:"status"`
	Type            string    `json:"type"`
	OriginAccountID uint64    `json:"origin_account_id"`
	DestAccountID   uint64    `json:"dest_account_id"`
	OriginBalance   int64     `json:"origin_balance"`
	DestBalance     int64     `json:"dest_balance"`
	CreditUsed      int64     `json:"credit_used"`
	CreditAvailable int64     `json:"credit_available"`
	Timestamp       time.Time `json:"timestamp"`
	ErrorMessage    *string   `json:"error_message"`
	DebitTxID       uint64    `json:"debit_tx_id"`
	CreditTxID      uint64    `json:"credit_tx_id"`
	TransferGroupID *string   `json:"transfer_group_id"`
}

type ReversalAccountResult struct {
	AccountID        uint64 `json:"account_id"`
	Balance          int64  `json:"balance"`
	AvailableBalance int64  `json:"available_balance"`
	ReservedBalance  int64  `json:"reserved_balance"`
}

type ReversalOperationResponse struct {
	TransactionID   string                  `json:"transaction_id"`
	ReferenceID     string                  `json:"reference_id"`
	Status          string                  `json:"status"`
	Type            string                  `json:"type"`
	Timestamp       time.Time               `json:"timestamp"`
	ErrorMessage    *string                 `json:"error_message"`
	OriginalRefID   string                  `json:"original_reference_id"`
	ReversalTxIDs   []uint64                `json:"reversal_tx_ids"`
	TransferGroupID *string                 `json:"transfer_group_id,omitempty"`
	Accounts        []ReversalAccountResult `json:"accounts"`
}

func CreateOperation(ctx *gin.Context) {
	logger := config.GetLogger()
	opType := ctx.Param("type")
	availableTypes := []string{"debit", "credit", "reserve", "capture", "reversal", "transfer"}

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

	if err := req.OperationValidate(opType); err != nil {
		logger.Errorf("Validation failed: %v", err)
		handler.SendError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	const maxRetries = 3
	delay := 100 * time.Millisecond

	var originAccount, destAccount *entities.Account
	var account *entities.Account
	var history *entities.TransactionHistory
	var transferHistories []entities.TransactionHistory
	var reversalAccounts []*entities.Account
	var reversalHistories []entities.TransactionHistory
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {

		switch opType {
		case "credit":
			account, history, err = creditOperationDbFunc(req.AccountID, req.Currency, req.Value, req.ReferenceID)
		case "debit":
			account, history, err = debitOperationDbFunc(req.AccountID, req.Currency, req.Value, req.ReferenceID)
		case "reserve":
			account, history, err = reserveOperationDbFunc(req.AccountID, req.Currency, req.Value, req.ReferenceID)
		case "capture":
			account, history, err = captureOperationDbFunc(req.AccountID, req.Currency, req.Value, req.ReferenceID)
		case "transfer":
			transferHistories, originAccount, destAccount, err = transferOperationDbFunc(req.AccountID, req.AccountDestID, req.Currency, req.Value, req.ReferenceID)
		case "reversal":
			reversalAccounts, reversalHistories, err = reversalOperationDbFunc(req.OriginalReferenceID, req.ReferenceID, req.Value)
		}

		if err == nil {
			break
		}

		if !helpers.IsRetryable(err) {
			break
		}
		logger.Warnf("Retryable error on operation %s ref=%s, attempt %d/%d: %v - delay %v", opType, req.ReferenceID, attempt, maxRetries, err, delay)
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
		case errors.Is(err, repositoryErrors.ErrSameAccount):
			handler.SendError(ctx, http.StatusUnprocessableEntity, err.Error())
			return
		case errors.Is(err, repositoryErrors.ErrTransactionNotFound):
			handler.SendError(ctx, http.StatusNotFound, "Original transaction not found")
			return
		case errors.Is(err, repositoryErrors.ErrAlreadyReversed):
			handler.SendError(ctx, http.StatusConflict, "Transaction has already been reversed")
			return
		case errors.Is(err, repositoryErrors.ErrTransactionNotReversible):
			handler.SendError(ctx, http.StatusUnprocessableEntity, err.Error())
			return
		case errors.Is(err, repositoryErrors.ErrReversalValueMismatch):
			handler.SendError(ctx, http.StatusUnprocessableEntity, err.Error())
			return
		case errors.Is(err, repositoryErrors.ErrInsufficientFunds):
			recordFailedOperationDbFunc(opType, req.AccountID, req.ReferenceID, req.Value, req.Currency, err)
			handler.SendError(ctx, http.StatusUnprocessableEntity, err.Error())
			return
		case errors.Is(err, repositoryErrors.ErrAccountNotActive),
			errors.Is(err, repositoryErrors.ErrInvalidCurrency):
			recordFailedOperationDbFunc(opType, req.AccountID, req.ReferenceID, req.Value, req.Currency, err)
			handler.SendError(ctx, http.StatusUnprocessableEntity, err.Error())
			return
		default:
			logger.Errorf("Operation %s failed after retries: %v", opType, err)
			handler.SendError(ctx, http.StatusInternalServerError, "Internal error processing operation")
			return
		}
	}

	switch opType {
	case "transfer":
		debit, credit := transferHistories[0], transferHistories[1]

		usedCredit := int64(0)
		if originAccount.AvailableBalance < 0 {
			usedCredit = -originAccount.AvailableBalance
		}

		resp := TransferOperationResponse{
			TransactionID:   debit.ReferenceID + "-PROCESSED",
			ReferenceID:     debit.ReferenceID,
			Status:          debit.Status,
			Type:            opType,
			OriginAccountID: originAccount.ID,
			DestAccountID:   destAccount.ID,
			OriginBalance:   originAccount.AvailableBalance + originAccount.ReservedBalance,
			DestBalance:     destAccount.AvailableBalance + destAccount.ReservedBalance,
			CreditUsed:      usedCredit,
			CreditAvailable: originAccount.CreditLimit - usedCredit,
			Timestamp:       debit.CreatedAt,
			ErrorMessage:    nil,
			DebitTxID:       debit.ID,
			CreditTxID:      credit.ID,
			TransferGroupID: debit.TransferGroupID,
		}
		handler.SendSuccess(ctx, "create-operation", "data", resp)
		return

	case "reversal":
		accountsResp := make([]ReversalAccountResult, 0, len(reversalAccounts))
		for _, acc := range reversalAccounts {
			accountsResp = append(accountsResp, ReversalAccountResult{
				AccountID:        acc.ID,
				Balance:          acc.AvailableBalance + acc.ReservedBalance,
				AvailableBalance: acc.AvailableBalance,
				ReservedBalance:  acc.ReservedBalance,
			})
		}

		txIDs := make([]uint64, 0, len(reversalHistories))
		for _, h := range reversalHistories {
			txIDs = append(txIDs, h.ID)
		}

		first := reversalHistories[0]

		resp := ReversalOperationResponse{
			TransactionID:   first.ReferenceID + "-PROCESSED",
			ReferenceID:     first.ReferenceID,
			Status:          first.Status,
			Type:            opType,
			Timestamp:       first.CreatedAt,
			ErrorMessage:    nil,
			OriginalRefID:   req.OriginalReferenceID,
			ReversalTxIDs:   txIDs,
			TransferGroupID: first.TransferGroupID,
			Accounts:        accountsResp,
		}
		handler.SendSuccess(ctx, "create-operation", "data", resp)
		return
	}

	usedCredit := int64(0)
	if account.AvailableBalance < 0 {
		usedCredit = -account.AvailableBalance
	}

	resp := OperationResponse{
		TransactionID:    history.ReferenceID + "-PROCESSED",
		ReferenceID:      history.ReferenceID,
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

func (r *OperationRequest) OperationValidate(opType string) error {
	r.ReferenceID = uuid.NewString()

	if opType == "reversal" {
		r.OriginalReferenceID = strings.TrimSpace(r.OriginalReferenceID)
		if r.OriginalReferenceID == "" {
			return handler.ErrParamIsRequired("original_reference_id", "string")
		}
		return nil
	}

	if r.Value <= 0 {
		return handler.ErrInvalidParam("value", "int64")
	}

	if r.AccountID == 0 {
		return handler.ErrParamIsRequired("account_id", "uint64")
	}

	if r.Currency != "" && len(r.Currency) != 3 {
		return handler.ErrInvalidParam("currency", "string")
	}

	if opType == "transfer" {
		if r.AccountDestID == 0 {
			return handler.ErrParamIsRequired("account_dest_id", "uint64")
		}
		if r.AccountDestID == r.AccountID {
			return handler.ErrInvalidParam("account_dest_id", "uint64")
		}
	}

	return nil
}
