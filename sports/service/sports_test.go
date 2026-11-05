package service

import (
	"errors"
	"testing"

	"golang.org/x/net/context"

	"git.neds.sh/matty/entain/sports/proto/sports"
)

// fakeEventsRepo is a db.EventsRepo stub that records the filter List was
// called with and returns canned results.
type fakeEventsRepo struct {
	events []*sports.Event
	err    error

	gotFilter *sports.ListEventsRequestFilter
}

func (f *fakeEventsRepo) Init() error { return nil }

func (f *fakeEventsRepo) List(filter *sports.ListEventsRequestFilter) ([]*sports.Event, error) {
	f.gotFilter = filter
	return f.events, f.err
}

func TestListEvents(t *testing.T) {
	repo := &fakeEventsRepo{events: []*sports.Event{{Id: 7}}}
	filter := &sports.ListEventsRequestFilter{Sports: []string{"Soccer"}}

	resp, err := NewSportsService(repo).ListEvents(context.Background(), &sports.ListEventsRequest{Filter: filter})
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}

	if repo.gotFilter != filter {
		t.Fatalf("repo got filter %v, want %v", repo.gotFilter, filter)
	}
	if len(resp.Events) != 1 || resp.Events[0].Id != 7 {
		t.Fatalf("got events %v, want the repo's events", resp.Events)
	}
}

func TestListEventsRepoError(t *testing.T) {
	repoErr := errors.New("disk on fire")

	_, err := NewSportsService(&fakeEventsRepo{err: repoErr}).
		ListEvents(context.Background(), &sports.ListEventsRequest{})

	if !errors.Is(err, repoErr) {
		t.Fatalf("got err %v, want %v", err, repoErr)
	}
}
