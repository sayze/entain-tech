package db

const (
	racesList = "list"
)

// raceByIDClause narrows the races list query down to a single race.
const raceByIDClause = " WHERE id = ?"

func getRaceQueries() map[string]string {
	return map[string]string{
		racesList: `
			SELECT 
				id, 
				meeting_id, 
				name, 
				number, 
				visible, 
				advertised_start_time 
			FROM races
		`,
	}
}
