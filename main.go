package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/dnys1/grpc-mongo/internal/gateway"
	"github.com/dnys1/grpc-mongo/internal/model/blogpb"
	"github.com/dnys1/grpc-mongo/internal/server"
	db "github.com/dnys1/grpc-mongo/internal/server/database/mongo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	// Command-line options
	grpcHost = flag.String("grpc-host", "localhost", "gRPC server endpoint host")
	grpcPort = flag.Int("grpc-port", 50051, "gRPC server endpoint port")
	dbHost   = flag.String("db-host", "localhost", "Database host")
	dbPort   = flag.Int("db-port", 27017, "Database port")
)

func main() {
	// Parse command-line flags
	flag.Parse()

	// If we crash the Go code, we get the filename and line number
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Create MongoDB client
	db, err := db.New(&db.MongoDatabaseOptions{
		Host: *dbHost,
		Port: *dbPort,
	})
	if err != nil {
		log.Fatalf("Error creating database: %v", err)
	}
	log.Printf("Connecting to database at %s ...", db.Endpoint())

	defer func() {
		log.Println("Closing database connection...")
		if err = db.Disconnect(context.Background()); err != nil {
			log.Fatalf("Error closing database connection: %v", err)
		}
		log.Println("MongoDB connection closed successfully.")
	}()

	// Connect to gRPC service
	log.Printf("Starting gRPC server on port %d ...", *grpcPort)
	grpcEndpoint := fmt.Sprintf("%s:%d", *grpcHost, *grpcPort)
	lis, err := net.Listen("tcp", grpcEndpoint)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	blogpb.RegisterBlogServiceServer(grpcServer, server.NewServer(db))

	// Register reflection service on gRPC server
	reflection.Register(grpcServer)

	// Start the gRPC server
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC server: %v", err)
		}
	}()

	// Start the gateway reverse proxy
	go func() {
		if err := gateway.Run(grpcEndpoint); err != nil {
			log.Fatalf("Failed to serve gateway server: %v", err)
		}
	}()

	// Wait for Control-C to exit
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)

	// Block until a signal is received
	<-ch

	// Shut down server
	log.Println("Shutting down server...")
	grpcServer.Stop()
	lis.Close()
	log.Println("Server shut down successfully.")
}
