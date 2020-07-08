package database

import (
	"context"

	"github.com/dnys1/grpc-mongo/internal/model/blogpb"
)

// Database defines the functionality required from a database client.
type Database interface {
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	Endpoint() string
	// Creates a blog in the database
	CreateBlog(ctx context.Context, blog *blogpb.Blog) (*blogpb.Blog, error)
	// Reads a user from the database
	ReadBlog(ctx context.Context, id string) (*blogpb.Blog, error)
	// Updates a blog in the database
	UpdateBlog(ctx context.Context, blog *blogpb.Blog) (blogpb.UpdateBlogResponse_UpdateStatus, error)
	// Deletes a blog from the database
	DeleteBlog(ctx context.Context, id string) (blogpb.DeleteBlogResponse_DeleteStatus, error)
	// Lists all the blogs in the database
	ListBlogs(stream blogpb.BlogService_ListBlogsServer) error
}
