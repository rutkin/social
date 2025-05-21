package services

import (
	"social/internal/db"
	"social/internal/models"
)

func GetFriendPosts(userID string, offset, limit int) ([]models.Post, error) {
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

	return posts, nil
}
