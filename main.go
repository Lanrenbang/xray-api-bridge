package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"xray-api-bridge/apiserver"
	"xray-api-bridge/bridge"
	"xray-api-bridge/xrayapi"
)

var (
	versionFlag bool
)

func init() {
	flag.BoolVar(&versionFlag, "v", false, "Print version and exit")
	flag.BoolVar(&versionFlag, "version", false, "Print version and exit")
}

func main() {
	flag.Parse()

	if versionFlag {
		fmt.Println(bridge.GetVersion())
		os.Exit(0)
	}

	fmt.Printf("Xray API Bridge %s\n", bridge.GetVersion())
	fmt.Println("Starting...")

	// Load configuration from environment variables
	listenAddr := os.Getenv("XRAY_API_BRIDGE_LISTEN")
	if listenAddr == "" {
		listenAddr = ":8081" // Default listen address
		log.Printf("XRAY_API_BRIDGE_LISTEN not set, using default: %s", listenAddr)
	}

	grpcAddress := os.Getenv("XRAY_API_BRIDGE_UPSTREAM")
	if grpcAddress == "" {
		log.Fatalf("XRAY_API_BRIDGE_UPSTREAM environment variable is required")
	}

	subsConfigPath := os.Getenv("XRAY_API_BRIDGE_SUBS_CONFIG")
	// subsConfigPath can be empty if subscription endpoint is not used.
	// The handler for that endpoint should check if the path is configured.

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize gRPC client
	fmt.Printf("Connecting to Xray gRPC server at %s...\n", grpcAddress)
	xrayClient, err := xrayapi.NewClient(ctx, grpcAddress)
	if err != nil {
		// With the new logic, a gRPC connection is mandatory.
		log.Fatalf("Failed to create Xray gRPC client: %v. A running Xray-core instance with gRPC API enabled is required.", err)
	}
	defer xrayClient.Close()
	fmt.Println("Successfully connected to Xray gRPC server.")

	// Initialize Chi router and API server
	apiServer := apiserver.NewAPIServer(xrayClient, listenAddr, subsConfigPath)

	// Start the HTTP server in a goroutine
	go func() {
		// Check if the listen address is a Unix socket
		if strings.HasPrefix(listenAddr, "/") {
			// It's a file-system Unix socket. Clean up existing socket file before starting.
			if err := os.Remove(listenAddr); err != nil && !os.IsNotExist(err) {
				log.Fatalf("Failed to remove existing unix socket: %v", err)
			}

			listener, err := net.Listen("unix", listenAddr)
			if err != nil {
				log.Fatalf("Failed to listen on unix socket: %v", err)
			}
			defer listener.Close()

			// When using a unix socket, we must change the file permissions to allow the web server to access it
			if err := os.Chmod(listenAddr, 0666); err != nil {
				log.Fatalf("Failed to change unix socket permissions: %v", err)
			}

			log.Printf("HTTP server listening on unix socket %s\n", listenAddr)
			if err := http.Serve(listener, apiServer.GetHandler()); err != nil && err != http.ErrServerClosed {
				log.Fatalf("HTTP server failed to start: %v", err)
			}
		} else if strings.HasPrefix(listenAddr, "@") {
			// It's an abstract Unix socket, no file to clean up.
			log.Printf("HTTP server listening on abstract unix socket %s\n", listenAddr)
			listener, err := net.Listen("unix", listenAddr)
			if err != nil {
				log.Fatalf("Failed to listen on abstract unix socket: %v", err)
			}
			defer listener.Close()

			if err := http.Serve(listener, apiServer.GetHandler()); err != nil && err != http.ErrServerClosed {
				log.Fatalf("HTTP server failed to start: %v", err)
			}
		} else {
			// It's a regular TCP address
			if err := apiServer.Start(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("HTTP server failed to start: %v", err)
			}
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("Shutting down Xray API Bridge...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Clean up the unix socket file on shutdown
	if strings.HasPrefix(listenAddr, "/") {
		if err := os.Remove(listenAddr); err != nil && !os.IsNotExist(err) {
			log.Printf("Failed to remove unix socket file on shutdown: %v", err)
		}
	}

	fmt.Println("Xray API Bridge stopped.")
}

