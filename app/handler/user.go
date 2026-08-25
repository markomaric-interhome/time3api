package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"time3api/app/assignment"
	"time3api/app/auth"
	"time3api/app/authorization"
	"time3api/app/user"
)

type UserHandler struct {
	users   *user.Store
	assigns *assignment.Store
	authz   *authorization.Service
}

func NewUserHandler(users *user.Store, assigns *assignment.Store, authz *authorization.Service) *UserHandler {
	return &UserHandler{
		users:   users,
		assigns: assigns,
		authz:   authz,
	}
}

func (h *UserHandler) Details(w http.ResponseWriter, r *http.Request) {
	currUsr, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// parse the target user's id from the url
	targetId, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	// get the user using the targetId
	targetUsr, err := h.users.GetByID(r.Context(), targetId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	allowed, err := h.authz.CanAccessUser(r.Context(), currUsr, targetUsr)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if !allowed {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	details := user.UserDetailsResponse{
		UserResponse: targetUsr.Response(),
	}

	switch targetUsr.Role {
	case user.RoleAdmin, user.RoleTrainer:
		apprentices, err := h.assigns.ApprenticesForTrainer(r.Context(), targetUsr.ID)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		summaries := make([]user.UserSummary, 0, len(apprentices))
		for _, apprentice := range apprentices {
			summaries = append(summaries, apprentice.Summary())
		}

		details.Apprentices = summaries

	case user.RoleApprentice:
		trainers, err := h.assigns.TrainersForApprentice(r.Context(), targetUsr.ID)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		summaries := make([]user.UserSummary, 0, len(trainers))
		for _, trainer := range trainers {
			summaries = append(summaries, trainer.Summary())
		}

		details.Trainers = summaries
	}

	// marshal the data
	data, err := json.Marshal(details)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	currUsr, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// parse the target user's id from the url
	targetId, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	allowed, err := h.authz.CanDeleteUser(r.Context(), currUsr)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if !allowed {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// delete the user
	usr, err := h.users.Delete(r.Context(), targetId)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(usr)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	currUsr, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// parse the target user's id from the url
	targetId, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	// get the target users
	targetUsr, err := h.users.GetByID(r.Context(), targetId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// decode the request body
	var req user.UpdateUserRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// check if there are any fields to update
	if req.Firstname == nil && req.Lastname == nil {
		http.Error(w, "no fields to update", http.StatusBadRequest)
		return
	}

	// check if the current user can make this change
	allowed, err := h.authz.CanUpdateUser(r.Context(), currUsr, targetUsr)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if !allowed {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	usr, err := h.users.Update(r.Context(), targetUsr.ID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(usr.Response())
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (h *UserHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	currUsr, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// parse the target user's id from the url
	targetId, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	// decode the new role from the payload
	var req user.UpdateUserRoleRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// check if the current user can make this change
	allowed, err := h.authz.CanUpdateUserRole(r.Context(), currUsr)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if !allowed {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// apply the update
	usr, err := h.users.UpdateRole(r.Context(), targetId, req.Role)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(usr.Response())
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
