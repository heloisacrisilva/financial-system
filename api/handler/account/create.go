package account

import (
	"financial/system/api/config"
	"financial/system/api/handler"
	repository "financial/system/api/repository/account"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

type CreateAccountRequest struct {
	Name  string `json:"name" binding:"required,min=3,max=100"`
	Email string `json:"email" binding:"required,email"`
	CPF   string `json:"cpf" binding:"required"`
}

func CreateAccountHandler(ctx *gin.Context) {
	logger := config.GetLogger()

	var request CreateAccountRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		logger.Errorf("Invalid JSON payload: %v", err)
		handler.SendError(ctx, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := request.CreateValidate(); err != nil {
		logger.Errorf("Validation failed: %v", err)
		handler.SendError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	account, err := repository.CreateAccountRepository(request.Name, request.Email, request.CPF)
	if err != nil {
		logger.Errorf("Error creating account: %v", err)

		if err.Error() == "This email is already in use by another user." {
			handler.SendError(ctx, http.StatusConflict, err.Error())
			return
		}

		handler.SendError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	logger.Infof("Account created successfully for email: %s", request.Email)
	handler.SendSuccess(ctx, "create-account", "account", account)

}

func (r *CreateAccountRequest) CreateValidate() error {
	var emailRegex = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)

	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return handler.ErrParamIsRequired("name", "string")
	}
	if len(r.Name) < 3 {
		return handler.ErrInvalidParam("name", "string")
	}

	r.Email = strings.TrimSpace(strings.ToLower(r.Email))
	if r.Email == "" {
		return handler.ErrParamIsRequired("email", "string")
	}
	if !emailRegex.MatchString(r.Email) {
		return handler.ErrInvalidParam("email", "string")
	}

	cleanCPF := strings.NewReplacer(".", "", "-", "", " ", "").Replace(r.CPF)
	if cleanCPF == "" {
		return handler.ErrParamIsRequired("CPF", "string")
	}
	if len(cleanCPF) != 11 {
		return handler.ErrInvalidParam("CPF", "string")
	}
	r.CPF = cleanCPF

	return nil
}
