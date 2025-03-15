package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"social/internal/db"
	"social/internal/handlers"
)

func main() {
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "social")

	dataSourceName := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db.InitDB(dataSourceName)
	db.CreateTables()

	mux := http.NewServeMux()
	mux.HandleFunc("/login", handlers.LoginHandler)
	mux.HandleFunc("/user/register", handlers.RegisterHandler)
	mux.HandleFunc("/user/get/", handlers.GetUserHandler)

	log.Fatal(http.ListenAndServe(":8080", mux))
}

func getEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback
	}
	return value
}
