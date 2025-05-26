package db

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

// Два отдельных соединения для чтения и записи
var (
	WriteDB *sql.DB
	ReadDB  *sql.DB
)

// InitDB инициализирует соединения с базой данных для чтения и записи
func InitDB(writeHost, writePort, readHost, readPort, user, password, dbname string) {
	// Формируем строку подключения для операций записи
	writeDataSourceName := buildDSN(writeHost, writePort, user, password, dbname)
	var err error
	WriteDB, err = sql.Open("postgres", writeDataSourceName)
	if err != nil {
		log.Fatalf("Failed to connect to write database: %v", err)
	}

	if err = WriteDB.Ping(); err != nil {
		log.Fatalf("Failed to ping write database: %v", err)
	}
	log.Printf("Connected to write database at %s:%s", writeHost, writePort)

	// Формируем строку подключения для операций чтения
	readDataSourceName := buildDSN(readHost, readPort, user, password, dbname)
	ReadDB, err = sql.Open("postgres", readDataSourceName)
	if err != nil {
		log.Fatalf("Failed to connect to read database: %v", err)
	}

	if err = ReadDB.Ping(); err != nil {
		log.Fatalf("Failed to ping read database: %v", err)
	}
	log.Printf("Connected to read database at %s:%s", readHost, readPort)

	// Создаем таблицу сообщений
	if err := createMessagesTable(); err != nil {
		log.Fatalf("Failed to create messages table: %v", err)
	}
}

// buildDSN формирует строку подключения к PostgreSQL
func buildDSN(host, port, user, password, dbname string) string {
	return "host=" + host +
		" port=" + port +
		" user=" + user +
		" password=" + password +
		" dbname=" + dbname +
		" sslmode=disable"
}

// CreateTables создает необходимые таблицы в базе данных
func CreateTables() {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		first_name TEXT,
		last_name TEXT,
		birthdate DATE,
		biography TEXT,
		city TEXT,
		password TEXT
	);

	CREATE TABLE IF NOT EXISTS posts (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		author_user_id UUID REFERENCES users(id),
		text TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS friends (
		user_id UUID REFERENCES users(id),
		friend_id UUID REFERENCES users(id),
		PRIMARY KEY (user_id, friend_id)
	);
	`
	// Используем WriteDB для создания таблиц
	_, err := WriteDB.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
}

// createMessagesTable создает таблицу сообщений
func createMessagesTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS messages (
			id SERIAL PRIMARY KEY,
			from_user_id UUID NOT NULL,
			to_user_id UUID NOT NULL,
			text TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL
		)
	`
	_, err := WriteDB.Exec(query)
	if err != nil {
		log.Printf("Failed to create messages table: %v", err)
	}
	return err
}

// Close закрывает все соединения с базой данных
func Close() {
	if WriteDB != nil {
		WriteDB.Close()
	}
	if ReadDB != nil {
		ReadDB.Close()
	}
}
