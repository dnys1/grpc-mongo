package internal

import (
	"context"
	"log"

	"github.com/dnys1/grpc-mongo/internal/model/blogpb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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
	// The master database collection from MongoDB
	Collection *mongo.Collection
	blogpb.UnimplementedBlogServiceServer
}

// NewServer creates a new Server object.
func NewServer(client *mongo.Client) *Server {
	return &Server{
		Collection: client.Database("mydb").Collection("blog"),
	}
}

// CreateBlog creates a blog in the database.
func (s *Server) CreateBlog(ctx context.Context, req *blogpb.CreateBlogRequest) (*blogpb.CreateBlogResponse, error) {
	blog := req.GetBlog()
	log.Printf("CreateBlog: Invoked with blog item %v", blog)

	data := blogItem{
		AuthorID: blog.GetAuthorId(),
		Title:    blog.GetTitle(),
		Content:  blog.GetContent(),
	}

	res, err := s.Collection.InsertOne(ctx, data)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error inserting document: %v", data, err)
	}

	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, status.Errorf(codes.Internal, "Error converting to OID: %v", err)
	}

	log.Printf("CreateBlog: Blog item successfully created (id %s)", oid.Hex())

	return &blogpb.CreateBlogResponse{
		Blog: &blogpb.Blog{
			Id:       oid.Hex(),
			AuthorId: blog.GetAuthorId(),
			Title:    blog.GetTitle(),
			Content:  blog.GetContent(),
		},
	}, nil
}

// ReadBlog reads a blog from the database.
func (s *Server) ReadBlog(ctx context.Context, req *blogpb.ReadBlogRequest) (*blogpb.ReadBlogResponse, error) {
	id := req.GetId()
	log.Printf("ReadBlog: Invoked with id %s", id)

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Error converting from OID: %v", err)
	}

	data := &blogItem{}
	filter := bson.M{"_id": oid}

	doc := s.Collection.FindOne(ctx, filter)
	if err := doc.Decode(data); err != nil {
		return nil, status.Errorf(codes.NotFound, "Document with id %s not found: %v", id, err)
	}

	log.Println("ReadBlog: Blog successfully found")

	return &blogpb.ReadBlogResponse{
		Blog: &blogpb.Blog{
			Id:       id,
			AuthorId: data.AuthorID,
			Title:    data.Title,
			Content:  data.Content,
		},
	}, nil
}

// UpdateBlog updates a blog in the database.
func (s *Server) UpdateBlog(ctx context.Context, req *blogpb.UpdateBlogRequest) (*blogpb.UpdateBlogResponse, error) {
	blog := req.GetBlog()
	log.Printf("UpdateBlog: Invoked with blog %v", blog)

	id := blog.GetId()
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Error converting from OID: %v", err)
	}

	data := &blogItem{}
	filter := bson.M{"_id": oid}

	// Get the old doc from the DB
	doc := s.Collection.FindOne(ctx, filter)
	if err := doc.Decode(data); err != nil {
		return nil, status.Errorf(codes.NotFound, "Document with id %s not found: %v", id, err)
	}

	// Update variables on the document
	data.AuthorID = blog.GetAuthorId()
	data.Title = blog.GetTitle()
	data.Content = blog.GetContent()

	_, err = s.Collection.ReplaceOne(ctx, filter, data)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Document update failed: %v", err)
	}

	return &blogpb.UpdateBlogResponse{
		Status: blogpb.UpdateBlogResponse_UPDATED,
	}, nil
}

// DeleteBlog deletes a blog from the database.
func (s *Server) DeleteBlog(ctx context.Context, req *blogpb.DeleteBlogRequest) (*blogpb.DeleteBlogResponse, error) {
	id := req.GetId()
	log.Printf("DeleteBlog: Invoked with id %v", id)

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Error converting from OID: %v", err)
	}

	filter := bson.M{"_id": oid}

	res, err := s.Collection.DeleteOne(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error deleting document: %v", err)
	}

	if res.DeletedCount == 0 {
		log.Printf("DeleteBlog: Blog not deleted.")
		return &blogpb.DeleteBlogResponse{
			Status: blogpb.DeleteBlogResponse_NOT_DELETED,
		}, nil
	}

	log.Printf("DeleteBlog: Blog successfully deleted.")
	return &blogpb.DeleteBlogResponse{
		Status: blogpb.DeleteBlogResponse_DELETED,
	}, nil
}

// ListBlogs lists all the blogs in the database.
func (s *Server) ListBlogs(req *blogpb.ListBlogsRequest, stream blogpb.BlogService_ListBlogsServer) error {
	log.Println("ListBlog: Invoked with no parameters")

	cur, err := s.Collection.Find(context.Background(), bson.D{})
	if err != nil {
		return status.Errorf(codes.Internal, "Error listing blogs: %v", err)
	}
	defer cur.Close(context.Background())

	for cur.Next(context.Background()) {
		data := &blogItem{}
		if err := cur.Decode(data); err != nil {
			return status.Errorf(codes.Internal, "Error decoding data: %v", err)
		}

		blog := &blogpb.Blog{
			Id:       data.ID.Hex(),
			AuthorId: data.AuthorID,
			Title:    data.Title,
			Content:  data.Content,
		}
		stream.Send(&blogpb.ListBlogsResponse{Blog: blog})
	}

	if err := cur.Err(); err != nil {
		return status.Errorf(codes.Internal, "Error in database lookup: %v", err)
	}

	return nil
}
