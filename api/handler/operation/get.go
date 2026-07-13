package operation

import (
	"errors"
	"financial/system/api/entities"
	"financial/system/api/handler"
	repository "financial/system/api/repository/operation"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type OperationHistoryResponse struct {
	ID              uint64         `json:"id"`
	AccountID       uint64         `json:"account_id"`
	ReferenceID     string         `json:"reference_id"`
	TransferGroupID *string        `json:"transfer_group_id"`
	RelatedTxID     *uint64        `json:"related_tx_id"`
	Type            string         `json:"type"`
	Direction       *string        `json:"direction"`
	Value           int64          `json:"value"`
	Currency        string         `json:"currency"`
	Status          string         `json:"status"`
	ErrorMessage    *string        `json:"error_message"`
	Metadata        datatypes.JSON `json:"metadata"`
	CreatedAt       time.Time      `json:"created_at"`
}

func GetOperationByID(ctx *gin.Context) {
	operationIDParam := strings.TrimSpace(ctx.Param("id"))
	if operationIDParam == "" {
		handler.SendError(ctx, http.StatusBadRequest, handler.ErrParamIsRequired("ID", "routeParameter").Error())
		return
	}

	operationID, err := strconv.ParseUint(operationIDParam, 10, 64)
	if err != nil || operationID == 0 {
		handler.SendError(ctx, http.StatusBadRequest, handler.ErrInvalidParam("ID", "routeParameter").Error())
		return
	}

	history, err := repository.GetOperationHistoryByID(operationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handler.SendError(ctx, http.StatusNotFound, "Operation history not found.")
			return
		}

		handler.SendError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	handler.SendSuccess(ctx, "get-operation-history", "history", newOperationHistoryResponse(history))
}

func newOperationHistoryResponse(history entities.TransactionHistory) OperationHistoryResponse {
	return OperationHistoryResponse{
		ID:              history.ID,
		AccountID:       history.AccountID,
		ReferenceID:     history.ReferenceID,
		TransferGroupID: history.TransferGroupID,
		RelatedTxID:     history.RelatedTxID,
		Type:            history.Type,
		Direction:       history.Direction,
		Value:           history.Value,
		Currency:        history.Currency,
		Status:          history.Status,
		ErrorMessage:    history.ErrorMessage,
		Metadata:        history.Metadata,
		CreatedAt:       history.CreatedAt,
	}
}
