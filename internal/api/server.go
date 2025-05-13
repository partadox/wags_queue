package api

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/partadox/wags_queue/internal/auth"
	"github.com/partadox/wags_queue/internal/config"
)

// Server represents the API server
type Server struct {
	server *http.Server
	router *mux.Router
	db     *sql.DB
	auth   *auth.Authenticator
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, db *sql.DB) *Server {
	router := mux.NewRouter()
	
	server := &Server{
		server: &http.Server{
			Addr:         ":" + cfg.Server.Port,
			Handler:      router,
			ReadTimeout:  cfg.Server.ReadTimeout,
			WriteTimeout: cfg.Server.WriteTimeout,
			IdleTimeout:  cfg.Server.IdleTimeout,
		},
		router: router,
		db:     db,
		auth:   auth.NewAuthenticator(db, cfg.Auth),
	}

	// Set up routes
	server.setupRoutes()
	
	return server
}

// setupRoutes configures the API routes
func (s *Server) setupRoutes() {
	// API routes
	api := s.router.PathPrefix("/api").Subrouter()
	
	// Auth routes (no authentication required)
	authRoutes := api.PathPrefix("/auth").Subrouter()
	authRoutes.HandleFunc("/login", s.handleLogin).Methods("POST")
	
	// Message routes (authentication required)
	messageRoutes := api.PathPrefix("/messages").Subrouter()
	messageRoutes.Use(s.auth.Middleware)
	messageRoutes.HandleFunc("/send", s.handleSendMessage).Methods("POST")
	messageRoutes.HandleFunc("/send-bulk", s.handleSendBulkMessage).Methods("POST")
	
	// UI data routes (authentication required)
	uiRoutes := api.PathPrefix("/ui").Subrouter()
	uiRoutes.Use(s.auth.Middleware)
	uiRoutes.HandleFunc("/messages", s.handleGetMessages).Methods("GET")
	uiRoutes.HandleFunc("/broadcasts", s.handleGetBroadcasts).Methods("GET")
	uiRoutes.HandleFunc("/broadcasts/{bulk_id}/details", s.handleGetBroadcastDetails).Methods("GET")
	uiRoutes.HandleFunc("/years", s.handleGetAvailableYears).Methods("GET")
	
	// Static files for UI
	s.router.PathPrefix("/").Handler(http.FileServer(http.Dir("./ui/static")))
}

// Start starts the API server
func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

// Stop gracefully stops the API server
func (s *Server) Stop(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	return s.server.Shutdown(ctx)
}
