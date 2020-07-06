package main

import (
	"context"
	"log"
	"time"

	"github.com/dnys1/grpc-mongo/server/model/blogpb"
	"google.golang.org/grpc"
)

func main() {
	cc, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Could not connect to gRPC server: %v", err)
	}
	defer cc.Close()

	c := blogpb.NewBlogServiceClient(cc)

	if err := createBlog(c); err != nil {
		log.Fatalf("Error creating blog: %v", err)
	}
}

func createBlog(c blogpb.BlogServiceClient) error {
	req := &blogpb.CreateBlogRequest{
		Blog: &blogpb.Blog{
			AuthorId: "Dillon Nys",
			Title:    "Blog Post #1",
			Content:  "My very first blog!",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := c.CreateBlog(ctx, req)
	if err != nil {
		return err
	}

	log.Printf("Received CreateBlog response: %v", res.GetBlog())
	return nil
}
