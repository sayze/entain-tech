package db

const (
	eventsList = "list"
)

func getEventQueries() map[string]string {
	return map[string]string{
		eventsList: `
			SELECT
				id,
				competition_id,
				name,
				sport,
				visible,
				advertised_start_time
			FROM events
		`,
	}
}

// eventsListOrder sorts listed events by start time (soonest first), with id
// as a tie-breaker so the order is always stable.
const eventsListOrder = " ORDER BY advertised_start_time ASC, id ASC"
