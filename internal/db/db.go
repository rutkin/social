package db

import (
	"database/sql"
	"log"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

// Три отдельных соединения: для записи, чтения и Citus
var (
	WriteDB *sql.DB
	ReadDB  *sql.DB
	CitusDB *sql.DB
)

// InitDB инициализирует соединения с базой данных для чтения, записи и Citus
func InitDB(writeHost, writePort, readHost, readPort, citusHost, citusPort, user, password, dbname string, workerHosts []string, workerPorts []string) {
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

	// Формируем строку подключения для Citus
	citusDataSourceName := buildDSN(citusHost, citusPort, user, password, dbname)
	CitusDB, err = sql.Open("postgres", citusDataSourceName)
	if err != nil {
		log.Fatalf("Failed to connect to Citus database: %v", err)
	}

	log.Printf("Connecting to Citus database with DSN: %s", citusDataSourceName)
	for retries := 0; retries < 5; retries++ {
		err = CitusDB.Ping()
		if err == nil {
			break
		}
		log.Printf("Retrying connection to Citus database (%d/5): %v", retries+1, err)
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		log.Fatalf("Failed to ping Citus database after retries: %v", err)
	}
	log.Printf("Connected to Citus database at %s:%s", citusHost, citusPort)

	_, err = CitusDB.Exec("SELECT citus_set_coordinator_host('citus-coordinator');")
	if err != nil {
		log.Fatalf("Failed to set Citus coordinator host: %v", err)
	}
	// Регистрация воркеров в координаторе
	for i := range workerHosts {
		query := `SELECT master_add_node($1, $2);`
		workerPort, err := strconv.Atoi(workerPorts[i])
		if err != nil {
			log.Fatalf("Failed to convert worker port %s to integer: %v", workerPorts[i], err)
		}
		_, err = CitusDB.Exec(query, workerHosts[i], workerPort)
		if err != nil {
			log.Fatalf("Failed to register worker %s:%d in coordinator: %v", workerHosts[i], workerPort, err)
		}
		log.Printf("Registered worker %s:%d in coordinator", workerHosts[i], workerPort)
	}

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
			id SERIAL,
			from_user_id UUID NOT NULL,
			to_user_id UUID NOT NULL,
			text TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			shard_key BIGINT NOT NULL
		);
	`
	_, err := CitusDB.Exec(query) // Create the table if it doesn't exist
	if err != nil {
		log.Printf("Failed to create messages table: %v", err)
		return err
	}

	// Check if the table is already distributed
	checkQuery := `SELECT logicalrelid FROM pg_dist_partition WHERE logicalrelid = 'messages'::regclass;`
	row := CitusDB.QueryRow(checkQuery)
	var logicalRelID string
	if err := row.Scan(&logicalRelID); err == sql.ErrNoRows {
		// Table is not distributed, proceed with distribution
		distributeQuery := `SELECT create_distributed_table('messages', 'shard_key');`
		_, err = CitusDB.Exec(distributeQuery)
		if err != nil {
			log.Printf("Failed to distribute messages table: %v", err)
			return err
		}
		log.Printf("Distributed messages table successfully")
	} else if err != nil {
		log.Printf("Failed to check distribution status of messages table: %v", err)
		return err
	} else {
		log.Printf("Messages table is already distributed")
	}

	return nil
}

// Close закрывает все соединения с базой данных
func Close() {
	if WriteDB != nil {
		WriteDB.Close()
	}
	if ReadDB != nil {
		ReadDB.Close()
	}
	if CitusDB != nil {
		CitusDB.Close()
	}
}
