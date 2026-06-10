package server

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/nezuchan/spotify-streamer/config"
	"github.com/sirupsen/logrus"
	librespot "github.com/devgianlu/go-librespot"
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

	// Parse track ID to SpotifyId
	spotifyId, err := librespot.SpotifyIdFromBase62(librespot.SpotifyIdTypeTrack, trackID)
	if err != nil {
		stream.err = fmt.Errorf("invalid track ID: %w", err)
		logger.WithError(err).Error("Failed to parse track ID")
		return
	}

	logger.Debug("Creating player stream")
	
	// Create HTTP client for player
	httpClient := &http.Client{Timeout: 30 * time.Second}
	
	// Create player stream - this loads the track and prepares it for playback
	playerStream, err := player.NewStream(ctx, httpClient, *spotifyId, 320, 0)
	if err != nil {
		stream.err = fmt.Errorf("failed to create player stream: %w", err)
		logger.WithError(err).Error("Failed to create player stream")
		return
	}

	logger.WithFields(logrus.Fields{
		"media_name":   playerStream.Media.Name(),
		"is_track":     playerStream.Media.IsTrack(),
		"file_format":  playerStream.File.Format,
	}).Info("Player stream created")

	// Read audio data from the source
	// AudioSource provides Read([]float32) which gives us raw PCM samples
	source := playerStream.Source
	
	// We'll read in chunks and convert to 16-bit PCM
	const chunkSize = 4096 // samples per channel
	samples := make([]float32, chunkSize*2) // stereo
	pcmBuf := make([]byte, chunkSize*2*2) // 16-bit stereo
	
	totalSamples := 0
	for {
		select {
		case <-ctx.Done():
			stream.err = ctx.Err()
			logger.Warn("Download cancelled")
			return
		default:
		}
		
		// Read samples from source
		n, err := source.Read(samples)
		if err != nil {
			if err == io.EOF {
				logger.WithField("total_samples", totalSamples).Info("Track download complete")
				break
			}
			stream.err = fmt.Errorf("failed to read audio: %w", err)
			logger.WithError(err).Error("Failed to read audio samples")
			return
		}
		
		if n == 0 {
			break
		}
		
		totalSamples += n
		
		// Convert float32 samples to 16-bit PCM
		for i := 0; i < n; i++ {
			sample := int16(samples[i] * 32767)
			binary.LittleEndian.PutUint16(pcmBuf[i*2:], uint16(sample))
		}
		
		// Write to buffer
		if _, err := stream.buffer.Write(pcmBuf[:n*2]); err != nil {
			stream.err = fmt.Errorf("failed to write to buffer: %w", err)
			logger.WithError(err).Error("Failed to write to buffer")
			return
		}
		
		// Log progress every 100k samples (~2 seconds of audio)
		if totalSamples%100000 == 0 {
			logger.WithField("samples", totalSamples).Debug("Download progress")
		}
	}
	
	logger.WithFields(logrus.Fields{
		"total_samples": totalSamples,
		"buffer_size": stream.buffer.Len(),
		"duration_seconds": totalSamples / 44100 / 2, // 44.1kHz stereo
	}).Info("Track downloaded successfully")
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
