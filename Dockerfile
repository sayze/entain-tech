FROM golang:1.24-alpine AS build

WORKDIR /src

# racing and sports use github.com/mattn/go-sqlite3, which needs cgo, so we need a C
# toolchain here and link statically so the binary runs on scratch.
RUN apk add --no-cache ca-certificates build-base

# racing, sports and api are separate modules, so each downloads its own deps.
COPY racing/go.mod racing/go.sum ./racing/
COPY sports/go.mod sports/go.sum ./sports/
COPY api/go.mod api/go.sum ./api/
RUN cd racing && go mod download
RUN cd sports && go mod download
RUN cd api && go mod download

COPY racing ./racing
COPY sports ./sports
COPY api ./api

RUN cd racing && CGO_ENABLED=1 go build -ldflags '-s -w -extldflags "-static"' -o /out/racing .
RUN cd sports && CGO_ENABLED=1 go build -ldflags '-s -w -extldflags "-static"' -o /out/sports .
RUN cd api && CGO_ENABLED=0 go build -ldflags '-s -w' -o /out/api .

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/racing /usr/local/bin/racing
COPY --from=build /out/sports /usr/local/bin/sports
COPY --from=build /out/api /usr/local/bin/api

# racing opens its SQLite file at the relative path ./db/racing.db, so the
# working directory has to hold db/racing.db. sports creates and seeds
# ./db/sports.db on first run, in the same db directory.
WORKDIR /app
COPY --from=build /src/racing/db/racing.db ./db/racing.db

# 9000: racing gRPC. 9001: sports gRPC. 8000: api REST gateway.
EXPOSE 9000 9001 8000

# No default ENTRYPOINT/CMD: this image holds both binaries, and the
# caller picks which one to run, e.g.
#   docker run -p 9000:9000 <image> racing
#   docker run -p 9001:9001 <image> sports --grpc-endpoint=0.0.0.0:9001
#   docker run -p 8000:8000 <image> api --api-endpoint=0.0.0.0:8000 \
#     --grpc-endpoint=<racing-host>:9000 --sports-grpc-endpoint=<sports-host>:9001
# api and sports default to listening on localhost, which is unreachable from
# outside the container, so their listen addresses must be overridden.
