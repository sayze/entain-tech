package db

import (
	"database/sql"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	// Registers the "sqlite3" driver with database/sql.
	_ "github.com/mattn/go-sqlite3"

	"git.neds.sh/matty/entain/sports/proto/sports"
)

// EventsRepo provides repository access to sporting events.
type EventsRepo interface {
	// Init will initialise our events repository.
	Init() error

	// List will return the events that match filter, sorted by advertised
	// start time.
	List(filter *sports.ListEventsRequestFilter) ([]*sports.Event, error)
}

type eventsRepo struct {
	db   *sql.DB
	init sync.Once
}

// NewEventsRepo creates a new events repository.
func NewEventsRepo(db *sql.DB) EventsRepo {
	return &eventsRepo{db: db}
}

// Init prepares the events repository dummy data. It only seeds once, however
// many times it is called.
func (r *eventsRepo) Init() error {
	var err error

	r.init.Do(func() {
		// For test/example purposes, we seed the DB with some dummy events.
		err = r.seed()
	})

	return err
}

// List returns the events that match the given filter, soonest first.
func (r *eventsRepo) List(filter *sports.ListEventsRequestFilter) ([]*sports.Event, error) {
	query, args := applyFilter(getEventQueries()[eventsList], filter)
	query += eventsListOrder

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanEvents(rows)
}

// applyFilter appends a WHERE clause for each filter field that is set,
// joined with AND. Values are always passed as ? parameters, never spliced
// into the SQL.
func applyFilter(query string, filter *sports.ListEventsRequestFilter) (string, []interface{}) {
	var (
		clauses []string
		args    []interface{}
	)

	if filter == nil {
		return query, args
	}

	if len(filter.CompetitionIds) > 0 {
		clauses = append(clauses, "competition_id IN ("+placeholders(len(filter.CompetitionIds))+")")

		for _, id := range filter.CompetitionIds {
			args = append(args, id)
		}
	}

	if len(filter.Sports) > 0 {
		clauses = append(clauses, "sport IN ("+placeholders(len(filter.Sports))+")")

		for _, sport := range filter.Sports {
			args = append(args, sport)
		}
	}

	if len(clauses) != 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}

	return query, args
}

// placeholders returns n comma-separated ? placeholders, e.g. "?,?,?". n must
// be at least 1.
func placeholders(n int) string {
	return strings.Repeat("?,", n-1) + "?"
}

// scanEvents maps every row to an Event.
func scanEvents(rows *sql.Rows) ([]*sports.Event, error) {
	var events []*sports.Event

	for rows.Next() {
		var (
			event           sports.Event
			advertisedStart time.Time
		)

		if err := rows.Scan(&event.Id, &event.CompetitionId, &event.Name, &event.Sport, &event.Visible, &advertisedStart); err != nil {
			return nil, err
		}

		event.AdvertisedStartTime = timestamppb.New(advertisedStart)

		events = append(events, &event)
	}

	// rows.Next returns false on error as well as at the end of the results,
	// so check which it was.
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}
