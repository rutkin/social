package services

import (
	"social/internal/db"
	"social/internal/models"
	"time"
)

func SendMessage(fromUserID, toUserID, text string) error {
	query := `
		INSERT INTO messages (from_user_id, to_user_id, text, created_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := db.WriteDB.Exec(query, fromUserID, toUserID, text, time.Now())
	return err
}

func GetDialog(userID1, userID2 string) ([]models.Message, error) {
	query := `
		SELECT from_user_id, to_user_id, text, created_at
		FROM messages
		WHERE (from_user_id = $1 AND to_user_id = $2) OR (from_user_id = $2 AND to_user_id = $1)
		ORDER BY created_at ASC
	`
	rows, err := db.ReadDB.Query(query, userID1, userID2)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var message models.Message
		err := rows.Scan(&message.FromUserID, &message.ToUserID, &message.Text, &message.CreatedAt)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}

	return messages, nil
}
