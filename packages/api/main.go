package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"status-page/packages/api/handlers"
	"status-page/packages/auth/client"
	"status-page/packages/core/adapters/factory"
	"status-page/packages/core/domain"
	"status-page/packages/core/envutil"
	"status-page/packages/core/ports"
)

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	envutil.LoadDotEnv()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := factory.LoadConfigFromEnv()
	log.Printf("[API] Initializing storage engine: %s", cfg.Engine)

	storage, err := factory.NewStoragePort(ctx, cfg)
	if err != nil {
		log.Fatalf("[API] Failed to initialize storage port: %v", err)
	}
	defer storage.Close()

	mux := http.NewServeMux()
	statusHandler := handlers.NewStatusHandler(storage)

	// Connect to standalone Auth Service via gRPC
	authGRPCAddr := os.Getenv("AUTH_GRPC_ADDR")
	if authGRPCAddr == "" {
		authGRPCAddr = "localhost:50051"
	}
	log.Printf("[API] Connecting to Auth Service via gRPC at %s...", authGRPCAddr)
	authClient, err := client.NewAuthClient(authGRPCAddr)
	if err != nil {
		log.Fatalf("[API] Failed to initialize auth gRPC client: %v", err)
	}
	defer authClient.Close()

	authGateway := handlers.NewAuthGateway(authClient)
	authGateway.RegisterRoutes(mux)

	// Public Read Endpoints
	mux.HandleFunc("GET /health", statusHandler.Health)
	mux.HandleFunc("GET /api/status", statusHandler.ListAll)
	mux.HandleFunc("GET /api/status/{tenant}", statusHandler.ListByTenant)
	mux.HandleFunc("GET /api/status/{tenant}/{product}", statusHandler.GetProductStatus)
	mux.HandleFunc("GET /api/incidents", statusHandler.ListIncidents)
	mux.HandleFunc("GET /api/incidents/{tenant}", statusHandler.ListIncidents)

	// Secure API Endpoints (Guarded by gRPC-backed Auth Middleware)
	mux.Handle("POST /api/incidents", client.RequireAdmin(authClient, "status_session")(http.HandlerFunc(statusHandler.CreateOrUpdateIncident)))
	mux.Handle("DELETE /api/incidents/{id}", client.RequireAdmin(authClient, "status_session")(http.HandlerFunc(statusHandler.DeleteIncident)))

	// Sync active banners from INCIDENT_BANNER_JSON if configured in environment
	syncIncidentsFromEnv(ctx, storage)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      withCORS(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("[API] Status Page API server listening on :%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[API] Server failed: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("[API] Shutting down gracefully...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("[API] Server shutdown error: %v", err)
	}
	fmt.Println("[API] Server exited successfully.")
}

func syncIncidentsFromEnv(ctx context.Context, storage ports.StoragePort) {
	raw := os.Getenv("INCIDENT_BANNER_JSON")
	if raw == "" {
		return
	}
	var incidents []domain.Incident
	if err := json.Unmarshal([]byte(raw), &incidents); err != nil {
		log.Printf("[API] Warning: Failed to parse INCIDENT_BANNER_JSON: %v", err)
		return
	}
	for _, inc := range incidents {
		if inc.CreatedAt.IsZero() {
			inc.CreatedAt = time.Now().UTC()
		}
		if inc.UpdatedAt.IsZero() {
			inc.UpdatedAt = time.Now().UTC()
		}
		if err := storage.SaveIncident(ctx, &inc); err != nil {
			log.Printf("[API] Warning: Failed to sync incident %s: %v", inc.ID, err)
		} else {
			log.Printf("[API] Synced incident banner from ENV: %s ('%s')", inc.ID, inc.Title)
		}
	}
}
