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

/**
 * Get the details of a user. This will return all data from the user + append a list of either trainers
 * or apprentices, depending on the role of the user.
 */
func (h *UserHandler) Details(w http.ResponseWriter, r *http.Request) {
	currUsr, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// parse the target user's ID from the url
	targetId, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	// get the user from the database by it's ID
	usr, err := h.users.GetByID(r.Context(), targetId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// check if the current user has permissions to retrieve user data
	allowed, err := h.authz.CanAccessUser(r.Context(), currUsr, usr)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if !allowed {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	details := user.UserDetailsResponse{
		UserResponse: usr.Response(),
	}

	// append the list of "trainers" or "apprentices" based on the user's role
	switch usr.Role {
	case user.RoleAdmin, user.RoleTrainer:
		apprentices, err := h.assigns.ApprenticesForTrainer(r.Context(), usr.ID)
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
		trainers, err := h.assigns.TrainersForApprentice(r.Context(), usr.ID)
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

	data, err := json.Marshal(details)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

/**
 * ADMIN ONLY HANDLER
 *
 * Delete a user. This will set the "deleted_at" timestamp which will prevent the user
 * from being able to authenticate, without actually deleting all their information from the DB.
 */
func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// parse the target users ID from the url
	targetId, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	// delete the user
	_, err = h.users.Delete(r.Context(), targetId)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

/**
 * Update an user. This will set the "updated_at" timestamp.
 */
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	currUsr, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// parse the target user's ID from the url
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	// get the target user
	usr, err := h.users.GetByID(r.Context(), id)
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

	// permission check
	allowed, err := h.authz.CanUpdateUser(r.Context(), currUsr, usr)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if !allowed {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// update the user
	usr, err = h.users.Update(r.Context(), usr.ID, req)
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

/**
 * ADMIN ONLY HANDLER
 *
 * Update the "role" of a user.
 */
func (h *UserHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	// parse the target users ID from the url
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	// decode the request body
	var req user.UpdateUserRoleRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// change the user's role
	usr, err := h.users.UpdateRole(r.Context(), id, req.Role)
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

/**
 * Get a list of all the users in the application.
 */
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	usrs, err := h.users.GetAll(r.Context())
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	summaries := make([]user.UserSummary, 0, len(usrs))
	for _, usr := range usrs {
		summaries = append(summaries, usr.Summary())
	}

	data, err := json.Marshal(summaries)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

