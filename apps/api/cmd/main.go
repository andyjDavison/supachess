package main

import (
	"api/internal/auth"
	"api/internal/server"
	"api/internal/transport"
	"api/internal/user"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
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

	// abs, err := filepath.Abs(migrationsPath)
	// if err != nil {
	// 	return fmt.Errorf("resolving migrations path: %w", err)
	// }

	// Points to your local /migrations folder
	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
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
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, relying on real environment variables")
	}

	connStr := getEnv("DATABASE_URL", "postgres://postgres:admin@localhost:5432/postgres?sslmode=disable")
	migrationsPath := getEnv("MIGRATIONS_PATH", "../db/migrations")
	addr := getEnv("ADDR", ":8080")
	secret := []byte(getEnv("SECRET", ""))
	ttl := 15 * time.Minute
	secure, secErr := strconv.ParseBool(getEnv("IS_PRODUCTION", "false"))

	if secErr != nil {
		secure = false
	}

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
	userHandler := transport.NewRegisterHandler(userService)

	authService := auth.NewAuthService(userRepo, user.NewBcryptHasher(10))
	authHandler := auth.NewAuthHandler(authService, secret, ttl, secure)
	
	router := server.NewRouter(secret,userHandler, authHandler)
	srv := server.New(addr, router)
 
	if err := srv.Run(10 * time.Second); err != nil {
		log.Fatalf("Server error: %v", err)
	}
	fmt.Println("Server stopped gracefully")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}