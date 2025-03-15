package handlers

import (
	"encoding/json"
	"net/http"
	"social/internal/errors"
	"social/internal/models"
	"social/internal/services"
	"strings"

	"github.com/google/uuid"
)

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var user models.User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	userID, err := services.RegisterUser(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"user_id": userID})
	w.WriteHeader(http.StatusOK)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var credentials models.Credentials
	err := json.NewDecoder(r.Body).Decode(&credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Validate that ID is a UUID and password is not empty
	if _, err := uuid.Parse(credentials.ID); err != nil {
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}
	if credentials.Password == "" {
		http.Error(w, "Password cannot be empty", http.StatusBadRequest)
		return
	}
	token, err := services.LoginUser(&credentials)
	if err != nil {
		switch err {
		case errors.ErrInvalidCredentials:
			http.Error(w, err.Error(), http.StatusUnauthorized)
		case errors.ErrUserNotFound:
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func GetUserHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/user/get/")
	user, err := services.GetUserByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(user)
}
