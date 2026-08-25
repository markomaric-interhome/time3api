package auth

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"time3api/app/user"

	"github.com/go-playground/validator/v10"
)

type Handler struct {
	users    *user.Store
	validate *validator.Validate
	tokens   *TokenManager
}

func NewHandler(users *user.Store, tokens *TokenManager) *Handler {
	return &Handler{
		users: users,
		validate: validator.New(
			validator.WithRequiredStructEnabled(),
		),
		tokens: tokens,
	}
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	usr, ok := UserFromContext(r.Context())
	if !ok {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(usr.Response())
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie := http.Cookie{
		Name:     "time3auth",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0).UTC(),
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, &cookie)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req user.UserRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// normalize inputs
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Firstname = strings.TrimSpace(req.Firstname)
	req.Lastname = strings.TrimSpace(req.Lastname)
	req.Team = strings.TrimSpace(req.Team)

	// parse the start date
	var startDate *time.Time
	if req.StartDate != "" {
		t, err := time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			http.Error(w, "error parsing start date", http.StatusBadRequest)
			return
		}
		startDate = &t
	}

	// validate the schema
	if err := h.validate.Struct(req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Hash the password
	passwordHash, err := HashPassword(req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	u := &user.User{
		Team:         req.Team,
		Role:         user.RoleApprentice,
		Email:        req.Email,
		PasswordHash: passwordHash,
		Firstname:    req.Firstname,
		Lastname:     req.Lastname,
		StartDate:    startDate,
	}

	// store the user in the db
	usr, err := h.users.Create(r.Context(), u)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// pre-marshal the response
	data, err := json.Marshal(usr.Response())
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(data)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req user.LoginRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	// decode the request body
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// normalize inputs & validate schema
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if err := h.validate.Struct(req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// get the user by email
	usr, err := h.users.GetByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "invalid email or password", http.StatusUnauthorized)
			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// validate the password
	if err := CheckPassword(usr.PasswordHash, req.Password); err != nil {
		http.Error(w, "invalid email or password", http.StatusUnauthorized)
		return
	}

	// update the users last_login_at timestamp
	updatedUsr, err := h.users.UpdateLastLoginAt(r.Context(), usr.ID)
	if err != nil {
		log.Printf("update last login for user %d: %v", usr.ID, err)
	}

	// generate the JWT token & cookie
	token, err := h.tokens.Create(updatedUsr.ID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	cookie := http.Cookie{
		Name:     "time3auth",
		Value:    token,
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}

	// prepare the response and serialize it
	data, err := json.Marshal(updatedUsr.Response())
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &cookie)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
