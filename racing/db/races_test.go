package db

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"git.neds.sh/matty/entain/racing/proto/racing"
)

// fixtureBase is the advertised start time of the earliest fixture race.
var fixtureBase = time.Date(2021, 2, 1, 10, 0, 0, 0, time.UTC)

// fixtureNow is the fixed "current time" of the test repo's clock. Races 2
// and 4 start exactly at this instant, race 3 before it and race 1 after it.
var fixtureNow = fixtureBase.Add(1 * time.Hour)

// newTestRepo returns a RacesRepo backed by an in-memory SQLite database
// holding a small, fixed set of races (instead of the random seed data). Its
// clock is frozen at fixtureNow so derived statuses are deterministic.
func newTestRepo(t *testing.T) RacesRepo {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	// Every connection to ":memory:" gets its own empty database, so pin the
	// pool to a single connection to keep the fixtures visible to List.
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`CREATE TABLE races (id INTEGER PRIMARY KEY, meeting_id INTEGER, name TEXT, number INTEGER, visible INTEGER, advertised_start_time DATETIME)`); err != nil {
		t.Fatalf("create table: %v", err)
	}

	// Start times are deliberately not in id order, so tests can tell the
	// default advertised_start_time sort apart from insertion order. Races
	// 2 and 4 share a start time to exercise the id tie-breaker.
	fixtures := []struct {
		id, meetingID int64
		name          string
		visible       bool
		startOffset   time.Duration
	}{
		{id: 1, meetingID: 1, name: "Charlie", visible: true, startOffset: 3 * time.Hour},
		{id: 2, meetingID: 1, name: "Alpha", visible: false, startOffset: 1 * time.Hour},
		{id: 3, meetingID: 2, name: "Delta", visible: true, startOffset: 0},
		{id: 4, meetingID: 2, name: "Bravo", visible: false, startOffset: 1 * time.Hour},
	}
	for _, f := range fixtures {
		if _, err := db.Exec(
			`INSERT INTO races(id, meeting_id, name, number, visible, advertised_start_time) VALUES (?,?,?,?,?,?)`,
			f.id, f.meetingID, f.name, 1, f.visible, fixtureBase.Add(f.startOffset).Format(time.RFC3339),
		); err != nil {
			t.Fatalf("insert race %d: %v", f.id, err)
		}
	}

	return &racesRepo{db: db, now: func() time.Time { return fixtureNow }}
}

