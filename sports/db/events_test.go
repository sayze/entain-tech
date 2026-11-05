package db

import (
	"database/sql"
	"testing"
	"time"

	"git.neds.sh/matty/entain/sports/proto/sports"
)

// fixtureBase is the advertised start time of the earliest fixture event.
var fixtureBase = time.Date(2021, 2, 1, 10, 0, 0, 0, time.UTC)

// openTestDB returns an empty in-memory SQLite database.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	// Every connection to ":memory:" gets its own empty database, so pin the
	// pool to a single connection to keep the data visible to every query.
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	return db
}

// newTestRepo returns an EventsRepo holding a small, fixed set of events
// (instead of the random seed data).
func newTestRepo(t *testing.T) EventsRepo {
	t.Helper()

	db := openTestDB(t)

	if _, err := db.Exec(`CREATE TABLE events (id INTEGER PRIMARY KEY, competition_id INTEGER, name TEXT, sport TEXT, visible INTEGER, advertised_start_time DATETIME)`); err != nil {
		t.Fatalf("create table: %v", err)
	}

	// Start times are deliberately not in id order, so tests can tell the
	// advertised_start_time sort apart from insertion order. Events 2 and 4
	// share a start time to exercise the id tie-breaker.
	fixtures := []struct {
		id, competitionID int64
		sport             string
		startOffset       time.Duration
	}{
		{id: 1, competitionID: 1, sport: "Soccer", startOffset: 3 * time.Hour},
		{id: 2, competitionID: 1, sport: "Tennis", startOffset: 1 * time.Hour},
		{id: 3, competitionID: 2, sport: "Soccer", startOffset: 0},
		{id: 4, competitionID: 2, sport: "Cricket", startOffset: 1 * time.Hour},
	}
	for _, f := range fixtures {
		if _, err := db.Exec(
			`INSERT INTO events(id, competition_id, name, sport, visible, advertised_start_time) VALUES (?,?,?,?,?,?)`,
			f.id, f.competitionID, "Lions v Tigers", f.sport, true, fixtureBase.Add(f.startOffset).Format(time.RFC3339),
		); err != nil {
			t.Fatalf("insert event %d: %v", f.id, err)
		}
	}

	return NewEventsRepo(db)
}

func TestEventsRepoList(t *testing.T) {
	tests := []struct {
		name    string
		filter  *sports.ListEventsRequestFilter
		wantIDs []int64
	}{
		{
			name:    "nil filter returns all events, soonest first",
			filter:  nil,
			wantIDs: []int64{3, 2, 4, 1},
		},
		{
			name:    "empty filter returns all events",
			filter:  &sports.ListEventsRequestFilter{},
			wantIDs: []int64{3, 2, 4, 1},
		},
		{
			name:    "competition ids",
			filter:  &sports.ListEventsRequestFilter{CompetitionIds: []int64{1}},
			wantIDs: []int64{2, 1},
		},
		{
			name:    "several sports",
			filter:  &sports.ListEventsRequestFilter{Sports: []string{"Soccer", "Cricket"}},
			wantIDs: []int64{3, 4, 1},
		},
		{
			name:    "sport matching is case-sensitive",
			filter:  &sports.ListEventsRequestFilter{Sports: []string{"soccer"}},
			wantIDs: []int64{},
		},
		{
			name: "competition ids and sports combine with AND",
			filter: &sports.ListEventsRequestFilter{
				CompetitionIds: []int64{2},
				Sports:         []string{"Soccer"},
			},
			wantIDs: []int64{3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events, err := newTestRepo(t).List(tt.filter)
			if err != nil {
				t.Fatalf("List: %v", err)
			}

			gotIDs := make([]int64, 0, len(events))
			for _, e := range events {
				gotIDs = append(gotIDs, e.Id)
			}

			if len(gotIDs) != len(tt.wantIDs) {
				t.Fatalf("got ids %v, want %v", gotIDs, tt.wantIDs)
			}
			for i := range gotIDs {
				if gotIDs[i] != tt.wantIDs[i] {
					t.Fatalf("got ids %v, want %v", gotIDs, tt.wantIDs)
				}
			}
		})
	}
}

func TestEventsRepoListMapsAllFields(t *testing.T) {
	events, err := newTestRepo(t).List(&sports.ListEventsRequestFilter{Sports: []string{"Tennis"}})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}

	e := events[0]
	if e.Id != 2 || e.CompetitionId != 1 || e.Name != "Lions v Tigers" || e.Sport != "Tennis" || !e.Visible {
		t.Fatalf("got event %+v, want event 2 (competition 1, Lions v Tigers, Tennis, visible)", e)
	}
	if got, want := e.AdvertisedStartTime.AsTime(), fixtureBase.Add(time.Hour); !got.Equal(want) {
		t.Fatalf("got advertised start time %v, want %v", got, want)
	}
}

func TestEventsRepoInitSeeds(t *testing.T) {
	repo := NewEventsRepo(openTestDB(t))

	// Init must be safe to call more than once and only seed the first time.
	for i := 0; i < 2; i++ {
		if err := repo.Init(); err != nil {
			t.Fatalf("Init #%d: %v", i+1, err)
		}
	}

	events, err := repo.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(events) != 100 {
		t.Fatalf("got %d seeded events, want 100", len(events))
	}

	for _, e := range events {
		if e.Name == "" || e.Sport == "" || e.AdvertisedStartTime == nil {
			t.Fatalf("seeded event has missing fields: %+v", e)
		}
	}
}
