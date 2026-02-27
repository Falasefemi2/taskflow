package utils

import (
	"encoding/json"
	"log"
	"net/http"
)

type successResponse struct {
	Data any `json:"data"`
}

type errorResponse struct {
	Error  string `json:"error"`
	Fields any    `json:"fields,omitempty"`
}

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(successResponse{Data: data}); err != nil {
		log.Printf("failed to encode JSON response: %v", err)
	}
}

func Error(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(errorResponse{Error: message}); err != nil {
		log.Printf("failed to encode error response: %v", err)
	}
}

func ValidationError(w http.ResponseWriter, fields any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)

	if err := json.NewEncoder(w).Encode(errorResponse{
		Error:  "validation failed",
		Fields: fields,
	}); err != nil {
		log.Printf("failed to encode validation error response: %v", err)
	}
}
