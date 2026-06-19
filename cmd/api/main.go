package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"earthquake-api/internal/bmkg"
	"earthquake-api/internal/config"
	"earthquake-api/internal/handler"
	"earthquake-api/internal/middleware"
	"earthquake-api/internal/service"

	"earthquake-api/docs"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

//	@title				BMKG Earthquake API
//	@version			1.0.0
//	@description		API for fetching earthquake data from BMKG (Badan Meteorologi, Klimatologi, dan Geofisika Indonesia).
//	@description		Data sumber: https://data.bmkg.go.id/gempabumi/
//	@description		**Atribusi**: Wajib mencantumkan BMKG sebagai sumber data pada aplikasi/sistem Anda.
//
//	@contact.name		BMKG
//	@contact.url		https://data.bmkg.go.id
//
//	@license.name		MIT
//	@license.url		https://opensource.org/licenses/MIT
//
//	@host				localhost:8090
//	@BasePath			/
//	@schemes			http https
//
//	@tag.name			Earthquake
//	@tag.description	Endpoint data gempabumi
//	@tag.name			System
//	@tag.description	Endpoint sistem (health check, informasi API)
func main() {
	cfg := config.Load()

	// Set Swagger host dynamically from config
	docs.SwaggerInfo.Host = cfg.Server.ListenAddr()

	// Initialize BMKG client
	bmkgClient := bmkg.New(cfg.BMKG)

	// Initialize service layer
	eqService := service.NewEarthquakeService(bmkgClient, cfg.Cache)

	// Initialize handler
	eqHandler := handler.NewEarthquakeHandler(eqService)

	// Setup router with middleware chain
	mux := http.NewServeMux()

	// Public endpoints
	mux.HandleFunc("GET /", eqHandler.Home)
	mux.HandleFunc("GET /health", eqHandler.Health)

	// API v1 endpoints
	mux.HandleFunc("GET /api/v1/earthquake/latest", eqHandler.GetLatest)
	mux.HandleFunc("GET /api/v1/earthquake/list/magnitude5", eqHandler.GetM5Plus)
	mux.HandleFunc("GET /api/v1/earthquake/list/felt", eqHandler.GetFelt)
	mux.HandleFunc("GET /api/v1/earthquake/shakemap", eqHandler.GetShakemap)

	// Swagger documentation
	mux.Handle("GET /swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("list"),
		httpSwagger.DomID("swagger-ui"),
	))

	// Build middleware chain (outermost first)
	var h http.Handler = mux
	h = middleware.Recovery(h)
	h = middleware.RequestID(h)
	h = middleware.CORS(cfg.CORS.AllowedOrigins, cfg.CORS.AllowedMethods, cfg.CORS.AllowedHeaders)(h)
	h = middleware.Logger(h)

	// Create HTTP server
	srv := &http.Server{
		Addr:         cfg.Server.ListenAddr(),
		Handler:      h,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Graceful shutdown
	go func() {
		log.Printf("Earthquake API server starting on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped gracefully")
}
