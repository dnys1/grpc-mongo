package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/dnys1/grpc-mongo/internal/gateway"
	"github.com/dnys1/grpc-mongo/internal/model/blogpb"
	"github.com/dnys1/grpc-mongo/internal/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
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
	mongoEndpoint := fmt.Sprintf("mongodb://%s:%d", *dbHost, *dbPort)
	log.Printf("Connecting to MongoDB at %s ...", mongoEndpoint)
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoEndpoint).SetConnectTimeout(30 * time.Second))
	if err != nil {
		log.Fatalf("Error instantiating MongoDB client: %v", err)
	}

	// Connect to MongoDB client
	if err = client.Connect(context.Background()); err != nil {
		log.Fatalf("Error connecting to MongoDB instance: %v", err)
	}

	defer func() {
		log.Println("Closing MongoDB connection...")
		if err = client.Disconnect(context.Background()); err != nil {
			log.Fatalf("Error closing MongoDB connection: %v", err)
		}
		log.Println("MongoDB connection closed successfully.")
	}()

	// Ping the MongoDB server
	if err := client.Ping(context.Background(), readpref.Primary()); err != nil {
		log.Fatalf("Error pinging the MongoDB instance: %v", err)
	}

	// Connect to gRPC service
	log.Printf("Starting gRPC server on port %d ...", *grpcPort)
	grpcEndpoint := fmt.Sprintf("%s:%d", *grpcHost, *grpcPort)
	lis, err := net.Listen("tcp", grpcEndpoint)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	blogpb.RegisterBlogServiceServer(grpcServer, server.NewServer(client))

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
