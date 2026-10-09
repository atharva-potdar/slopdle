package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"moodleplusplus/internal/db"
	"moodleplusplus/internal/router"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("MARIADB_DSN")
	}
	if dsn == "" {
		// Default to local Unix socket auth on MariaDB for user 'atharva-potdar'
		dsn = "atharva-potdar@unix(/run/mysqld/mysqld.sock)/test?parseTime=true"
	}

	log.Printf("Connecting to MariaDB: %s", dsn)
	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to open MariaDB connection: %v", err)
	}
	defer sqlDB.Close()

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Failed to ping MariaDB: %v", err)
	}
	log.Println("Connected to MariaDB successfully.")

	queries := db.New(sqlDB)
	r := router.NewRouter(queries, sqlDB)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Printf("\n🚀 Moodle++ Go Backend API running on http://localhost:%s\n\n", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
