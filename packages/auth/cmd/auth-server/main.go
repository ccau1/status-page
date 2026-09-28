package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"status-page/packages/auth"
	"status-page/packages/auth/proto"
	"status-page/packages/auth/server"
	"status-page/packages/core/envutil"
)

func main() {
	envutil.LoadDotEnv()

	port := os.Getenv("AUTH_GRPC_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "50051"
	}

	cfg := auth.LoadConfigFromEnv()
	log.Printf("[Auth gRPC] Initializing with provider: %s (%s)", cfg.ProviderName, cfg.ProviderType)

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("[Auth gRPC] Failed to listen on port %s: %v", port, err)
	}

	grpcServer := grpc.NewServer()
	authServer := server.NewAuthServer(cfg)
	proto.RegisterAuthServiceServer(grpcServer, authServer)

	go func() {
		log.Printf("[Auth gRPC] Stateless Auth gRPC service listening on :%s", port)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("[Auth gRPC] Server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("[Auth gRPC] Shutting down gracefully...")
	grpcServer.GracefulStop()
	fmt.Println("[Auth gRPC] Service stopped.")
}
