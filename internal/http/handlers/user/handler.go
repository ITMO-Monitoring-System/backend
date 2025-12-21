package user

import (
	"encoding/json"
	"monitoring_backend/internal/http/response"
	"net/http"
)

type UserService interface {
	AddUser(request AddUserRequest) error
	AddUserFaces(request AddUserFacesRequest) error
}

type UserHandler struct {
	userService UserService
}

func NewUserHandler(userService UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

func (h *UserHandler) AddUser(w http.ResponseWriter, r *http.Request) {
	var request AddUserRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
	}

	if err := h.userService.AddUser(request); err != nil {
		response.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.WriteJSON(w, http.StatusCreated, "ok")
}
