package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"social/internal/errors"
	"social/internal/models"
	"social/internal/rabbit"
	"social/internal/services"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
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

func PostFeedHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("User-Id")
	if userID == "" {
		http.Error(w, "User-Id header is required", http.StatusBadRequest)
		return
	}

	offset, limit := 0, 10
	if val := r.URL.Query().Get("offset"); val != "" {
		offset, _ = strconv.Atoi(val)
	}
	if val := r.URL.Query().Get("limit"); val != "" {
		limit, _ = strconv.Atoi(val)
	}

	posts, err := services.GetFriendPosts(r.Context(), userID, offset, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}

func SendMessageHandler(w http.ResponseWriter, r *http.Request) {
	fromUserID := r.Header.Get("User-Id")
	// Debug log to help trace header value
	log.Printf("SendMessageHandler: User-Id header value: '%s'", fromUserID)
	if fromUserID == "" {
		log.Printf("SendMessageHandler: missing User-Id header")
		http.Error(w, "User-Id header is required", http.StatusBadRequest)
		return
	}

	toUserID := r.PathValue("user_id")
	if toUserID == "" {
		log.Printf("SendMessageHandler: missing recipient user_id in path")
		http.Error(w, "Recipient user ID is required", http.StatusBadRequest)
		return
	}

	var payload struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("SendMessageHandler: failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if payload.Text == "" {
		log.Printf("SendMessageHandler: empty message text")
		http.Error(w, "Message text cannot be empty", http.StatusBadRequest)
		return
	}

	err := services.SendMessage(fromUserID, toUserID, payload.Text)
	if err != nil {
		log.Printf("SendMessageHandler: failed to send message from %s to %s: %v", fromUserID, toUserID, err)
		http.Error(w, "Failed to send message", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func GetDialogHandler(w http.ResponseWriter, r *http.Request) {
	userID1 := r.Header.Get("User-Id")
	// Debug log to help trace header value
	log.Printf("GetDialogHandler: User-Id header value: '%s'", userID1)
	if userID1 == "" {
		log.Printf("GetDialogHandler: missing User-Id header")
		http.Error(w, "User-Id header is required", http.StatusBadRequest)
		return
	}

	userID2 := r.PathValue("user_id")
	if userID2 == "" {
		log.Printf("GetDialogHandler: missing user_id in path")
		http.Error(w, "Other user ID is required", http.StatusBadRequest)
		return
	}

	messages, err := services.GetDialog(userID1, userID2)
	if err != nil {
		log.Printf("GetDialogHandler: failed to retrieve dialog between %s and %s: %v", userID1, userID2, err)
		http.Error(w, "Failed to retrieve dialog", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func CreatePostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID := r.Header.Get("User-Id")
	if userID == "" {
		http.Error(w, "User-Id header is required", http.StatusUnauthorized)
		return
	}
	var payload struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if payload.Text == "" {
		http.Error(w, "Text is required", http.StatusBadRequest)
		return
	}
	postID, err := services.CreatePost(userID, payload.Text)
	if err != nil {
		http.Error(w, "Failed to create post", http.StatusInternalServerError)
		return
	}
	// Шардинг: публикуем задачи в очереди feed_shard_{N}
	go func() {
		friendIDs, _ := services.GetFriendIDs(userID)
		const batchSize = 100
		const numShards = 8
		for i := 0; i < len(friendIDs); i += batchSize {
			end := i + batchSize
			if end > len(friendIDs) {
				end = len(friendIDs)
			}
			batch := friendIDs[i:end]
			// Группируем по шардам
			shardBatches := make(map[int][]string)
			for _, fid := range batch {
				shard := int(hashString(fid)) % numShards
				shardBatches[shard] = append(shardBatches[shard], fid)
			}
			for shard, shardFriendIDs := range shardBatches {
				task := struct {
					FriendIDs []string `json:"friend_ids"`
					Post      struct {
						PostID       string `json:"postId"`
						PostText     string `json:"postText"`
						AuthorUserID string `json:"author_user_id"`
					} `json:"post"`
				}{
					FriendIDs: shardFriendIDs,
					Post: struct {
						PostID       string `json:"postId"`
						PostText     string `json:"postText"`
						AuthorUserID string `json:"author_user_id"`
					}{
						PostID:       postID,
						PostText:     payload.Text,
						AuthorUserID: userID,
					},
				}
				body, _ := json.Marshal(task)
				queueName := "feed_shard_" + strconv.Itoa(shard)
				rabbit.RabbitChan.Publish(
					"",        // default exchange
					queueName, // routing key = имя очереди
					false, false,
					amqp.Publishing{
						ContentType: "application/json",
						Body:        body,
					},
				)
			}
		}
	}()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": postID})
}

// hashString - простая хеш-функция для строк (userID)
func hashString(s string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h = (h * 16777619) ^ uint32(s[i])
	}
	return h
}
