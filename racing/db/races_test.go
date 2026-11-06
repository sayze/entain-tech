package db

import (
	"database/sql"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"git.neds.sh/matty/entain/racing/proto/racing"
)

// newTestRepo returns a RacesRepo backed by an in-memory SQLite database
// holding a small, fixed set of races (instead of the random seed data).
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

	start := time.Date(2021, 2, 1, 10, 0, 0, 0, time.UTC).Format(time.RFC3339)
	fixtures := []struct {
		id, meetingID int64
		visible       bool
	}{
		{id: 1, meetingID: 1, visible: true},
		{id: 2, meetingID: 1, visible: false},
		{id: 3, meetingID: 2, visible: true},
		{id: 4, meetingID: 2, visible: false},
	}
	for _, f := range fixtures {
		if _, err := db.Exec(
			`INSERT INTO races(id, meeting_id, name, number, visible, advertised_start_time) VALUES (?,?,?,?,?,?)`,
			f.id, f.meetingID, "race", 1, f.visible, start,
		); err != nil {
			t.Fatalf("insert race %d: %v", f.id, err)
		}
	}

	return NewRacesRepo(db)
}

func TestRacesRepoList(t *testing.T) {
	tests := []struct {
		name    string
		filter  *racing.ListRacesRequestFilter
		wantIDs []int64
	}{
		{
			name:    "nil filter returns all races",
			filter:  nil,
			wantIDs: []int64{1, 2, 3, 4},
		},
		{
			name:    "unset visible returns all races",
			filter:  &racing.ListRacesRequestFilter{},
			wantIDs: []int64{1, 2, 3, 4},
		},
		{
			name:    "visible true returns only visible races",
			filter:  &racing.ListRacesRequestFilter{Visible: proto.Bool(true)},
			wantIDs: []int64{1, 3},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			races, err := newTestRepo(t).List(tt.filter)
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
