# Spotify Streamer

REST API service for streaming Spotify tracks using go-librespot.

## Overview

This service provides a REST API for downloading and streaming Spotify tracks as OGG Vorbis audio. It uses [go-librespot](https://github.com/devgianlu/go-librespot) as a library to interact with Spotify's API and handle audio streaming.

## Features

- 🎵 Stream Spotify tracks as WAV (PCM audio)
- 📊 Get track metadata (title, artist, album, duration, artwork) via Spotify Web API
- 🔍 Search for tracks (coming soon)
- 💿 Get album and playlist information (coming soon)
- 💾 In-memory caching for frequently played tracks
- 🔐 Interactive authentication with Spotify
- 🎛️ Configurable audio quality (96, 160, 320 kbps)

## Requirements

- Go 1.25 or higher
- Spotify Premium account (required for audio streaming)
- Native libraries (automatically handled by go-librespot):
  - libogg
  - libvorbis
  - libflac

## Installation

### From Source

```bash
git clone https://github.com/nezuchan/spotify-streamer.git
cd spotify-streamer
go build -o spotify-streamer ./cmd/server
```

### Using Docker

```bash
docker pull ghcr.io/nezuchan/spotify-streamer:latest
```

Or with docker-compose:

```bash
# Create config.yaml from example
cp config.example.yaml config.yaml
# Edit config.yaml with your settings

# Start the service
docker-compose up -d
```

### Pre-built Binaries

Download pre-built binaries from the [Releases](https://github.com/NezuChan/spotify-streamer/releases) page.

## Configuration

Create a `config.yaml` file in the same directory as the binary:

```yaml
server:
  port: "8080"
  host: "0.0.0.0"

librespot:
  device_name: "Spotify-Streamer"
  device_type: "computer"
  credentials:
    type: "interactive"
  audio:
    bitrate: 320
    normalisation: true
    normalisation_pregain: 0
  state_path: "./state.json"

cache:
  enabled: true
  max_tracks: 100
  ttl: 3600

log_level: "info"
```

### Configuration Options

#### Server
- `port`: HTTP server port (default: "8080")
- `host`: HTTP server host (default: "0.0.0.0")

#### Librespot
- `device_name`: Name of the Spotify device (default: "Spotify-Streamer")
- `device_type`: Device type (default: "computer")
- `credentials.type`: Authentication method ("interactive" or "stored")
- `audio.bitrate`: Audio quality in kbps (96, 160, or 320)
- `audio.normalisation`: Enable audio normalization (default: true)
- `audio.normalisation_pregain`: Pregain in dB (-5 for quiet, 0 for normal, +3 for loud)
- `state_path`: Path to store authentication state (default: "./state.json")

#### Cache
- `enabled`: Enable in-memory track caching (default: true)
- `max_tracks`: Maximum number of tracks to cache (default: 100)
- `ttl`: Time-to-live for cached tracks in seconds (default: 3600)

#### Logging
- `log_level`: Log level (trace, debug, info, warn, error)

## Usage

### Starting the Service

```bash
./spotify-streamer -config config.yaml
```

On first run with `interactive` authentication, you'll need to:
1. Open the provided URL in your browser
2. Log in to your Spotify account
3. Authorize the application
4. The service will save your credentials to `state.json` for future use

### API Endpoints

#### Health Check
```http
GET /health
```

Response:
```json
{
  "status": "ok",
  "authenticated": true,
  "username": "your_spotify_username"
}
```

#### Get Track Metadata
```http
GET /track/{track_id}
```

Example:
```bash
curl http://localhost:8080/track/3n3Ppam7vgaVa1iaRUc9Lp
```

Response:
```json
{
  "id": "3n3Ppam7vgaVa1iaRUc9Lp",
  "title": "Mr. Brightside",
  "artist": "The Killers",
  "artists": ["The Killers"],
  "album": "Hot Fuss",
  "duration": 222000,
  "uri": "spotify:track:3n3Ppam7vgaVa1iaRUc9Lp",
  "url": "https://open.spotify.com/track/3n3Ppam7vgaVa1iaRUc9Lp",
  "artwork": "..."
}
```

#### Stream Track Audio
```http
GET /stream/{track_id}
```

Example:
```bash
curl http://localhost:8080/stream/3n3Ppam7vgaVa1iaRUc9Lp -o track.wav
```

Returns WAV audio stream (16-bit PCM, 44.1kHz, stereo).

#### Search Tracks
```http
GET /search?q={query}
```

Example:
```bash
curl "http://localhost:8080/search?q=mr+brightside"
```

*(Coming soon)*

#### Get Album
```http
GET /album/{album_id}
```

*(Coming soon)*

#### Get Playlist
```http
GET /playlist/{playlist_id}
```

*(Coming soon)*

## Integration with Nezulity

This service is designed to work with [Nezulity](https://github.com/nezuchan/nezulity), a Discord music bot. Nezulity can use this service as a source for Spotify tracks:

1. Start the spotify-streamer service
2. Configure Nezulity with `LIBRESPOT_URL=http://localhost:8080`
3. Nezulity will use the `/track` and `/stream` endpoints to play Spotify tracks

## Development

### Building

```bash
go build -o spotify-streamer ./cmd/server
```

### Running Tests

```bash
go test ./...
```

### Project Structure

```
spotify-streamer/
├── cmd/
│   └── server/          # Main application entry point
├── config/              # Configuration management
│   ├── config.go       # Config loading and parsing
│   ├── state.go        # State persistence
│   └── logger.go       # Logger adapter for go-librespot
├── server/              # HTTP server and handlers
│   ├── server.go       # HTTP server and routes
│   ├── librespot.go    # go-librespot integration
│   └── stream_manager.go # Track download and caching
├── internal/
│   └── models/         # Data models and responses
└── config.yaml         # Configuration file
```

## Known Limitations

- Audio is streamed as WAV (PCM) format instead of OGG Vorbis (future enhancement)
- Search, album, and playlist endpoints are not yet implemented
- No support for seeking within tracks
- Requires Spotify Premium account
- Tracks are fully downloaded before streaming begins (no progressive streaming yet)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- [go-librespot](https://github.com/devgianlu/go-librespot) - Open source Spotify client library
- [Nezulity](https://github.com/nezuchan/nezulity) - Discord music bot

## Disclaimer

This project is for educational purposes only. Please respect Spotify's Terms of Service and only use this with a valid Spotify Premium subscription.
