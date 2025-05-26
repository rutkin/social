package main

import (
	"log"
	"net/http"
	"os"
	"social/internal/db"
	"social/internal/handlers"
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

	// Инициализируем соединения с базой данных
	db.InitDB(writeHost, writePort, readHost, readPort, dbUser, dbPassword, dbName)
	defer db.Close()

	// Создаем таблицы, если они не существуют
	db.CreateTables()

	log.Printf("Database configuration: Write: %s:%s, Read: %s:%s", writeHost, writePort, readHost, readPort)

	// Настраиваем HTTP маршруты
	mux := http.NewServeMux()
	mux.HandleFunc("/login", handlers.LoginHandler)
	mux.HandleFunc("/user/register", handlers.RegisterHandler)
	mux.HandleFunc("/user/get/", handlers.GetUserHandler)
	mux.HandleFunc("/user/search", handlers.SearchUsersHandler)
	mux.HandleFunc("GET /post/feed", handlers.PostFeedHandler)
	mux.HandleFunc("POST /dialog/{user_id}/send", handlers.SendMessageHandler)
	mux.HandleFunc("GET /dialog/{user_id}/list", handlers.GetDialogHandler)

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
