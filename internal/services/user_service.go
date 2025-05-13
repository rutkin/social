package services

import (
	"fmt"
	"social/internal/db"
	"social/internal/errors"
	"social/internal/models"
	"social/internal/utils"

	"github.com/google/uuid"
)

func RegisterUser(user *models.User) (string, error) {
	hashedPassword, err := utils.HashPassword(user.Password)
	if err != nil {
		return "", err
	}
	user.Password = hashedPassword
	var userID string
	err = db.WriteDB.QueryRow("INSERT INTO users (id, first_name, last_name, birthdate, biography, city, password) VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6) RETURNING id",
		user.FirstName, user.LastName, user.Birthdate, user.Biography, user.City, user.Password).Scan(&userID)
	if err != nil {
		return "", err
	}
	return userID, nil
}

func LoginUser(credentials *models.Credentials) (string, error) {
	var storedPassword string
	err := db.WriteDB.QueryRow("SELECT password FROM users WHERE id = $1", credentials.ID).Scan(&storedPassword)
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
	_, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid UUID format: %w", err)
	}

	var user models.User
	err = db.ReadDB.QueryRow("SELECT id, first_name, last_name, birthdate, biography, city FROM users WHERE id = $1", id).Scan(
		&user.ID, &user.FirstName, &user.LastName, &user.Birthdate, &user.Biography, &user.City)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func SearchUsers(firstNamePrefix, lastNamePrefix string) ([]models.User, error) {
	query := `
		SELECT id, first_name, last_name, birthdate, biography, city 
		FROM users 
		WHERE first_name LIKE $1 AND last_name LIKE $2 
		ORDER BY id
	`
	rows, err := db.ReadDB.Query(query, firstNamePrefix+"%", lastNamePrefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.ID, &user.FirstName, &user.LastName, &user.Birthdate, &user.Biography, &user.City)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
