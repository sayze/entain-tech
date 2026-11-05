// Package db provides SQLite-backed repositories for sports data.
package db

import (
	"time"

	"syreclabs.com/go/faker"
)

// seedSports are the sports that seeded events are spread across.
var seedSports = []string{"Soccer", "Basketball", "Tennis", "Cricket", "Rugby League", "AFL"}

// seed creates the events table if needed and fills it with 100 dummy
// events, starting between a day ago and two days from now. Rows that
// already exist are left as they are (INSERT OR IGNORE), so re-running it
// against an existing database is safe.
func (r *eventsRepo) seed() error {
	if _, err := r.db.Exec(`CREATE TABLE IF NOT EXISTS events (id INTEGER PRIMARY KEY, competition_id INTEGER, name TEXT, sport TEXT, visible INTEGER, advertised_start_time DATETIME)`); err != nil {
		return err
	}

	// One transaction for all the inserts: much faster than 100 separate
	// commits, and either every event is seeded or none are.
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	// Rollback after a successful Commit is a no-op that returns
	// sql.ErrTxDone, so it is safe to defer unconditionally.
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO events(id, competition_id, name, sport, visible, advertised_start_time) VALUES (?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()

	for i := 1; i <= 100; i++ {
		if _, err := stmt.Exec(
			i,
			faker.Number().Between(1, 10),
			faker.Team().Name()+" v "+faker.Team().Name(),
			faker.RandomChoice(seedSports),
			faker.Number().Between(0, 1),
			faker.Time().Between(time.Now().AddDate(0, 0, -1), time.Now().AddDate(0, 0, 2)).Format(time.RFC3339),
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}
