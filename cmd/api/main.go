package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"time3api/app/assignment"
	"time3api/app/auth"
	"time3api/app/authorization"
	"time3api/app/database"
	"time3api/app/handler"
	"time3api/app/user"

	"github.com/joho/godotenv"
)

func main() {
	ctx := context.Background()

	// 1. load the local environment variables from the `.env` file
	// ------------------------------------------------------------
	if err := godotenv.Load(); err != nil {
		log.Printf("no .env file found, using environment variables...")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is mandatory")
	}

	// 2. setup the database, run migrations and establish a connection to it
	// ----------------------------------------------------------------------
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./data/app.db"
	}

	// make sure the folder for the database exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		log.Fatalf("create database directory: %v", err)
	}

	// open the database connection
	db, err := database.Open(ctx, dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}

	defer db.Close()

	if err := database.Migrate(ctx, db); err != nil {
		log.Fatalf("migrate database: %v", err)
	}

	log.Printf("database ready: %s", dbPath)

	// 3. create the necessary database stores, handlers, managers, services, etc.
	// ---------------------------------------------------------------------------
	assignmentStore := assignment.NewStore(db)
	userStore := user.NewStore(db)

	tokenManager := auth.NewTokenManager(jwtSecret, "time3api", 8*time.Hour)

	authzService := authorization.NewService(assignmentStore, userStore)

	// handlers
	authHandler := auth.NewHandler(userStore, tokenManager)
	userHandler := handler.NewUserHandler(userStore, assignmentStore, authzService)
	assignmentHandler := handler.NewAssignmentHandler(assignmentStore, userStore, authzService)

	// 4. setup the routes
	// -------------------
	mux := http.NewServeMux()

	// authentication
	mux.HandleFunc("POST /auth/register", authHandler.Register)
	mux.HandleFunc("POST /auth/login", authHandler.Login)
	mux.HandleFunc("DELETE /auth/logout", authHandler.Logout)
	mux.Handle("GET /auth/me", authHandler.Authenticate(http.HandlerFunc(authHandler.Me)))

	// user
	mux.Handle("GET /user", authHandler.Authenticate(http.HandlerFunc(userHandler.List)))
	mux.Handle("GET /user/{id}", authHandler.Authenticate(http.HandlerFunc(userHandler.Details)))
	mux.Handle("DELETE /user/{id}", authHandler.Authenticate(authorization.RequireRole(user.RoleAdmin)(http.HandlerFunc(userHandler.Delete))))
	mux.Handle("PATCH /user/{id}", authHandler.Authenticate(http.HandlerFunc(userHandler.Update)))
	mux.Handle("PATCH /user/{id}/role", authHandler.Authenticate(authorization.RequireRole(user.RoleAdmin)(http.HandlerFunc(userHandler.UpdateRole))))

	// assignment
	mux.Handle("POST /assignments", authHandler.Authenticate(authorization.RequireRole(user.RoleAdmin)(http.HandlerFunc(assignmentHandler.Create))))
	mux.Handle("DELETE /assignments/{tID}/{aID}", authHandler.Authenticate(authorization.RequireRole(user.RoleAdmin)(http.HandlerFunc(assignmentHandler.Delete))))

	// 5. start the server and listen for incoming requests
	// ----------------------------------------------------
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Fatal(server.ListenAndServe())
}
