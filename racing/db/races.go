package db

import (
	"database/sql"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes"
	// Registers the "sqlite3" driver with database/sql.
	_ "github.com/mattn/go-sqlite3"

	"git.neds.sh/matty/entain/racing/proto/racing"
)

// RacesRepo provides repository access to races.
type RacesRepo interface {
	// Init will initialise our races repository.
	Init() error

	// List will return a list of races that match filter, sorted according
	// to orderBy (see applyOrder for the syntax). It returns an error
	// wrapping ErrInvalidOrderBy if orderBy is not valid.
	List(filter *racing.ListRacesRequestFilter, orderBy string) ([]*racing.Race, error)

	// Get will return the race with the given ID. It returns ErrNotFound
	// if there is no such race.
	Get(id int64) (*racing.Race, error)
}

// ErrNotFound is returned when a requested race does not exist.
var ErrNotFound = errors.New("race not found")

type racesRepo struct {
	db   *sql.DB
	init sync.Once
	// now returns the current time. It is used to derive each race's status
	// and is a field so tests can substitute a fixed clock.
	now func() time.Time
}

// NewRacesRepo creates a new races repository.
func NewRacesRepo(db *sql.DB) RacesRepo {
	return &racesRepo{db: db, now: time.Now}
}

// Init prepares the race repository dummy data.
func (r *racesRepo) Init() error {
	var err error

	r.init.Do(func() {
		// For test/example purposes, we seed the DB with some dummy races.
		err = r.seed()
	})

	return err
}

// List returns the races that match the given filter, in orderBy order.
func (r *racesRepo) List(filter *racing.ListRacesRequestFilter, orderBy string) ([]*racing.Race, error) {
	var (
		err   error
		query string
		args  []interface{}
	)

	query = getRaceQueries()[racesList]

	query, args = r.applyFilter(query, filter)

	query, err = applyOrder(query, orderBy)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	return r.scanRaces(rows, r.now())
}

// Get returns the race with the given ID, or ErrNotFound if there is none.
func (r *racesRepo) Get(id int64) (*racing.Race, error) {
	// Reuse the list query and row scanning so a race fetched on its own
	// always looks exactly like the same race in a list.
	rows, err := r.db.Query(getRaceQueries()[racesList]+raceByIDClause, id)
	if err != nil {
		return nil, err
	}

	races, err := r.scanRaces(rows, r.now())
	if err != nil {
		return nil, err
	}

	if len(races) == 0 {
		return nil, ErrNotFound
	}

	return races[0], nil
}

func (r *racesRepo) applyFilter(query string, filter *racing.ListRacesRequestFilter) (string, []interface{}) {
	var (
		clauses []string
		args    []interface{}
	)

	if filter == nil {
		return query, args
	}

	if len(filter.MeetingIds) > 0 {
		clauses = append(clauses, "meeting_id IN ("+strings.Repeat("?,", len(filter.MeetingIds)-1)+"?)")

		for _, meetingID := range filter.MeetingIds {
			args = append(args, meetingID)
		}
	}

	// Visible is optional: nil means "any visibility", so only filter on it
	// when the caller has set it explicitly (true or false).
	if filter.Visible != nil {
		clauses = append(clauses, "visible = ?")
		args = append(args, filter.GetVisible())
	}

	if len(clauses) != 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}

	return query, args
}

// scanRaces maps rows to races. Each race's status is derived by comparing
// its advertised start time with now, which the caller reads once so that
// every race in a single response is judged against the same instant.
func (r *racesRepo) scanRaces(
	rows *sql.Rows,
	now time.Time,
) ([]*racing.Race, error) {
	var races []*racing.Race

	for rows.Next() {
		var race racing.Race
		var advertisedStart time.Time

		if err := rows.Scan(&race.Id, &race.MeetingId, &race.Name, &race.Number, &race.Visible, &advertisedStart); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}

			return nil, err
		}

		ts, err := ptypes.TimestampProto(advertisedStart)
		if err != nil {
			return nil, err
		}

		race.AdvertisedStartTime = ts
		race.Status = raceStatus(advertisedStart, now)

		races = append(races, &race)
	}

	return races, nil
}

// raceStatus derives a race's status from its advertised start time. A race
// is CLOSED from the moment its advertised start time is reached, and OPEN
// before that.
func raceStatus(advertisedStart, now time.Time) racing.Race_Status {
	if advertisedStart.After(now) {
		return racing.Race_OPEN
	}

	return racing.Race_CLOSED
}
