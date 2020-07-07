package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/dnys1/grpc-mongo/server/model/blogpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// The master database collection from MongoDB
var collection *mongo.Collection

// A mapping of a blog item to MongoDB types
type blogItem struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	AuthorID string             `bson:"author_id,omitempty"`
	Title    string             `bson:"title,omitempty"`
	Content  string             `bson:"content,omitempty"`
}

type server struct {
	blogpb.UnimplementedBlogServiceServer
}

func newServer() *server {
	return &server{}
}

func (*server) CreateBlog(ctx context.Context, req *blogpb.CreateBlogRequest) (*blogpb.CreateBlogResponse, error) {
	blog := req.GetBlog()
	log.Printf("CreateBlog: Invoked with blog item %v", blog)

	data := blogItem{
		AuthorID: blog.GetAuthorId(),
		Title:    blog.GetTitle(),
		Content:  blog.GetContent(),
	}

	res, err := collection.InsertOne(ctx, data)
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

func (*server) ReadBlog(ctx context.Context, req *blogpb.ReadBlogRequest) (*blogpb.ReadBlogResponse, error) {
	id := req.GetBlogId()
	log.Printf("ReadBlog: Invoked with id %s", id)

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Error converting from OID: %v", err)
	}

	data := &blogItem{}
	filter := bson.M{"_id": oid}

	doc := collection.FindOne(ctx, filter)
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

func (*server) UpdateBlog(ctx context.Context, req *blogpb.UpdateBlogRequest) (*blogpb.UpdateBlogResponse, error) {
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
	doc := collection.FindOne(ctx, filter)
	if err := doc.Decode(data); err != nil {
		return nil, status.Errorf(codes.NotFound, "Document with id %s not found: %v", id, err)
	}

	// Update variables on the document
	data.AuthorID = blog.GetAuthorId()
	data.Title = blog.GetTitle()
	data.Content = blog.GetContent()

	_, err = collection.ReplaceOne(ctx, filter, data)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Document update failed: %v", err)
	}

	return &blogpb.UpdateBlogResponse{
		Status: blogpb.UpdateBlogResponse_UPDATED,
	}, nil
}

func (*server) DeleteBlog(ctx context.Context, req *blogpb.DeleteBlogRequest) (*blogpb.DeleteBlogResponse, error) {
	id := req.GetBlogId()
	log.Printf("DeleteBlog: Invoked with id %v", id)

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Error converting from OID: %v", err)
	}

	filter := bson.M{"_id": oid}

	res, err := collection.DeleteOne(ctx, filter)
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

func (*server) ListBlogs(req *blogpb.ListBlogsRequest, stream blogpb.BlogService_ListBlogsServer) error {
	log.Println("ListBlog: Invoked with no parameters")

	cur, err := collection.Find(context.Background(), bson.D{})
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

func main() {
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

	// Open MongoDB collection
	collection = client.Database("mydb").Collection("blog")

	// Connect to gRPC service
	log.Println("Connecting to gRPC service...")
	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	blogpb.RegisterBlogServiceServer(grpcServer, newServer())

	// Register reflection service on gRPC server
	reflection.Register(grpcServer)

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
	log.Println("Server shut down successfully.")
}
