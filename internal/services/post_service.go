package services

import (
	"context"
	"encoding/json"
	"fmt"
	"social/internal/db"
	"social/internal/models"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	redisClient = redis.NewClient(&redis.Options{
		Addr: "redis:6379", // Redis host and port
	})
)

const (
	cacheKeyPrefix = "friend_posts:"
	cacheTTL       = 5 * time.Minute // Cache expiration time
	cacheLimit     = 1000            // Maximum number of posts to keep in cache
)

func GetFriendPosts(ctx context.Context, userID string, offset, limit int) ([]models.Post, error) {
	cacheKey := fmt.Sprintf("%s%s", cacheKeyPrefix, userID)

	// Check if posts are cached
	cachedPosts, err := redisClient.LRange(ctx, cacheKey, 0, -1).Result()
	if err == nil && len(cachedPosts) > 0 {
		var posts []models.Post
		for _, postJSON := range cachedPosts {
			var post models.Post
			if err := json.Unmarshal([]byte(postJSON), &post); err == nil {
				posts = append(posts, post)
			}
		}
		// Apply offset and limit to the cached posts
		start := offset
		end := offset + limit
		if start > len(posts) {
			return []models.Post{}, nil
		}
		if end > len(posts) {
			end = len(posts)
		}
		return posts[start:end], nil
	}

	// Fetch posts from the database
	query := `
		SELECT posts.id, posts.text, posts.created_at, posts.author_user_id
		FROM posts
		JOIN friends ON posts.author_user_id = friends.friend_id
		WHERE friends.user_id = $1
		ORDER BY posts.created_at DESC
		OFFSET $2 LIMIT $3
	`
	rows, err := db.ReadDB.Query(query, userID, offset, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var post models.Post
		err := rows.Scan(&post.ID, &post.Text, &post.CreatedAt, &post.AuthorUserID)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	// Cache the posts
	for _, post := range posts {
		postJSON, err := json.Marshal(post)
		if err == nil {
			redisClient.LPush(ctx, cacheKey, postJSON)
		}
	}
	// Trim the cache to the last 1000 entries
	redisClient.LTrim(ctx, cacheKey, 0, cacheLimit-1)
	redisClient.Expire(ctx, cacheKey, cacheTTL)

	return posts, nil
}

func CreatePost(userID, text string) (string, error) {
	var postID string
	err := db.WriteDB.QueryRow(
		"INSERT INTO posts (author_user_id, text) VALUES ($1, $2) RETURNING id",
		userID, text,
	).Scan(&postID)
	if err != nil {
		return "", err
	}
	return postID, nil
}
