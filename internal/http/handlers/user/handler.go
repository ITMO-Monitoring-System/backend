package user

import (
	"context"
	"encoding/json"
	"monitoring_backend/internal/http/response"
	"net/http"
)

type UserService interface {
	AddUser(ctx context.Context, request AddUserRequest) error
	AddUserFaces(ctx context.Context, request AddUserFacesRequest) error
}

type UserHandler struct {
	userService UserService
}

func NewUserHandler(userService UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// AddUser godoc
// @Summary      Добавление нового пользователя
// @Description  Создаёт нового пользователя с ISU, именем, фамилией и факультативным отчеством.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        user  body      AddUserRequest  true  "Пользователь для добавления"
// @Success      201   {string}  string               "ok"
// @Failure      400   {object}  response.ErrorResponse      "Некорректный JSON или обязательные поля отсутствуют"
// @Failure      500   {object}  response.ErrorResponse      "Ошибка сервиса при добавлении пользователя"
// @Router       /api/user/create [post]
func (h *UserHandler) AddUser(w http.ResponseWriter, r *http.Request) {
	var request AddUserRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
	}

	if err := h.userService.AddUser(r.Context(), request); err != nil {
		response.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.WriteJSON(w, http.StatusCreated, "ok")
}
