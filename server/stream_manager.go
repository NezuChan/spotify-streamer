package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/nezuchan/spotify-streamer/config"
	"github.com/sirupsen/logrus"
)

// StreamData holds cached audio stream data
type StreamData struct {
	buffer    *bytes.Buffer
	ready     chan struct{}
	err       error
	refCount  int
	createdAt time.Time
	mu        sync.RWMutex
}

// StreamManager manages track downloads and caching
type StreamManager struct {
	app     *LibrespotApp
	cfg     *config.Config
	logger  *logrus.Logger
	streams map[string]*StreamData
	mu      sync.RWMutex
}

// NewStreamManager creates a new stream manager
func NewStreamManager(app *LibrespotApp, cfg *config.Config, logger *logrus.Logger) *StreamManager {
	sm := &StreamManager{
		app:     app,
		cfg:     cfg,
		logger:  logger,
		streams: make(map[string]*StreamData),
	}

	// Start cleanup goroutine if caching is enabled
	if cfg.Cache.Enabled {
		go sm.cleanupLoop()
	}

	return sm
}

// GetStream gets or creates a stream for a track
func (sm *StreamManager) GetStream(ctx context.Context, trackID string) (*StreamData, error) {
	// Check if stream exists
	sm.mu.RLock()
	if stream, exists := sm.streams[trackID]; exists {
		sm.mu.RUnlock()
		sm.logger.WithField("track_id", trackID).Debug("Stream found in cache")
		
		// Wait for stream to be ready
		select {
		case <-stream.ready:
			if stream.err != nil {
				return nil, stream.err
			}
			stream.mu.Lock()
			stream.refCount++
			stream.mu.Unlock()
			return stream, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	sm.mu.RUnlock()

	// Create new stream
	sm.mu.Lock()
	
	// Double-check after acquiring write lock
	if stream, exists := sm.streams[trackID]; exists {
		sm.mu.Unlock()
		select {
		case <-stream.ready:
			if stream.err != nil {
				return nil, stream.err
			}
			stream.mu.Lock()
			stream.refCount++
			stream.mu.Unlock()
			return stream, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	sm.logger.WithField("track_id", trackID).Info("Creating new stream")
	
	stream := &StreamData{
		buffer:    new(bytes.Buffer),
		ready:     make(chan struct{}),
		createdAt: time.Now(),
	}
	
	if sm.cfg.Cache.Enabled {
		sm.streams[trackID] = stream
	}
	
	sm.mu.Unlock()

	// Download track in background
	go sm.downloadTrack(ctx, trackID, stream)

	// Wait for it to be ready
	select {
	case <-stream.ready:
		if stream.err != nil {
			return nil, stream.err
		}
		stream.mu.Lock()
		stream.refCount++
		stream.mu.Unlock()
		return stream, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// downloadTrack downloads a track from Spotify
func (sm *StreamManager) downloadTrack(ctx context.Context, trackID string, stream *StreamData) {
	defer close(stream.ready)

	logger := sm.logger.WithField("track_id", trackID)
	logger.Info("Starting track download")

	// Get session from app
	session := sm.app.GetSession()
	if session == nil {
		stream.err = fmt.Errorf("not authenticated")
		logger.Error("Session not available")
		return
	}

	// Get player
	player := sm.app.GetPlayer()
	if player == nil {
		stream.err = fmt.Errorf("player not initialized")
		logger.Error("Player not available")
		return
	}

	// Create Spotify URI
	_ = fmt.Sprintf("spotify:track:%s", trackID)
	
	// TODO: Implement actual audio streaming
	// The go-librespot player API needs to be used to:
	// 1. Load the track using player methods
	// 2. Capture audio output (OGG Vorbis format)
	// 3. Write to stream.buffer
	//
	// This requires deeper integration with go-librespot's player architecture
	// which is designed for live playback, not downloading
	
	stream.err = fmt.Errorf("audio streaming not yet implemented - requires go-librespot player integration")
	logger.Error("Track download not implemented - this is a framework implementation")
	
	// TODO: Implement actual streaming logic:
	// 1. Use session.Spclient() to load track metadata
	// 2. Use player to create audio stream
	// 3. Read OGG Vorbis data and write to stream.buffer
	// 4. Handle errors and cleanup
}

// WriteTo writes the stream data to an io.Writer
func (sd *StreamData) WriteTo(w io.Writer) (int64, error) {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	reader := bytes.NewReader(sd.buffer.Bytes())
	return io.Copy(w, reader)
}

// Release decrements the reference count
func (sd *StreamData) Release() {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	if sd.refCount > 0 {
		sd.refCount--
	}
}

// cleanupLoop periodically removes old cached streams
func (sm *StreamManager) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		sm.cleanup()
	}
}

// cleanup removes expired streams from cache
func (sm *StreamManager) cleanup() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	ttl := time.Duration(sm.cfg.Cache.TTL) * time.Second

	for trackID, stream := range sm.streams {
		stream.mu.RLock()
		expired := now.Sub(stream.createdAt) > ttl && stream.refCount == 0
		stream.mu.RUnlock()

		if expired {
			sm.logger.WithField("track_id", trackID).Debug("Removing expired stream from cache")
			delete(sm.streams, trackID)
		}
	}

	// Also check if we have too many cached streams
	if len(sm.streams) > sm.cfg.Cache.MaxTracks {
		sm.logger.WithField("count", len(sm.streams)).Warn("Cache limit exceeded, pruning oldest entries")
		// TODO: Implement LRU eviction
	}
}
