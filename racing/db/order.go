package db

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidOrderBy is returned (wrapped) when an order_by value cannot be
// parsed or names a field that cannot be sorted on. Callers can detect it
// with errors.Is to report a client error rather than an internal one.
var ErrInvalidOrderBy = errors.New("invalid order_by")

// defaultRaceOrderBy is the sort order used when the caller gives none.
const defaultRaceOrderBy = "advertised_start_time"

// raceSortColumns maps the API field names that may appear in order_by to
// their SQL columns. Only names in this allowlist ever reach the query, which
// is what keeps the ORDER BY clause safe from SQL injection: column names
// cannot be bound as ? parameters.
var raceSortColumns = map[string]string{
	"id":                    "id",
	"meeting_id":            "meeting_id",
	"name":                  "name",
	"number":                "number",
	"visible":               "visible",
	"advertised_start_time": "advertised_start_time",
}

// applyOrder appends an ORDER BY clause built from orderBy to query.
//
// orderBy follows the Google API design guide (AIP-132): a comma-separated
// list of field names, each optionally followed by "asc" or "desc", e.g.
// "advertised_start_time desc, name". An empty value falls back to
// defaultRaceOrderBy. The id column is always appended as a final tie-breaker
// (unless already present) so results come back in a stable order.
func applyOrder(query, orderBy string) (string, error) {
	if strings.TrimSpace(orderBy) == "" {
		orderBy = defaultRaceOrderBy
	}

	var (
		terms   []string
		seen    = make(map[string]bool)
		fields  = strings.Split(orderBy, ",")
		hasIDBy bool
	)

	for _, field := range fields {
		parts := strings.Fields(field)
		if len(parts) == 0 || len(parts) > 2 {
			return "", fmt.Errorf("%w: malformed term %q", ErrInvalidOrderBy, strings.TrimSpace(field))
		}

		column, ok := raceSortColumns[parts[0]]
		if !ok {
			return "", fmt.Errorf("%w: unknown field %q", ErrInvalidOrderBy, parts[0])
		}

		if seen[column] {
			return "", fmt.Errorf("%w: field %q listed more than once", ErrInvalidOrderBy, parts[0])
		}
		seen[column] = true

		direction := "ASC"
		if len(parts) == 2 {
			switch strings.ToLower(parts[1]) {
			case "asc":
			case "desc":
				direction = "DESC"
			default:
				return "", fmt.Errorf("%w: unknown direction %q for field %q", ErrInvalidOrderBy, parts[1], parts[0])
			}
		}

		if column == "id" {
			hasIDBy = true
		}

		terms = append(terms, column+" "+direction)
	}

	if !hasIDBy {
		terms = append(terms, "id ASC")
	}

	return query + " ORDER BY " + strings.Join(terms, ", "), nil
}
