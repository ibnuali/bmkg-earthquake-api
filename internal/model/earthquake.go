package model

import "time"

// Earthquake represents a single earthquake event from BMKG.
type Earthquake struct {
	Date        string `json:"Tanggal"`
	Time        string `json:"Jam"`
	DateTime    string `json:"DateTime"`
	Coordinates string `json:"Coordinates"`
	Latitude    string `json:"Lintang"`
	Longitude   string `json:"Bujur"`
	Magnitude   string `json:"Magnitude"`
	Depth       string `json:"Kedalaman"`
	Region      string `json:"Wilayah"`
	Potency     string `json:"Potensi,omitempty"`
	Felt        string `json:"Dirasakan,omitempty"`
	Shakemap    string `json:"Shakemap,omitempty"`
}

// AutoGempaResponse wraps the single earthquake response from BMKG.
type AutoGempaResponse struct {
	Infogempa struct {
		Gempa Earthquake `json:"gempa"`
	} `json:"Infogempa"`
}

// GempaListResponse wraps the earthquake list response from BMKG.
type GempaListResponse struct {
	Infogempa struct {
		Gempa []Earthquake `json:"gempa"`
	} `json:"Infogempa"`
}

// ParsedEarthquake holds a processed earthquake with parsed fields.
type ParsedEarthquake struct {
	Date        string    `json:"date"`
	Time        string    `json:"time"`
	DateTime    time.Time `json:"datetime"`
	Coordinates []float64 `json:"coordinates"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	Magnitude   float64   `json:"magnitude"`
	DepthKM     float64   `json:"depth_km"`
	Region      string    `json:"region"`
	Potency     string    `json:"potency,omitempty"`
	Felt        string    `json:"felt,omitempty"`
	ShakemapURL string    `json:"shakemap_url,omitempty"`
}
