// Command sports runs the sports gRPC service.
package main

import (
	"database/sql"
	"flag"
	"log"
	"net"

	"google.golang.org/grpc"

	"git.neds.sh/matty/entain/sports/db"
	"git.neds.sh/matty/entain/sports/proto/sports"
	"git.neds.sh/matty/entain/sports/service"
)

var (
	grpcEndpoint = flag.String("grpc-endpoint", "localhost:9001", "gRPC server endpoint to listen on")
	dbPath       = flag.String("db", "./db/sports.db", "path to the SQLite database file (created and seeded if missing)")
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		log.Fatalf("failed running grpc server: %s\n", err)
	}
}

func run() error {
	conn, err := net.Listen("tcp", *grpcEndpoint)
	if err != nil {
		return err
	}

	sportsDB, err := sql.Open("sqlite3", *dbPath)
	if err != nil {
		return err
	}

	eventsRepo := db.NewEventsRepo(sportsDB)
	if err := eventsRepo.Init(); err != nil {
		return err
	}

	grpcServer := grpc.NewServer()

	sports.RegisterSportsServer(
		grpcServer,
		service.NewSportsService(
			eventsRepo,
		),
	)

	log.Printf("gRPC server listening on: %s\n", *grpcEndpoint)

	return grpcServer.Serve(conn)
}
