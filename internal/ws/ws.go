package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	UserID string
	Conn   *websocket.Conn
}

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	clients      = make(map[string]map[*Client]struct{}) // userID -> set of clients
	clientsMutex sync.RWMutex
)

func ServeWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}
	userID := token // For now, token is userID

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	client := &Client{UserID: userID, Conn: conn}

	clientsMutex.Lock()
	if clients[userID] == nil {
		clients[userID] = make(map[*Client]struct{})
	}
	clients[userID][client] = struct{}{}
	clientsMutex.Unlock()

	log.Printf("WebSocket connected: user %s", userID)

	go func() {
		defer func() {
			conn.Close()
			clientsMutex.Lock()
			delete(clients[userID], client)
			if len(clients[userID]) == 0 {
				delete(clients, userID)
			}
			clientsMutex.Unlock()
			log.Printf("WebSocket disconnected: user %s", userID)
		}()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

// PostFeedPostedMessage is the payload for /post/feed/posted
type PostFeedPostedMessage struct {
	PostID       string `json:"postId"`
	PostText     string `json:"postText"`
	AuthorUserID string `json:"author_user_id"`
}

// NotifyFriends sends a post event to all friends' websocket clients
func NotifyFriends(friendIDs []string, post PostFeedPostedMessage) {
	msg, _ := json.Marshal(post)
	clientsMutex.RLock()
	defer clientsMutex.RUnlock()
	for _, fid := range friendIDs {
		for c := range clients[fid] {
			go func(cl *Client) {
				cl.Conn.WriteMessage(websocket.TextMessage, msg)
			}(c)
		}
	}
}

// NotifyFriendsBatch sends a post event to a batch of friends' websocket clients
func NotifyFriendsBatch(friendIDs []string, post PostFeedPostedMessage) {
	msg, _ := json.Marshal(post)
	var targets []*Client
	clientsMutex.RLock()
	for _, fid := range friendIDs {
		for c := range clients[fid] {
			targets = append(targets, c)
		}
	}
	clientsMutex.RUnlock()
	for _, cl := range targets {
		go func(cl *Client) {
			cl.Conn.WriteMessage(websocket.TextMessage, msg)
		}(cl)
	}
}
