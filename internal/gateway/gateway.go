package gateway

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/dnys1/grpc-mongo/internal/model/blogpb"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
)

var (
	gatewayPort = flag.Int("gateway-port", 8081, "Gateway port to serve on")
)

// Run starts the gateway server on the given port,
// connecting to the grpc server at the given endpoint.
func Run(grpcEndpoint string) error {
	log.Printf("Starting gateway server on port %d...", *gatewayPort)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := blogpb.RegisterBlogServiceHandlerFromEndpoint(ctx, mux, grpcEndpoint, opts)
	if err != nil {
		return fmt.Errorf("Error registering reverse proxy: %v", err)
	}

	addr := fmt.Sprintf(":%d", *gatewayPort)
	if err := http.ListenAndServe(addr, mux); err != nil {
		return fmt.Errorf("Failed to serve reverse proxy: %v", err)
	}

	return nil
}
