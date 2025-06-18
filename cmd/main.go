package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"social/internal/db"
	"social/internal/handlers"
	"social/internal/rabbit"
	"social/internal/services"
	"social/internal/ws"
	"strconv"
)

func main() {
	// Получаем параметры подключения к базе данных из переменных окружения
	writeHost := getEnv("DB_WRITE_HOST", "localhost")
	writePort := getEnv("DB_WRITE_PORT", "5433") // Порт для записи в HAProxy
	readHost := getEnv("DB_READ_HOST", "localhost")
	readPort := getEnv("DB_READ_PORT", "5434") // Порт для чтения в HAProxy
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "social")

	citusHost := "citus-coordinator"
	citusPort := "5432"
	workerHosts := []string{"citus-worker", "citus-worker-2"}
	workerPorts := []string{"5432", "5432"}

	// Инициализируем соединения с базой данных
	db.InitDB(writeHost, writePort, readHost, readPort, citusHost, citusPort, dbUser, dbPassword, dbName, workerHosts, workerPorts)
	defer db.Close()

	// Создаем таблицы, если они не существуют
	db.CreateTables()

	log.Printf("Database configuration: Write: %s:%s, Read: %s:%s", writeHost, writePort, readHost, readPort)

	// Init RabbitMQ
	log.Printf("RABBITMQ_URL: %s", os.Getenv("RABBITMQ_URL"))
	if err := rabbit.InitRabbit(); err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbit.CloseRabbit()

	// Создаем очереди для шардов и запускаем воркеры
	const numShards = 8
	for shard := 0; shard < numShards; shard++ {
		queueName := "feed_shard_" + strconv.Itoa(shard)
		_, err := rabbit.RabbitChan.QueueDeclare(queueName, true, false, false, false, nil)
		if err != nil {
			log.Fatalf("Failed to declare queue %s: %v", queueName, err)
		}
		go func(qn string) {
			msgs, _ := rabbit.RabbitChan.Consume(qn, "", true, false, false, false, nil)
			for d := range msgs {
				var task struct {
					FriendIDs []string                 `json:"friend_ids"`
					Post      ws.PostFeedPostedMessage `json:"post"`
				}
				if err := json.Unmarshal(d.Body, &task); err == nil {
					ws.NotifyFriendsBatch(task.FriendIDs, task.Post)
				}
			}
		}(queueName)
	}

	// Initialize Redis
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}
	db.InitRedis(redisAddr)

	// Инициализация Lua-скриптов для сообщений
	if err := services.InitMessageScripts(); err != nil {
		log.Fatalf("Failed to load Redis Lua scripts: %v", err)
	}

	// Настраиваем HTTP маршруты
	mux := http.NewServeMux()
	mux.HandleFunc("/login", handlers.LoginHandler)
	mux.HandleFunc("/user/register", handlers.RegisterHandler)
	mux.HandleFunc("/user/get/", handlers.GetUserHandler)
	mux.HandleFunc("/user/search", handlers.SearchUsersHandler)
	mux.HandleFunc("GET /post/feed", handlers.PostFeedHandler)
	mux.HandleFunc("POST /post/create", handlers.CreatePostHandler)
	mux.HandleFunc("POST /dialog/{user_id}/send", handlers.SendMessageHandler)
	mux.HandleFunc("GET /dialog/{user_id}/list", handlers.GetDialogHandler)
	mux.HandleFunc("/ws", ws.ServeWS)

	log.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func getEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback
	}
	return value
}
