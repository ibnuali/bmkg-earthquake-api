package service

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"earthquake-api/internal/bmkg"
	"earthquake-api/internal/cache"
	"earthquake-api/internal/config"
	"earthquake-api/internal/model"
)

const (
	cacheKeyLatest      = "earthquake:latest"
	cacheKeyM5Plus      = "earthquake:m5plus"
	cacheKeyFelt        = "earthquake:felt"
)

// EarthquakeService handles business logic for earthquake data.
type EarthquakeService struct {
	client bmkg.Client
	cache  *cache.Cache
	cacheEnabled bool
}

// NewEarthquakeService creates a new EarthquakeService.
func NewEarthquakeService(client bmkg.Client, cfg config.CacheConfig) *EarthquakeService {
	var c *cache.Cache
	if cfg.Enabled {
		c = cache.New(cfg.TTL, cfg.CleanupIntv)
	}

	return &EarthquakeService{
		client:       client,
		cache:        c,
		cacheEnabled: cfg.Enabled,
	}
}

// GetLatest returns the latest earthquake, with caching.
func (s *EarthquakeService) GetLatest() (*model.ParsedEarthquake, error) {
	if s.cacheEnabled {
		if cached := s.cache.Get(cacheKeyLatest); cached != nil {
			return cached.(*model.ParsedEarthquake), nil
		}
	}

	gempa, err := s.client.FetchLatest()
	if err != nil {
		return nil, err
	}

	parsed, err := parseEarthquake(gempa)
	if err != nil {
		return nil, fmt.Errorf("parse latest: %w", err)
	}

	if s.cacheEnabled {
		s.cache.Set(cacheKeyLatest, parsed)
	}

	return parsed, nil
}

// GetM5Plus returns the 15 latest M5.0+ earthquakes.
func (s *EarthquakeService) GetM5Plus() ([]model.ParsedEarthquake, error) {
	if s.cacheEnabled {
		if cached := s.cache.Get(cacheKeyM5Plus); cached != nil {
			return cached.([]model.ParsedEarthquake), nil
		}
	}

	list, err := s.client.FetchM5Plus()
	if err != nil {
		return nil, err
	}

	parsed, err := parseEarthquakeList(list)
	if err != nil {
		return nil, fmt.Errorf("parse m5+: %w", err)
	}

	if s.cacheEnabled {
		s.cache.Set(cacheKeyM5Plus, parsed)
	}

	return parsed, nil
}

// GetFelt returns the 15 latest felt earthquakes.
func (s *EarthquakeService) GetFelt() ([]model.ParsedEarthquake, error) {
	if s.cacheEnabled {
		if cached := s.cache.Get(cacheKeyFelt); cached != nil {
			return cached.([]model.ParsedEarthquake), nil
		}
	}

	list, err := s.client.FetchFelt()
	if err != nil {
		return nil, err
	}

	parsed, err := parseEarthquakeList(list)
	if err != nil {
		return nil, fmt.Errorf("parse felt: %w", err)
	}

	if s.cacheEnabled {
		s.cache.Set(cacheKeyFelt, parsed)
	}

	return parsed, nil
}

// GetShakemapURL returns the full shakemap URL for a given code.
func (s *EarthquakeService) GetShakemapURL(code string) (string, error) {
	if err := bmkg.ValidateURL(code); err != nil {
		return "", fmt.Errorf("%w: %v", model.ErrInvalidRequest, err)
	}
	return s.client.ShakemapURL(code), nil
}

// parseEarthquake converts a raw BMKG earthquake into a parsed one.
func parseEarthquake(e *model.Earthquake) (*model.ParsedEarthquake, error) {
	parsed := &model.ParsedEarthquake{
		Date:        e.Date,
		Time:        e.Time,
		Region:      e.Region,
		Potency:     e.Potency,
		Felt:        e.Felt,
		DateTime:    parseDateTime(e.DateTime),
		Magnitude:   parseMagnitude(e.Magnitude),
		DepthKM:     parseDepth(e.Depth),
		ShakemapURL: buildShakemapURL(e.Shakemap),
	}

	parseCoordinates(e.Coordinates, parsed)

	return parsed, nil
}

// parseDateTime parses an ISO 8601 datetime string with multiple format fallbacks.
func parseDateTime(raw string) time.Time {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05+00:00",
		"2006-01-02T15:04:05Z",
	}

	for _, format := range formats {
		if dt, err := time.Parse(format, raw); err == nil {
			return dt
		}
	}

	return time.Now().UTC()
}

// parseCoordinates parses "lat,lon" string and sets coordinate fields on parsed.
func parseCoordinates(raw string, parsed *model.ParsedEarthquake) {
	if raw == "" {
		return
	}

	parts := strings.Split(raw, ",")
	if len(parts) != 2 {
		return
	}

	lat, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	lon, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err1 != nil || err2 != nil {
		return
	}

	parsed.Coordinates = []float64{lat, lon}
	parsed.Latitude = lat
	parsed.Longitude = lon
}

// parseMagnitude parses a magnitude string to a rounded float.
func parseMagnitude(raw string) float64 {
	if raw == "" {
		return 0
	}

	mag, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}

	return math.Round(mag*10) / 10
}

// parseDepth parses a depth string like "10 km" to a float.
func parseDepth(raw string) float64 {
	if raw == "" {
		return 0
	}

	depthStr := strings.TrimSuffix(raw, " km")
	depthStr = strings.TrimSuffix(depthStr, " Km")

	depth, err := strconv.ParseFloat(strings.TrimSpace(depthStr), 64)
	if err != nil {
		return 0
	}

	return depth
}

// buildShakemapURL constructs the full shakemap URL from a code or path.
func buildShakemapURL(raw string) string {
	if raw == "" {
		return ""
	}

	if strings.HasPrefix(raw, "http") {
		return raw
	}

	return fmt.Sprintf("https://static.bmkg.go.id/%s", raw)
}

// parseEarthquakeList converts a slice of raw earthquakes.
func parseEarthquakeList(list []model.Earthquake) ([]model.ParsedEarthquake, error) {
	result := make([]model.ParsedEarthquake, 0, len(list))
	for _, e := range list {
		parsed, err := parseEarthquake(&e)
		if err != nil {
			continue // skip malformed entries
		}
		result = append(result, *parsed)
	}
	return result, nil
}
