package models

// TrackResponse represents a track metadata response
type TrackResponse struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Artist   string   `json:"artist"`
	Artists  []string `json:"artists"`
	Album    string   `json:"album"`
	Duration int64    `json:"duration"` // milliseconds
	ISRC     string   `json:"isrc,omitempty"`
	URI      string   `json:"uri"`
	URL      string   `json:"url"`
	Artwork  string   `json:"artwork,omitempty"`
}

// AlbumResponse represents an album response
type AlbumResponse struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	Artist  string          `json:"artist"`
	Artwork string          `json:"artwork,omitempty"`
	Tracks  []TrackResponse `json:"tracks"`
}

// PlaylistResponse represents a playlist response
type PlaylistResponse struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	Owner   string          `json:"owner,omitempty"`
	Artwork string          `json:"artwork,omitempty"`
	Tracks  []TrackResponse `json:"tracks"`
}

// SearchResponse represents search results
type SearchResponse struct {
	Query   string          `json:"query"`
	Results []TrackResponse `json:"results"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status        string `json:"status"`
	Authenticated bool   `json:"authenticated"`
	Username      string `json:"username,omitempty"`
}
