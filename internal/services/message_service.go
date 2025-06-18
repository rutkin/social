package services

import (
	"context"
	"encoding/json"
	"fmt"
	"social/internal/db"
	"social/internal/models"
	"time"
)

var (
	sendMessageSha string
	getDialogSha   string
)

func InitMessageScripts() error {
	ctx := context.Background()

	sendMessageScript := `
		redis.call("LPUSH", "dialog:" .. KEYS[1] .. ":" .. KEYS[2], ARGV[1])
		redis.call("LPUSH", "dialog:" .. KEYS[2] .. ":" .. KEYS[1], ARGV[1])
		return "OK"
	`
	sha, err := db.RedisClient.ScriptLoad(ctx, sendMessageScript).Result()
	if err != nil {
		return fmt.Errorf("failed to load SendMessage script: %w", err)
	}
	sendMessageSha = sha

	getDialogScript := `
		local messages = redis.call("LRANGE", "dialog:" .. KEYS[1] .. ":" .. KEYS[2], 0, -1)
		return cjson.encode(messages)
	`
	sha, err = db.RedisClient.ScriptLoad(ctx, getDialogScript).Result()
	if err != nil {
		return fmt.Errorf("failed to load GetDialog script: %w", err)
	}
	getDialogSha = sha

	return nil
}

// SendMessage stores the message in Redis using a Redis UDF.
func SendMessage(fromUserID, toUserID, text string) error {
	ctx := context.Background()
	message := models.Message{
		FromUserID: fromUserID,
		ToUserID:   toUserID,
		Text:       text,
		CreatedAt:  time.Now(),
	}
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Use EVALSHA to call the Lua script
	_, err = db.RedisClient.EvalSha(ctx, sendMessageSha, []string{fromUserID, toUserID}, messageJSON).Result()
	if err != nil {
		return fmt.Errorf("failed to execute Redis Lua script: %w", err)
	}
	return nil
}

// GetDialog retrieves the dialog messages from Redis using a Redis UDF.
func GetDialog(userID1, userID2 string) ([]models.Message, error) {
	ctx := context.Background()
	result, err := db.RedisClient.EvalSha(ctx, getDialogSha, []string{userID1, userID2}).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to execute Redis Lua script: %w", err)
	}

	// result.(string) is a JSON array of strings, each string is a JSON message
	var rawMessages []string
	if err := json.Unmarshal([]byte(result.(string)), &rawMessages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dialog array: %w", err)
	}

	var messages []models.Message
	for _, raw := range rawMessages {
		var msg models.Message
		if err := json.Unmarshal([]byte(raw), &msg); err == nil {
			messages = append(messages, msg)
		}
	}
	return messages, nil
}
