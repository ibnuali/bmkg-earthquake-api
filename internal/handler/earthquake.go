package handler

import (
	"errors"
	"net/http"
	"strings"

	"earthquake-api/internal/model"
	"earthquake-api/internal/response"
	"earthquake-api/internal/service"
)


// EarthquakeHandler handles HTTP requests for earthquake data.
type EarthquakeHandler struct {
	svc *service.EarthquakeService
}

// NewEarthquakeHandler creates a new EarthquakeHandler.
func NewEarthquakeHandler(svc *service.EarthquakeService) *EarthquakeHandler {
	return &EarthquakeHandler{svc: svc}
}

// GetLatest handles GET /api/v1/earthquake/latest
//
//	@Summary		Get latest earthquake
//	@Description	Mengambil data gempabumi terkini dari BMKG.
//	@Description	Data mencakup magnitudo, kedalaman, koordinat, lokasi, dan potensi tsunami.
//	@Tags			Earthquake
//	@Produce		json
//	@Success		200	{object}	response.APIResponse	"Data gempa terbaru"
//	@Failure		503	{object}	response.APIResponse	"Gagal mengambil data dari BMKG"
//	@Router			/api/v1/earthquake/latest [get]
func (h *EarthquakeHandler) GetLatest(w http.ResponseWriter, r *http.Request) {
	gempa, err := h.svc.GetLatest()
	if err != nil {
		writeError(w, err)
		return
	}

	response.Success(w, gempa)
}

// GetM5Plus handles GET /api/v1/earthquake/list/magnitude5
//
//	@Summary		Get 15 latest M5.0+ earthquakes
//	@Description	Mengambil 15 gempabumi terakhir dengan magnitudo >= 5.0 dari BMKG.
//	@Tags			Earthquake
//	@Produce		json
//	@Success		200	{array}		response.APIResponse	"Daftar 15 gempa M5.0+"
//	@Failure		503	{object}	response.APIResponse	"Gagal mengambil data dari BMKG"
//	@Router			/api/v1/earthquake/list/magnitude5 [get]
func (h *EarthquakeHandler) GetM5Plus(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.GetM5Plus()
	if err != nil {
		writeError(w, err)
		return
	}

	response.Success(w, list)
}

// GetFelt handles GET /api/v1/earthquake/list/felt
//
//	@Summary		Get 15 latest felt earthquakes
//	@Description	Mengambil 15 gempabumi terakhir yang dirasakan (berdasarkan skala MMI) dari BMKG.
//	@Tags			Earthquake
//	@Produce		json
//	@Success		200	{array}		response.APIResponse	"Daftar 15 gempa dirasakan"
//	@Failure		503	{object}	response.APIResponse	"Gagal mengambil data dari BMKG"
//	@Router			/api/v1/earthquake/list/felt [get]
func (h *EarthquakeHandler) GetFelt(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.GetFelt()
	if err != nil {
		writeError(w, err)
		return
	}

	response.Success(w, list)
}

// GetShakemap handles GET /api/v1/earthquake/shakemap?code=xxx
// It redirects to the BMKG static shakemap image.
//
//	@Summary		Get shakemap image
//	@Description	Redirect ke gambar shakemap (peta guncangan) dari server statis BMKG.
//	@Description	Gunakan parameter `code` yang diperoleh dari response endpoint gempa terbaru (field `shakemap_url`).
//	@Tags			Earthquake
//	@Produce		json
//	@Param			code	query		string	true	"Kode shakemap (contoh: 20260619085320.mmi.jpg)"
//	@Success		302		{string}	string	"Redirect ke gambar shakemap BMKG"
//	@Failure		400		{object}	response.APIResponse	"Parameter code tidak valid atau tidak disertakan"
//	@Router			/api/v1/earthquake/shakemap [get]
func (h *EarthquakeHandler) GetShakemap(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PARAMETER", "query parameter 'code' is required")
		return
	}

	shakemapURL, err := h.svc.GetShakemapURL(code)
	if err != nil {
		writeError(w, err)
		return
	}

	http.Redirect(w, r, shakemapURL, http.StatusFound)
}

// Health handles GET /health
//
//	@Summary		Health check
//	@Description	Memeriksa status kesehatan API service.
//	@Tags			System
//	@Produce		json
//	@Success		200	{object}	response.APIResponse	"Service sehat"
//	@Router			/health [get]
func (h *EarthquakeHandler) Health(w http.ResponseWriter, r *http.Request) {
	response.Success(w, map[string]string{
		"status": "healthy",
	})
}

// Home handles GET /
//
//	@Summary		API information
//	@Description	Menampilkan informasi API dan daftar endpoint yang tersedia.
//	@Tags			System
//	@Produce		json
//	@Success		200	{object}	response.APIResponse	"Informasi API"
//	@Router			/ [get]
func (h *EarthquakeHandler) Home(w http.ResponseWriter, r *http.Request) {
	response.Success(w, map[string]interface{}{
		"name":        "BMKG Earthquake API",
		"version":     "1.0.0",
		"description": "API for fetching earthquake data from BMKG (Badan Meteorologi, Klimatologi, dan Geofisika Indonesia)",
		"endpoints": map[string]string{
			"GET /":                                  "API information",
			"GET /health":                            "Health check",
			"GET /api/v1/earthquake/latest":          "Latest earthquake",
			"GET /api/v1/earthquake/list/magnitude5": "15 latest M5.0+ earthquakes",
			"GET /api/v1/earthquake/list/felt":       "15 latest felt earthquakes",
			"GET /api/v1/earthquake/shakemap":        "Redirect to shakemap (use ?code= parameter)",
			"GET /swagger/":                          "Swagger API documentation",
		},
		"source": "https://data.bmkg.go.id/gempabumi/",
	})
}

func detectContentType(p string) string {
	switch {
	case strings.HasSuffix(p, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(p, ".js"):
		return "application/javascript; charset=utf-8"
	case strings.HasSuffix(p, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(p, ".png"):
		return "image/png"
	case strings.HasSuffix(p, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(p, ".json"):
		return "application/json"
	default:
		return "text/plain; charset=utf-8"
	}
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, model.ErrNotFound):
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, model.ErrRateLimited):
		response.Error(w, http.StatusTooManyRequests, "UPSTREAM_RATE_LIMITED", "BMKG rate limit exceeded. Please try again later.")
	case errors.Is(err, model.ErrInvalidRequest):
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	default:
		response.Error(w, http.StatusServiceUnavailable, "UPSTREAM_ERROR", "Failed to fetch data from BMKG")
	}
}
