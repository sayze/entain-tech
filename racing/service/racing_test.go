package service

import (
	"errors"
	"fmt"
	"testing"

	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"git.neds.sh/matty/entain/racing/db"
	"git.neds.sh/matty/entain/racing/proto/racing"
)

// fakeRacesRepo is a db.RacesRepo stub that records the arguments List was
// called with and returns canned results.
type fakeRacesRepo struct {
	races []*racing.Race
	err   error

	gotFilter  *racing.ListRacesRequestFilter
	gotOrderBy string
}

func (f *fakeRacesRepo) Init() error { return nil }

func (f *fakeRacesRepo) List(filter *racing.ListRacesRequestFilter, orderBy string) ([]*racing.Race, error) {
	f.gotFilter, f.gotOrderBy = filter, orderBy
	return f.races, f.err
}

func TestListRacesPassesRequestToRepo(t *testing.T) {
	repo := &fakeRacesRepo{races: []*racing.Race{{Id: 7}}}
	filter := &racing.ListRacesRequestFilter{MeetingIds: []int64{1}}

	resp, err := NewRacingService(repo).ListRaces(context.Background(), &racing.ListRacesRequest{
		Filter:  filter,
		OrderBy: "name desc",
	})
	if err != nil {
		t.Fatalf("ListRaces: %v", err)
	}

	if repo.gotFilter != filter || repo.gotOrderBy != "name desc" {
		t.Fatalf("repo got filter %v, order_by %q", repo.gotFilter, repo.gotOrderBy)
	}
	if len(resp.Races) != 1 || resp.Races[0].Id != 7 {
		t.Fatalf("got races %v, want the repo's races", resp.Races)
	}
}

func TestListRacesErrors(t *testing.T) {
	tests := []struct {
		name     string
		repoErr  error
		wantCode codes.Code
	}{
		{
			name:     "invalid order_by is InvalidArgument",
			repoErr:  fmt.Errorf("%w: unknown field %q", db.ErrInvalidOrderBy, "venue"),
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "other repo errors are passed through as Unknown",
			repoErr:  errors.New("disk on fire"),
			wantCode: codes.Unknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewRacingService(&fakeRacesRepo{err: tt.repoErr}).
				ListRaces(context.Background(), &racing.ListRacesRequest{})

			if got := status.Code(err); got != tt.wantCode {
				t.Fatalf("got code %v (err %v), want %v", got, err, tt.wantCode)
			}
		})
	}
}
