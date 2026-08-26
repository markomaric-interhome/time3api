package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"time3api/app/assignment"
	"time3api/app/authorization"
	"time3api/app/user"
)

type AssignmentHandler struct {
	assigns *assignment.Store
	users   *user.Store
	authz   *authorization.Service
}

func NewAssignmentHandler(assigns *assignment.Store, users *user.Store, authz *authorization.Service) *AssignmentHandler {
	return &AssignmentHandler{
		assigns: assigns,
		users:   users,
		authz:   authz,
	}
}

/**
 * Assign an apprentice to a trainer. It will automatically validate if both target users
 * have the required roles for this operation to succeed.
 */
func (h *AssignmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req assignment.CreateAssignmentRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// get the trainer
	trainer, err := h.users.GetByID(r.Context(), req.TrainerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "trainer not found", http.StatusNotFound)
			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// get the apprentice
	apprentice, err := h.users.GetByID(r.Context(), req.ApprenticeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "apprentice not found", http.StatusNotFound)
			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// check if both "trainer" and "apprentice" have the proper roles
	if trainer.Role != user.RoleTrainer && trainer.Role != user.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if apprentice.Role != user.RoleApprentice {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	ass, err := h.assigns.Assign(r.Context(), trainer.ID, apprentice.ID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(ass)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(data)
}

/**
 * Unassign an apprentice from a trainer.
 */
func (h *AssignmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tID, err := strconv.ParseInt(r.PathValue("tID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid trainer id", http.StatusBadRequest)
		return
	}

	aID, err := strconv.ParseInt(r.PathValue("aID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid apprentice id", http.StatusBadRequest)
		return
	}

	if err := h.assigns.Unassign(r.Context(), tID, aID); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
