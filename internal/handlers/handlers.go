package handlers

import (
	"encoding/json"
	"log"
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
	log.Printf("GetUserHandler: received request for user ID: %s", id)

	user, err := services.GetUserByID(id)
	if err != nil {
		log.Printf("GetUserHandler: error getting user by ID %s: %v", id, err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	log.Printf("GetUserHandler: successfully retrieved user with ID: %s", id)
	json.NewEncoder(w).Encode(user)
}

func SearchUsersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Printf("SearchUsersHandler: invalid method %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	firstName := r.URL.Query().Get("first_name")
	lastName := r.URL.Query().Get("last_name")

	log.Printf("SearchUsersHandler: received search request for first_name='%s', last_name='%s'", firstName, lastName)

	if firstName == "" || lastName == "" {
		log.Printf("SearchUsersHandler: missing required parameters. first_name='%s', last_name='%s'", firstName, lastName)
		http.Error(w, "Both first_name and last_name parameters are required", http.StatusBadRequest)
		return
	}

	users, err := services.SearchUsers(firstName, lastName)
	if err != nil {
		log.Printf("SearchUsersHandler: error searching users with first_name='%s', last_name='%s': %v", firstName, lastName, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("SearchUsersHandler: found %d users for first_name='%s', last_name='%s'", len(users), firstName, lastName)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}
