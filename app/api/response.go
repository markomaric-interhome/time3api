package api

import "net/http"

func WriteJSON(w http.ResponseWriter, status int, data any) error {
	return nil
}

func WriteError(w http.ResponseWriter, status int, message string) error {
	return nil
}
