protoc -I proto \
    --go_out=server/model/blogpb \
    --go_opt=paths=source_relative \
    --go-grpc_out=server/model/blogpb \
    --go-grpc_opt=paths=source_relative \
    proto/blog.proto