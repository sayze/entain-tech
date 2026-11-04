.PHONY: racing api

# Build and start the racing gRPC service on :9000. It has to run from inside
# racing/ because it opens its SQLite file at the relative path ./db/racing.db.
racing:
	cd racing && go build && ./racing

# Build and start the REST gateway on localhost:8000. It proxies requests to
# the racing service, so start that first (e.g. `make racing` in another shell).
api:
	cd api && go build && ./api
