protoc -I proto \
    --go_out=internal/model/blogpb \
    --go_opt=paths=source_relative \
    --go-grpc_out=internal/model/blogpb \
    --go-grpc_opt=paths=source_relative \
    --grpc-gateway_out=logtostderr=true,paths=source_relative:internal/model/blogpb \
    proto/blog.proto