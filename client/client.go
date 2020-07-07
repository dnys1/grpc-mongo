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

	id, err := createBlog(c)
	if err != nil {
		log.Fatalf("Error creating blog: %v", err)
	}

	if err := readBlog(c, id); err != nil {
		log.Fatalf("Error reading blog: %v", err)
	}
}

func createBlog(c blogpb.BlogServiceClient) (string, error) {
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
		return "", err
	}

	blog := res.GetBlog()
	log.Printf("Received CreateBlog response: %v", blog)
	return blog.GetId(), nil
}

func readBlog(c blogpb.BlogServiceClient, id string) error {
	req := &blogpb.ReadBlogRequest{
		BlogId: id,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := c.ReadBlog(ctx, req)
	if err != nil {
		return err
	}

	blog := res.GetBlog()
	log.Printf("Received ReadBlogResponse: %v", blog)
	return nil
}
