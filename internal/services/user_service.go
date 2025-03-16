package services

import (
	"social/internal/db"
	"social/internal/errors"
	"social/internal/models"
	"social/internal/utils"
)

func RegisterUser(user *models.User) (string, error) {
	hashedPassword, err := utils.HashPassword(user.Password)
	if err != nil {
		return "", err
	}
	user.Password = hashedPassword
	var userID string
	err = db.DB.QueryRow("INSERT INTO users (id, first_name, last_name, birthdate, biography, city, password) VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6) RETURNING id",
		user.FirstName, user.LastName, user.Birthdate, user.Biography, user.City, user.Password).Scan(&userID)
	if err != nil {
		return "", err
	}
	return userID, nil
}

func LoginUser(credentials *models.Credentials) (string, error) {
	var storedPassword string
	err := db.DB.QueryRow("SELECT password FROM users WHERE id = $1", credentials.ID).Scan(&storedPassword)
	if err != nil {
		return "", errors.ErrUserNotFound
	}
	if !utils.CheckPasswordHash(credentials.Password, storedPassword) {
		return "", errors.ErrInvalidCredentials
	}
	// Generate token (for simplicity, using user ID as token)
	return credentials.ID, nil
}

func GetUserByID(id string) (*models.User, error) {
	var user models.User
	err := db.DB.QueryRow("SELECT id, first_name, last_name, birthdate, biography, city FROM users WHERE id = $1", id).Scan(
		&user.ID, &user.FirstName, &user.LastName, &user.Birthdate, &user.Biography, &user.City)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
