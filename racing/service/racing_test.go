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

// fakeRacesRepo is a db.RacesRepo stub that records the arguments it was
// called with and returns canned results. err is returned by both List and
// Get.
type fakeRacesRepo struct {
	races []*racing.Race // returned by List
	race  *racing.Race   // returned by Get
	err   error

	gotFilter  *racing.ListRacesRequestFilter
	gotOrderBy string
	gotID      int64
}

func (f *fakeRacesRepo) Init() error { return nil }

func (f *fakeRacesRepo) List(filter *racing.ListRacesRequestFilter, orderBy string) ([]*racing.Race, error) {
	f.gotFilter, f.gotOrderBy = filter, orderBy
	return f.races, f.err
}

func (f *fakeRacesRepo) Get(id int64) (*racing.Race, error) {
	f.gotID = id
	return f.race, f.err
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

func TestGetRace(t *testing.T) {
	want := &racing.Race{Id: 42, Name: "Alpha"}
	repo := &fakeRacesRepo{race: want}

	got, err := NewRacingService(repo).GetRace(context.Background(), &racing.GetRaceRequest{Id: 42})
	if err != nil {
		t.Fatalf("GetRace: %v", err)
	}

	if repo.gotID != 42 {
		t.Fatalf("repo got id %d, want 42", repo.gotID)
	}
	if got != want {
		t.Fatalf("got race %v, want the repo's race %v", got, want)
	}
}

func TestGetRaceErrors(t *testing.T) {
	tests := []struct {
		name     string
		repoErr  error
		wantCode codes.Code
	}{
		{
			name:     "missing race is NotFound",
			repoErr:  db.ErrNotFound,
			wantCode: codes.NotFound,
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
				GetRace(context.Background(), &racing.GetRaceRequest{Id: 1})

			if got := status.Code(err); got != tt.wantCode {
				t.Fatalf("got code %v (err %v), want %v", got, err, tt.wantCode)
			}
		})
	}
}
