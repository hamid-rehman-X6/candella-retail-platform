// Package response provides a consistent JSON envelope for every handler
// across every domain, so API consumers (the frontend, future mobile
// clients, partner integrations) always parse the same shape.
package response

import (
	"encoding/json"
	"net/http"
)

type successEnvelope[T any] struct {
	Success bool `json:"success"`
	Data    T    `json:"data"`
}

type errorEnvelope struct {
	Success bool      `json:"success"`
	Error   errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func JSON[T any](w http.ResponseWriter, status int, data T) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(successEnvelope[T]{Success: true, Data: data})
}

func Error(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorEnvelope{
		Success: false,
		Error:   errorBody{Code: code, Message: message},
	})
}
