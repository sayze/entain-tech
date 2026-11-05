.PHONY: racing sports api

# Build and start the racing gRPC service on :9000. It has to run from inside
# racing/ because it opens its SQLite file at the relative path ./db/racing.db.
racing:
	cd racing && go build && ./racing

# Build and start the sports gRPC service on localhost:9001. On first run it
# creates and seeds its SQLite database at sports/db/sports.db.
sports:
	cd sports && go build && ./sports

# Build and start the REST gateway on localhost:8000. It proxies requests to
# the racing and sports services, so start those first (e.g. `make racing` and
# `make sports` in other shells).
api:
	cd api && go build && ./api
