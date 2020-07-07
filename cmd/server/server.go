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
	"time"

	"github.com/dnys1/grpc-mongo/internal"
	"github.com/dnys1/grpc-mongo/internal/model/blogpb"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	// Command-line options
	host = flag.String("host", "localhost", "gRPC server endpoint host")
	port = flag.Int("port", 50051, "gRPC server endpoint port")
)

func main() {
	// Parse command-line flags
	flag.Parse()

	// If we crash the Go code, we get the filename and line number
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Create MongoDB client
	log.Println("Connecting to MongoDB...")
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatalf("Error instantiating MongoDB client: %v", err)
	}

	// Connect to MongoDB client
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		log.Fatalf("Error connecting to MongoDB instance: %v", err)
	}

	defer func() {
		log.Println("Closing MongoDB connection...")
		if err = client.Disconnect(ctx); err != nil {
			log.Fatalf("Error closing MongoDB connection: %v", err)
		}
		log.Println("MongoDB connection closed successfully.")
	}()

	// Ping the MongoDB server
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		log.Fatalf("Error pinging the MongoDB instance: %v", err)
	}

	// Connect to gRPC service
	log.Println("Connecting to gRPC service...")
	grpcEndpoint := fmt.Sprintf("%s:%d", *host, *port)
	lis, err := net.Listen("tcp", grpcEndpoint)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	blogpb.RegisterBlogServiceServer(grpcServer, internal.NewServer(client))

	// Register reflection service on gRPC server
	reflection.Register(grpcServer)

	// Start the gRPC server
	go func() {
		log.Printf("Starting server on port %d ...", *port)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC server: %v", err)
		}
	}()

	// Start the reverse proxy
	go func() {
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		mux := runtime.NewServeMux()
		opts := []grpc.DialOption{grpc.WithInsecure()}
		err := blogpb.RegisterBlogServiceHandlerFromEndpoint(ctx, mux, grpcEndpoint, opts)
		if err != nil {
			log.Fatalf("Error registering reverse proxy: %v", err)
		}

		if err := http.ListenAndServe(":8081", mux); err != nil {
			log.Fatalf("Failed to serve reverse proxy: %v", err)
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
