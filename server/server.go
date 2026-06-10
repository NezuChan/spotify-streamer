package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/nezuchan/spotify-streamer/config"
	"github.com/nezuchan/spotify-streamer/internal/models"
	"github.com/sirupsen/logrus"
)

// Server represents the HTTP server
type Server struct {
	app     *LibrespotApp
	streams *StreamManager
	cfg     *config.Config
	logger  *logrus.Logger
	router  *mux.Router
	server  *http.Server
}

// New creates a new server instance
func New(logger *logrus.Logger, cfg *config.Config) (*Server, error) {
	// Create librespot app
	app, err := NewLibrespotApp(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create librespot app: %w", err)
	}

	// Create server
	s := &Server{
		app:    app,
		cfg:    cfg,
		logger: logger,
		router: mux.NewRouter(),
	}

	// Create stream manager
	s.streams = NewStreamManager(app, cfg, logger)

	// Setup routes
	s.setupRoutes()

	return s, nil
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	// Health check
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")

	// Track endpoints
	s.router.HandleFunc("/track/{id}", s.handleTrackMetadata).Methods("GET")
	s.router.HandleFunc("/stream/{id}", s.handleStream).Methods("GET")

	// Search endpoint
	s.router.HandleFunc("/search", s.handleSearch).Methods("GET")

	// Album endpoint
	s.router.HandleFunc("/album/{id}", s.handleAlbum).Methods("GET")

	// Playlist endpoint
	s.router.HandleFunc("/playlist/{id}", s.handlePlaylist).Methods("GET")

	// Add CORS middleware
	s.router.Use(s.corsMiddleware)
	s.router.Use(s.loggingMiddleware)
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	// Start librespot app
	if err := s.app.Start(ctx); err != nil {
		return fmt.Errorf("failed to start librespot app: %w", err)
	}

	// Create HTTP server
	s.server = &http.Server{
		Addr:         s.cfg.Server.Host + ":" + s.cfg.Server.Port,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP server
	s.logger.WithField("address", s.server.Addr).Info("Starting HTTP server")
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if s.server != nil {
		s.logger.Info("Shutting down HTTP server")
		if err := s.server.Shutdown(ctx); err != nil {
			s.logger.WithError(err).Error("Failed to shutdown HTTP server gracefully")
		}
	}

	s.logger.Info("Shutting down librespot app")
	s.app.Shutdown()
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := models.HealthResponse{
		Status:        "ok",
		Authenticated: s.app.IsAuthenticated(),
		Username:      s.app.GetUsername(),
	}

	s.writeJSON(w, http.StatusOK, response)
}

// handleTrackMetadata handles track metadata requests
func (s *Server) handleTrackMetadata(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	trackID := vars["id"]

	logger := s.logger.WithField("track_id", trackID)
	logger.Debug("Fetching track metadata")

	// Check authentication
	if !s.app.IsAuthenticated() {
		s.writeError(w, http.StatusUnauthorized, "not authenticated", nil)
		return
	}

	session := s.app.GetSession()
	if session == nil {
		s.writeError(w, http.StatusServiceUnavailable, "session not available", nil)
		return
	}

	// TODO: Implement track metadata fetching using Spotify Web API
	// For now, return basic information
	response := models.TrackResponse{
		ID:       trackID,
		Title:    "Track " + trackID,
		Artist:   "Unknown Artist",
		Artists:  []string{"Unknown Artist"},
		Album:    "Unknown Album",
		Duration: 0,
		URI:      fmt.Sprintf("spotify:track:%s", trackID),
		URL:      fmt.Sprintf("https://open.spotify.com/track/%s", trackID),
	}

	s.writeJSON(w, http.StatusOK, response)
}

// handleStream handles audio stream requests
func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	trackID := vars["id"]

	logger := s.logger.WithField("track_id", trackID)
	logger.Info("Stream requested")

	// Check authentication
	if !s.app.IsAuthenticated() {
		s.writeError(w, http.StatusUnauthorized, "not authenticated", nil)
		return
	}

	// Get or create stream
	stream, err := s.streams.GetStream(r.Context(), trackID)
	if err != nil {
		logger.WithError(err).Error("Failed to get stream")
		s.writeError(w, http.StatusInternalServerError, "failed to get stream", err)
		return
	}
	defer stream.Release()

	// Set headers
	w.Header().Set("Content-Type", "audio/ogg")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("Accept-Ranges", "bytes")

	// Stream the data
	logger.Debug("Streaming audio data")
	if _, err := stream.WriteTo(w); err != nil {
		logger.WithError(err).Error("Failed to write stream data")
	}
}

// handleSearch handles search requests
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		s.writeError(w, http.StatusBadRequest, "missing query parameter", nil)
		return
	}

	logger := s.logger.WithField("query", query)
	logger.Debug("Search requested")

	// Check authentication
	if !s.app.IsAuthenticated() {
		s.writeError(w, http.StatusUnauthorized, "not authenticated", nil)
		return
	}

	session := s.app.GetSession()
	if session == nil {
		s.writeError(w, http.StatusServiceUnavailable, "session not available", nil)
		return
	}

	// TODO: Implement search using spclient
	// This is a placeholder response
	response := models.SearchResponse{
		Query:   query,
		Results: []models.TrackResponse{},
	}

	s.writeJSON(w, http.StatusOK, response)
}

// handleAlbum handles album requests
func (s *Server) handleAlbum(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	albumID := vars["id"]

	logger := s.logger.WithField("album_id", albumID)
	logger.Debug("Album requested")

	// Check authentication
	if !s.app.IsAuthenticated() {
		s.writeError(w, http.StatusUnauthorized, "not authenticated", nil)
		return
	}

	// TODO: Implement album fetching using spclient
	s.writeError(w, http.StatusNotImplemented, "album endpoint not yet implemented", nil)
}

// handlePlaylist handles playlist requests
func (s *Server) handlePlaylist(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playlistID := vars["id"]

	logger := s.logger.WithField("playlist_id", playlistID)
	logger.Debug("Playlist requested")

	// Check authentication
	if !s.app.IsAuthenticated() {
		s.writeError(w, http.StatusUnauthorized, "not authenticated", nil)
		return
	}

	// TODO: Implement playlist fetching using spclient
	s.writeError(w, http.StatusNotImplemented, "playlist endpoint not yet implemented", nil)
}

// writeJSON writes a JSON response
func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.WithError(err).Error("Failed to encode JSON response")
	}
}

// writeError writes an error response
func (s *Server) writeError(w http.ResponseWriter, status int, message string, err error) {
	response := models.ErrorResponse{
		Error:   message,
		Message: "",
	}
	if err != nil {
		response.Message = err.Error()
	}
	s.writeJSON(w, status, response)
}

// corsMiddleware adds CORS headers
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		s.logger.WithFields(logrus.Fields{
			"method":   r.Method,
			"path":     r.URL.Path,
			"duration": time.Since(start),
			"remote":   r.RemoteAddr,
		}).Debug("HTTP request")
	})
}
