package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"github.com/nezuchan/spotify-streamer/config"
	"github.com/sirupsen/logrus"
	librespot "github.com/devgianlu/go-librespot"
	"github.com/devgianlu/go-librespot/session"
	"github.com/devgianlu/go-librespot/player"
	devicespb "github.com/devgianlu/go-librespot/proto/spotify/connectstate/devices"
)

// LibrespotApp wraps go-librespot daemon functionality
type LibrespotApp struct {
	cfg        *config.Config
	logger     librespot.Logger
	stateStore *config.FileStateStore
	session    *session.Session
	player     *player.Player
	mu         sync.RWMutex
}

// NewLibrespotApp creates a new librespot application
func NewLibrespotApp(cfg *config.Config, logger *logrus.Logger) (*LibrespotApp, error) {
	app := &LibrespotApp{
		cfg:        cfg,
		logger:     config.NewLogrusAdapter(logger),
		stateStore: &config.FileStateStore{Path: cfg.Librespot.StatePath},
	}

	return app, nil
}

// Start initializes and starts the librespot session
func (app *LibrespotApp) Start(ctx context.Context) error {
	// Load state
	state, err := app.stateStore.Load()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	// Create credentials based on config
	var creds any
	
	switch app.cfg.Librespot.Credentials.Type {
	case "interactive":
		if len(state.Credentials.Data) > 0 && state.Credentials.Username != "" {
			// Use stored credentials if available
			creds = session.StoredCredentials{
				Username: state.Credentials.Username,
				Data:     state.Credentials.Data,
			}
		} else {
			// Use interactive login
			creds = session.InteractiveCredentials{
				CallbackPort: 36842, // Default callback port
			}
		}
	case "stored":
		if len(state.Credentials.Data) == 0 {
			return fmt.Errorf("no stored credentials available")
		}
		creds = session.StoredCredentials{
			Username: state.Credentials.Username,
			Data:     state.Credentials.Data,
		}
	case "username_password":
		if app.cfg.Librespot.Credentials.Username == "" || app.cfg.Librespot.Credentials.Password == "" {
			return fmt.Errorf("username and password required for username_password auth")
		}
		// Note: go-librespot may not support direct username/password
		return fmt.Errorf("username_password authentication not supported by go-librespot")
	default:
		return fmt.Errorf("unknown credentials type: %s", app.cfg.Librespot.Credentials.Type)
	}

	// Get device ID
	deviceID := state.DeviceId
	if deviceID == "" {
		deviceIDBytes := make([]byte, 20)
		_, _ = rand.Read(deviceIDBytes)
		deviceID = hex.EncodeToString(deviceIDBytes)
		app.logger.Infof("Generated new device ID: %s", deviceID)
	}

	// Parse device type
	deviceType, err := parseDeviceType(app.cfg.Librespot.DeviceType)
	if err != nil {
		return fmt.Errorf("invalid device type: %w", err)
	}

	// Create session
	sess, err := session.NewSessionFromOptions(ctx, &session.Options{
		Log:         app.logger,
		DeviceType:  deviceType,
		DeviceId:    deviceID,
		Credentials: creds,
	})
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	app.mu.Lock()
	app.session = sess
	app.mu.Unlock()

	app.logger.Infof("Session created for user: %s", sess.Username())

	// Save updated state
	storedCreds := sess.StoredCredentials()
	state.Credentials.Username = sess.Username()
	state.Credentials.Data = storedCreds
	state.DeviceId = deviceID
	if err := app.stateStore.Save(state); err != nil {
		app.logger.WithError(err).Warnf("Failed to save state")
	}

	// Create player
	plr, err := player.NewPlayer(&player.Options{
		Spclient:             sess.Spclient(),
		AudioKey:             sess.AudioKey(),
		Events:               sess.Events(),
		Log:                  app.logger,
		AudioBackend:         "pipe",
		AudioDevice:          "",
		ExternalVolume:       true,
		FlacEnabled:          false,
		NormalisationEnabled: app.cfg.Librespot.Audio.Normalisation,
		NormalisationPregain: float32(app.cfg.Librespot.Audio.NormalisationPregain),
	})
	if err != nil {
		return fmt.Errorf("failed to create player: %w", err)
	}

	app.mu.Lock()
	app.player = plr
	app.mu.Unlock()

	app.logger.Infof("Player initialized")

	return nil
}

// GetSession returns the current session
func (app *LibrespotApp) GetSession() *session.Session {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return app.session
}

// GetPlayer returns the current player
func (app *LibrespotApp) GetPlayer() *player.Player {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return app.player
}

// IsAuthenticated checks if the session is authenticated
func (app *LibrespotApp) IsAuthenticated() bool {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return app.session != nil
}

// GetUsername returns the authenticated username
func (app *LibrespotApp) GetUsername() string {
	app.mu.RLock()
	defer app.mu.RUnlock()
	if app.session != nil {
		return app.session.Username()
	}
	return ""
}

// Shutdown cleanly shuts down the application
func (app *LibrespotApp) Shutdown() {
	app.mu.Lock()
	defer app.mu.Unlock()

	if app.player != nil {
		app.player.Stop()
		app.player = nil
	}

	if app.session != nil {
		// Save state before closing
		if state, err := app.stateStore.Load(); err == nil {
			storedCreds := app.session.StoredCredentials()
			state.Credentials.Username = app.session.Username()
			state.Credentials.Data = storedCreds
			app.stateStore.Save(state)
		}
		app.session.Close()
		app.session = nil
	}
}

// parseDeviceType converts string device type to protobuf enum
func parseDeviceType(val string) (devicespb.DeviceType, error) {
	valEnum, ok := devicespb.DeviceType_value[strings.ToUpper(val)]
	if !ok {
		return 0, fmt.Errorf("invalid device type: %s", val)
	}
	return devicespb.DeviceType(valEnum), nil
}
