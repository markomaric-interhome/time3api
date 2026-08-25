package auth

import (
	"database/sql"
	"errors"
	"net/http"
)

func (h *Handler) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// get the `time3auth` cookie
		cookie, err := r.Cookie("time3auth")
		if err != nil {
			if errors.Is(err, http.ErrNoCookie) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		// verify the token to be valid
		userID, err := h.tokens.Verify(cookie.Value)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// get the user from the database
		usr, err := h.users.GetByID(r.Context(), userID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		// make sure the user is not deleted
		if usr.DeletedAt != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := withUser(r.Context(), usr)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
