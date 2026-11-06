// Command api runs the REST gateway that proxies HTTP requests to the
// backend gRPC services.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"

	"git.neds.sh/matty/entain/api/proto/racing"
	"git.neds.sh/matty/entain/api/proto/sports"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

var (
	apiEndpoint  = flag.String("api-endpoint", "localhost:8000", "API endpoint")
	grpcEndpoint = flag.String("grpc-endpoint", "localhost:9000", "racing gRPC server endpoint")
	// The racing flag keeps its original, generic name so existing callers
	// are unaffected; each further backend gets its own named flag.
	sportsEndpoint = flag.String("sports-grpc-endpoint", "localhost:9001", "sports gRPC server endpoint")
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		log.Printf("failed running api server: %s\n", err)
	}
}

func run() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Every backend service registers its REST routes on the same mux. The
	// gateway dials each one lazily, so the API starts even if a backend is
	// down; requests to that backend fail until it comes up.
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}

	if err := racing.RegisterRacingHandlerFromEndpoint(ctx, mux, *grpcEndpoint, opts); err != nil {
		return err
	}

	if err := sports.RegisterSportsHandlerFromEndpoint(ctx, mux, *sportsEndpoint, opts); err != nil {
		return err
	}

	log.Printf("API server listening on: %s\n", *apiEndpoint)

	return http.ListenAndServe(*apiEndpoint, mux)
}
