package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"time3api/app/auth"
	"time3api/app/authorization"
	"time3api/app/review"
	"time3api/app/user"

	"github.com/go-playground/validator/v10"
)

type ReviewHandler struct {
	reviews  *review.Store
	users    *user.Store
	authz    *authorization.Service
	validate *validator.Validate
}

func NewReviewHandler(reviews *review.Store, users *user.Store, authz *authorization.Service) *ReviewHandler {
	return &ReviewHandler{
		reviews: reviews,
		users:   users,
		authz:   authz,
	}
}

func (h *ReviewHandler) Create(w http.ResponseWriter, r *http.Request) {
	currUsr, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req review.CreateReviewRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// validate the schema
	if err := h.validate.Struct(req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// get the trainer user
	trainerUsr, err := h.users.GetByID(r.Context(), req.TrainerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "trainer not found", http.StatusNotFound)
			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// get the apprentice user
	apprenticeUsr, err := h.users.GetByID(r.Context(), req.ApprenticeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "apprentice not found", http.StatusNotFound)
			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// check if this constellation of "currUsr", "trainerUsr" & "apprenticeUsr" is allowed
	allowed, err := h.authz.CanCreateReview(r.Context(), currUsr, trainerUsr, apprenticeUsr)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if !allowed {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	rev, err := h.reviews.Create(r.Context(), &req)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(rev)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(data)
}

func (h *ReviewHandler) Delete(w http.ResponseWriter, r *http.Request) {
	currUsr, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

}
