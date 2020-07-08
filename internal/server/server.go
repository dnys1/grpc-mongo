package server

import (
	"context"
	"log"

	"github.com/dnys1/grpc-mongo/internal/model/blogpb"
	"github.com/dnys1/grpc-mongo/internal/server/database"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// A mapping of a blog item to MongoDB types
type blogItem struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	AuthorID string             `bson:"author_id,omitempty"`
	Title    string             `bson:"title,omitempty"`
	Content  string             `bson:"content,omitempty"`
}

// Server is the interface for interacting with the database.
type Server struct {
	// The database for the server
	db database.Database
	blogpb.UnimplementedBlogServiceServer
}

// NewServer creates a new Server object.
func NewServer(db database.Database) *Server {
	return &Server{
		db: db,
	}
}

// CreateBlog creates a blog in the database.
func (s *Server) CreateBlog(ctx context.Context, req *blogpb.CreateBlogRequest) (*blogpb.CreateBlogResponse, error) {
	blog := req.GetBlog()
	log.Printf("CreateBlog: Invoked with blog item %v", blog)

	res, err := s.db.CreateBlog(ctx, blog)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error inserting document: %v", err)
	}

	log.Printf("CreateBlog: Blog item successfully created (id %s)", res.GetId())

	return &blogpb.CreateBlogResponse{
		Blog: res,
	}, nil
}

// ReadBlog reads a blog from the database.
func (s *Server) ReadBlog(ctx context.Context, req *blogpb.ReadBlogRequest) (*blogpb.ReadBlogResponse, error) {
	id := req.GetId()
	log.Printf("ReadBlog: Invoked with id %s", id)

	res, err := s.db.ReadBlog(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error retrieving document: %v", err)
	}

	log.Println("ReadBlog: Blog successfully found")

	return &blogpb.ReadBlogResponse{
		Blog: res,
	}, nil
}

// UpdateBlog updates a blog in the database.
func (s *Server) UpdateBlog(ctx context.Context, req *blogpb.UpdateBlogRequest) (*blogpb.UpdateBlogResponse, error) {
	blog := req.GetBlog()
	log.Printf("UpdateBlog: Invoked with blog %v", blog)

	res, err := s.db.UpdateBlog(ctx, blog)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error updating document: %v", err)
	}

	return &blogpb.UpdateBlogResponse{
		Status: res,
	}, nil
}

// DeleteBlog deletes a blog from the database.
func (s *Server) DeleteBlog(ctx context.Context, req *blogpb.DeleteBlogRequest) (*blogpb.DeleteBlogResponse, error) {
	id := req.GetId()
	log.Printf("DeleteBlog: Invoked with id %v", id)

	res, err := s.db.DeleteBlog(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error deleting document: %v", err)
	}

	return &blogpb.DeleteBlogResponse{
		Status: res,
	}, nil
}

// ListBlogs lists all the blogs in the database.
func (s *Server) ListBlogs(req *blogpb.ListBlogsRequest, stream blogpb.BlogService_ListBlogsServer) error {
	log.Println("ListBlog: Invoked with no parameters")

	if err := s.db.ListBlogs(stream); err != nil {
		return status.Errorf(codes.Internal, "Error listing documents: %v", err)
	}

	return nil
}
