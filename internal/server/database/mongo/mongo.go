package database

import (
	"context"
	"fmt"
	"time"

	"github.com/dnys1/grpc-mongo/internal/model/blogpb"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	errOidConvert = errors.New("Error converting id to type primitive.ObjectID")
)

// MongoDatabase represents a Database object which
// connects to a MongoDB client at the specified host
// and port, along with other connection options.
type MongoDatabase struct {
	Options    *MongoDatabaseOptions
	client     *mongo.Client
	collection *mongo.Collection
}

// MongoDatabaseOptions specifies the options for
// connecting to a MongoDatabase.
type MongoDatabaseOptions struct {
	Host string
	Port int
}

// A mapping of a blog item to MongoDB types
type blogItem struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	AuthorID string             `bson:"author_id,omitempty"`
	Title    string             `bson:"title,omitempty"`
	Content  string             `bson:"content,omitempty"`
}

// New creates a new MongoDatabase with the specified options.
func New(opts *MongoDatabaseOptions) (*MongoDatabase, error) {
	db := &MongoDatabase{
		Options: opts,
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(db.Endpoint()).SetConnectTimeout(30 * time.Second))
	if err != nil {
		return nil, errors.Wrap(err, "Error instantiating MongoDB client")
	}
	db.client = client
	db.collection = client.Database("mydb").Collection("blog")
	return db, nil
}

// Endpoint returns the endpoint of the database.
func (db *MongoDatabase) Endpoint() string {
	return fmt.Sprintf("mongodb://%s:%d", db.Options.Host, db.Options.Port)
}

// Connect connects to and pings the MongoDatabase.
func (db *MongoDatabase) Connect(ctx context.Context) error {
	// Connect to MongoDB client
	if err := db.client.Connect(ctx); err != nil {
		return errors.Wrap(err, "Error connecting to MongoDB instance")
	}

	// Ping the MongoDB server
	if err := db.client.Ping(ctx, readpref.Primary()); err != nil {
		return errors.Wrap(err, "Error pinging the MongoDB instance")
	}

	return nil
}

// Disconnect disconnects from the MongoDatabase.
//
// This function should be called during takedown of services.
func (db *MongoDatabase) Disconnect(ctx context.Context) error {
	if err := db.client.Disconnect(ctx); err != nil {
		return errors.Wrap(err, "Error closing the MongoDB connection")
	}

	return nil
}

// CreateBlog creates a blog in the database
func (db *MongoDatabase) CreateBlog(ctx context.Context, blog *blogpb.Blog) (*blogpb.Blog, error) {
	data := blogItem{
		AuthorID: blog.GetAuthorId(),
		Title:    blog.GetTitle(),
		Content:  blog.GetContent(),
	}

	res, err := db.collection.InsertOne(ctx, data)
	if err != nil {
		return nil, err
	}

	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, errOidConvert
	}

	return &blogpb.Blog{
		Id:       oid.Hex(),
		AuthorId: blog.GetAuthorId(),
		Title:    blog.GetTitle(),
		Content:  blog.GetContent(),
	}, nil
}

// ReadBlog reads a user from the database
func (db *MongoDatabase) ReadBlog(ctx context.Context, id string) (*blogpb.Blog, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	data := &blogItem{}
	filter := bson.M{"_id": oid}

	doc := db.collection.FindOne(ctx, filter)
	if err := doc.Decode(data); err != nil {
		return nil, err
	}

	return &blogpb.Blog{
		Id:       id,
		AuthorId: data.AuthorID,
		Title:    data.Title,
		Content:  data.Content,
	}, nil
}

// UpdateBlog updates a blog in the database.
func (db *MongoDatabase) UpdateBlog(ctx context.Context, blog *blogpb.Blog) (blogpb.UpdateBlogResponse_UpdateStatus, error) {
	id := blog.GetId()
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return blogpb.UpdateBlogResponse_NOT_UPDATED, err
	}

	data := &blogItem{}
	filter := bson.M{"_id": oid}

	// Get the old doc from the DB
	doc := db.collection.FindOne(ctx, filter)
	if err := doc.Decode(data); err != nil {
		return blogpb.UpdateBlogResponse_NOT_UPDATED, err
	}

	// Update variables on the document
	data.AuthorID = blog.GetAuthorId()
	data.Title = blog.GetTitle()
	data.Content = blog.GetContent()

	_, err = db.collection.ReplaceOne(ctx, filter, data)
	if err != nil {
		return blogpb.UpdateBlogResponse_NOT_UPDATED, err
	}

	return blogpb.UpdateBlogResponse_UPDATED, nil
}

// DeleteBlog deletes a blog from the database
func (db *MongoDatabase) DeleteBlog(ctx context.Context, id string) (blogpb.DeleteBlogResponse_DeleteStatus, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return blogpb.DeleteBlogResponse_NOT_DELETED, err
	}

	filter := bson.M{"_id": oid}

	res, err := db.collection.DeleteOne(ctx, filter)
	if err != nil {
		return blogpb.DeleteBlogResponse_NOT_DELETED, err
	}

	if res.DeletedCount == 0 {
		return blogpb.DeleteBlogResponse_NOT_DELETED, nil
	}

	return blogpb.DeleteBlogResponse_DELETED, nil
}

// ListBlogs lists all the blogs in the database.
func (db *MongoDatabase) ListBlogs(stream blogpb.BlogService_ListBlogsServer) error {
	cur, err := db.collection.Find(context.Background(), bson.D{})
	if err != nil {
		return err
	}
	defer cur.Close(context.Background())

	for cur.Next(context.Background()) {
		data := &blogItem{}
		if err := cur.Decode(data); err != nil {
			return err
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
		return err
	}

	return nil
}
