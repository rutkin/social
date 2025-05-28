package services

import (
	"crypto/sha1"
	"encoding/binary"
	"math"
	"social/internal/db"
	"social/internal/models"
	"sort"
	"time"
)

func SendMessage(fromUserID, toUserID, text string) error {
	db := db.CitusDB // Use the Citus coordinator for sharded messages
	query := `
		INSERT INTO messages (from_user_id, to_user_id, text, created_at, shard_key)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := db.Exec(query, fromUserID, toUserID, text, time.Now(), calcShardKey(fromUserID, toUserID))
	return err
}

func GetDialog(userID1, userID2 string) ([]models.Message, error) {
	db := db.CitusDB // Use the Citus coordinator for sharded messages
	query := `
		SELECT from_user_id, to_user_id, text, created_at
		FROM messages
		WHERE shard_key = $1 AND
		((from_user_id = $2 AND to_user_id = $3) OR (from_user_id = $3 AND to_user_id = $2))
		ORDER BY created_at ASC
	`
	rows, err := db.Query(query, calcShardKey(userID1, userID2), userID1, userID2)
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

func calcShardKey(fromID, toID string) int64 {
	pair := []string{fromID, toID}
	sort.Strings(pair)

	// Хешируем объединённую строку
	joined := pair[0] + pair[1]
	hash := sha1.Sum([]byte(joined))

	// Используем первые 8 байт как uint64 и делаем его положительным int64
	val := int64(binary.BigEndian.Uint64(hash[:8]) & math.MaxInt64)

	return val
}