func TestRacesRepoList(t *testing.T) {
	tests := []struct {
		name    string
		filter  *racing.ListRacesRequestFilter
		orderBy string
		wantIDs []int64
	}{
		// Filtering. Results use the default order: advertised_start_time
		// ascending, then id.
		{
			name:    "nil filter returns all races",
			filter:  nil,
			wantIDs: []int64{3, 2, 4, 1},
		},
		{
			name:    "unset visible returns all races",
			filter:  &racing.ListRacesRequestFilter{},
			wantIDs: []int64{3, 2, 4, 1},
		},
		{
			name:    "visible true returns only visible races",
			filter:  &racing.ListRacesRequestFilter{Visible: proto.Bool(true)},
			wantIDs: []int64{3, 1},
		},
		{
			name:    "visible false returns only hidden races",
			filter:  &racing.ListRacesRequestFilter{Visible: proto.Bool(false)},
			wantIDs: []int64{2, 4},
		},
		{
			name:    "meeting ids only",
			filter:  &racing.ListRacesRequestFilter{MeetingIds: []int64{2}},
			wantIDs: []int64{3, 4},
		},
		{
			name: "visible combined with meeting ids",
			filter: &racing.ListRacesRequestFilter{
				MeetingIds: []int64{1},
				Visible:    proto.Bool(true),
			},
			wantIDs: []int64{1},
		},

		// Ordering.
		{
			name:    "whitespace-only order_by uses the default",
			orderBy: "   ",
			wantIDs: []int64{3, 2, 4, 1},
		},
		{
			name:    "advertised_start_time explicitly ascending",
			orderBy: "advertised_start_time asc",
			wantIDs: []int64{3, 2, 4, 1},
		},
		{
			name:    "advertised_start_time descending keeps id tie-breaker ascending",
			orderBy: "advertised_start_time desc",
			wantIDs: []int64{1, 2, 4, 3},
		},
		{
			name:    "name ascending",
			orderBy: "name",
			wantIDs: []int64{2, 4, 1, 3},
		},
		{
			name:    "direction is case insensitive",
			orderBy: "name DESC",
			wantIDs: []int64{3, 1, 4, 2},
		},
		{
			name:    "multiple fields",
			orderBy: "meeting_id desc, name",
			wantIDs: []int64{4, 3, 2, 1},
		},
		{
			name:    "explicit id order replaces tie-breaker",
			orderBy: "advertised_start_time, id desc",
			wantIDs: []int64{3, 4, 2, 1},
		},
		{
			name:    "ordering combined with filter",
			filter:  &racing.ListRacesRequestFilter{Visible: proto.Bool(true)},
			orderBy: "advertised_start_time desc",
			wantIDs: []int64{1, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			races, err := newTestRepo(t).List(tt.filter, tt.orderBy)
			if err != nil {
				t.Fatalf("List: %v", err)
			}

			gotIDs := make([]int64, 0, len(races))
			for _, r := range races {
				gotIDs = append(gotIDs, r.Id)
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

func TestRacesRepoListInvalidOrderBy(t *testing.T) {
	tests := []struct {
		name    string
		orderBy string
	}{
		{name: "unknown field", orderBy: "venue"},
		{name: "sql injection attempt", orderBy: "name; DROP TABLE races"},
		{name: "unknown direction", orderBy: "name sideways"},
		{name: "too many words", orderBy: "name asc desc"},
		{name: "empty term", orderBy: "name,,number"},
		{name: "trailing comma", orderBy: "name,"},
		{name: "duplicate field", orderBy: "name, name desc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newTestRepo(t).List(nil, tt.orderBy)
			if !errors.Is(err, ErrInvalidOrderBy) {
				t.Fatalf("got err %v, want ErrInvalidOrderBy", err)
			}
		})
	}
}

func TestRacesRepoListStatus(t *testing.T) {
	races, err := newTestRepo(t).List(nil, "id")
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	// See fixtureNow: a race starting exactly now counts as CLOSED.
	want := map[int64]racing.Race_Status{
		1: racing.Race_OPEN,
		2: racing.Race_CLOSED,
		3: racing.Race_CLOSED,
		4: racing.Race_CLOSED,
	}
	if len(races) != len(want) {
		t.Fatalf("got %d races, want %d", len(races), len(want))
	}
	for _, r := range races {
		if r.Status != want[r.Id] {
			t.Errorf("race %d: got status %v, want %v", r.Id, r.Status, want[r.Id])
		}
	}
}

func TestRaceStatus(t *testing.T) {
	now := time.Date(2021, 2, 1, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		start time.Time
		want  racing.Race_Status
	}{
		{name: "future start is open", start: now.Add(time.Nanosecond), want: racing.Race_OPEN},
		{name: "start exactly now is closed", start: now, want: racing.Race_CLOSED},
		{name: "past start is closed", start: now.Add(-time.Nanosecond), want: racing.Race_CLOSED},
		{name: "same instant in another zone is closed", start: now.In(time.FixedZone("AEST", 10*60*60)), want: racing.Race_CLOSED},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := raceStatus(tt.start, now); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRacesRepoGet(t *testing.T) {
	race, err := newTestRepo(t).Get(1)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	// Check every column is mapped, not just the ID, and that the derived
	// status is filled in just as it is for List.
	if race.Id != 1 || race.MeetingId != 1 || race.Name != "Charlie" || race.Number != 1 || !race.Visible {
		t.Fatalf("got race %+v, want race 1 (meeting 1, Charlie, number 1, visible)", race)
	}
	if want := fixtureBase.Add(3 * time.Hour); !race.AdvertisedStartTime.AsTime().Equal(want) {
		t.Fatalf("got advertised start time %v, want %v", race.AdvertisedStartTime.AsTime(), want)
	}
	if race.Status != racing.Race_OPEN {
		t.Fatalf("got status %v, want OPEN", race.Status)
	}
}

func TestRacesRepoGetNotFound(t *testing.T) {
	for _, id := range []int64{0, -1, 5} {
		if _, err := newTestRepo(t).Get(id); !errors.Is(err, ErrNotFound) {
			t.Errorf("Get(%d): got err %v, want ErrNotFound", id, err)
		}
	}
}
