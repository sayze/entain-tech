// Package service implements the racing gRPC service.
package service

import (
	"errors"

	"git.neds.sh/matty/entain/racing/db"
	"git.neds.sh/matty/entain/racing/proto/racing"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Racing is the gRPC server interface for the racing service.
type Racing interface {
	// ListRaces will return a collection of races.
	ListRaces(ctx context.Context, in *racing.ListRacesRequest) (*racing.ListRacesResponse, error)
}

// racingService implements the Racing interface.
type racingService struct {
	racesRepo db.RacesRepo
}

// NewRacingService instantiates and returns a new racingService.
func NewRacingService(racesRepo db.RacesRepo) Racing {
	return &racingService{racesRepo}
}

// ListRaces returns the races that match the request filter, sorted by the
// request's order_by (advertised_start_time ascending by default). An invalid
// order_by is reported as codes.InvalidArgument.
func (s *racingService) ListRaces(_ context.Context, in *racing.ListRacesRequest) (*racing.ListRacesResponse, error) {
	races, err := s.racesRepo.List(in.GetFilter(), in.GetOrderBy())
	if err != nil {
		if errors.Is(err, db.ErrInvalidOrderBy) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		return nil, err
	}

	return &racing.ListRacesResponse{Races: races}, nil
}
