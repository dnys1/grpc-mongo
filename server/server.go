package main

import (
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/dnys1/grpc-mongo/server/model/blogpb"
	"google.golang.org/grpc"
)

type server struct {
	blogpb.UnimplementedBlogServiceServer
}

func newServer() *server {
	return &server{}
}

func main() {
	// If we crash the Go code, we get the filename and line number
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	blogpb.RegisterBlogServiceServer(grpcServer, newServer())

	go func() {
		log.Println("Starting server on port 50051...")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
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
}
