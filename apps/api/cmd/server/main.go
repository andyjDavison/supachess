package main

import (
	"api/internal/transport"
	"api/internal/user"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func InitDB(connectionString string) (*sql.DB, error) {
	db, err := sql.Open("pgx", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)                 // Maximum open connections
	db.SetMaxIdleConns(25)                 // Maximum idle connections
	db.SetConnMaxLifetime(5 * time.Minute) // Reclaim old connections safely

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func RunMigrations(db *sql.DB, migrationsPath string) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	abs, err := filepath.Abs(migrationsPath)
	if err != nil {
		return fmt.Errorf("resolving migrations path: %w", err)
	}

	// Points to your local /migrations folder
	m, err := migrate.NewWithDatabaseInstance(
		"file://"+abs,
		"postgres", driver,
	)
	if err != nil {
		return err
	}

	// Applies all pending SQL updates
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}

func main() {
	connStr := getEnv("DATABASE_URL", "postgres://postgres:admin@localhost:5432/postgres?sslmode=disable")
	migrationsPath := getEnv("MIGRATIONS_PATH", "db/migrations")

	db, err := InitDB(connStr)
	if err != nil {
		log.Fatalf("Database initialization error: %v", err)
	}
	defer db.Close()

	if err := RunMigrations(db, migrationsPath); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	fmt.Println("Database migrations applied successfully!")

	userRepo := user.NewUserRepository(db)
	userService := user.NewUserService(userRepo, user.NewBcryptHasher(10))
	userHandler := transport.NewUserHandler(userService)
	
	http.HandleFunc("POST /api/register", userHandler.CreateUserHandler)
	http.HandleFunc("GET /api/user/{id}", userHandler.FindUserByIdHandler)

	fmt.Println("Server starting locally on http://localhost:8080...")
	
	// Start the server listener block
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}