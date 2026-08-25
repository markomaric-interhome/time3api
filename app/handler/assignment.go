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
 * Assign an apprentice user to a trainer user. It will automatically validate if the target
 * users have the proper roles for this operation.
 */
func (h *AssignmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	currUsr, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// decode the request body
	var req assignment.CreateAssignmentRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// get the target users
	trainerUsr, err := h.users.GetByID(r.Context(), req.TrainerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "trainer not found", http.StatusNotFound)
			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	apprenticeUsr, err := h.users.GetByID(r.Context(), req.ApprenticeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "apprentice not found", http.StatusNotFound)
			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	allowed, err := h.authz.CanAssignApprentice(r.Context(), currUsr, trainerUsr, apprenticeUsr)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if !allowed {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	ass, err := h.assigns.Assign(r.Context(), trainerUsr.ID, apprenticeUsr.ID)
	if err != nil {
		http.Error(w, "internal server error assign", http.StatusInternalServerError)
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
 * Delete an assignment between an apprentice and their trainer.
 */
func (h *AssignmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	currUsr, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// parse the target user id's from the url
	trainerId, err := strconv.ParseInt(r.PathValue("tID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid trainer id", http.StatusBadRequest)
		return
	}

	apprenticeId, err := strconv.ParseInt(r.PathValue("aID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid apprentice id", http.StatusBadRequest)
		return
	}

	allowed, err := h.authz.CanUnassignApprentice(r.Context(), currUsr)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if !allowed {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if err := h.assigns.Unassign(r.Context(), trainerId, apprenticeId); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
