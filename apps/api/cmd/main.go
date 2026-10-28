package main

import (
	"api/internal/auth"
	"api/internal/friends"
	"api/internal/game"
	"api/internal/invite"
	"api/internal/matchmaking"
	"api/internal/server"
	"api/internal/transport/web"
	"api/internal/transport/ws"
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
	frontendOrigin := getEnv("FRONTEND_ORIGIN", "http://localhost:5173")
	migrationsPath := getEnv("MIGRATIONS_PATH", "../db/migrations")
	addr := getEnv("ADDR", ":8080")
	secret := []byte(getEnv("JWT_SECRET", ""))
	ttl := 60 * time.Minute
	secure, secErr := strconv.ParseBool(getEnv("IS_PRODUCTION", "false"))
	ws.ConfigureAllowedOrigins(frontendOrigin)

	if secErr != nil {
		secure = false
	}

	// An empty key still signs and verifies tokens, so nothing fails loudly
	// on its own - refuse to start rather than ship forgeable sessions.
	if secure && len(secret) == 0 {
		log.Fatal("JWT_SECRET must be set when IS_PRODUCTION=true")
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
	userHandler := web.NewRegisterHandler(userService)

	authService := auth.NewAuthService(userRepo, user.NewBcryptHasher(10))
	authHandler := web.NewAuthHandler(authService, secret, ttl, secure)

	hub := ws.NewHub()
	eventPublisher := ws.NewGameEventPublisher(hub)

	gameRepo := game.NewPostgresGameRepository(db)
	moveValidator := game.NewNotnilMoveValidator()
	gameService := game.NewGameService(gameRepo, moveValidator, eventPublisher, nil)
	timerManager := game.NewTimerManager(gameService)
	gameService.SetTimers(timerManager)

	matchmakingNotifier := ws.NewMatchmakingNotifier(hub)
	matchmakingService := matchmaking.NewMatchmakingService(gameService, userService, matchmakingNotifier)

	friendRepo := friends.NewPostgresFriendRepository(db)
	friendService := friends.NewFriendService(friendRepo)
	friendsHandler := web.NewFriendsHandler(friendService)

	// Invite state lives in memory, so no waiting game from before this
	// start can ever be accepted - clear them before serving.
	discarded, err := gameService.DiscardAllWaitingGames(context.Background())
	if err != nil {
		log.Fatalf("Discarding stale invites: %v", err)
	}
	if discarded > 0 {
		fmt.Printf("Discarded %d stale invite game(s)\n", discarded)
	}
	inviteNotifier := ws.NewInviteNotifier(hub)
	inviteService := invite.NewInviteService(gameService, userService, friendService, hub, inviteNotifier)

	hub.OnDisconnect = func(userID string) {
		matchmakingService.HandleDisconnect(userID)
		inviteService.HandleDisconnect(userID)
	}
	
	gameMessageHandler := ws.NewGameMessageHandler(gameService, matchmakingService, inviteService, hub)
	gameHandler := web.NewGameHandler(gameService)

	sweepCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go matchmakingService.RunSweepLoop(sweepCtx, 2*time.Second)
	
	router := server.NewRouter(secret, userHandler, authHandler, gameHandler, friendsHandler, gameMessageHandler, hub)
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