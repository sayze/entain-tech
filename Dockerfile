FROM golang:1.24-alpine AS build

WORKDIR /src

# racing uses github.com/mattn/go-sqlite3, which needs cgo, so we need a C
# toolchain here and link statically so the binary runs on scratch.
RUN apk add --no-cache ca-certificates build-base

# racing and api are separate modules, so each downloads its own deps.
COPY racing/go.mod racing/go.sum ./racing/
COPY api/go.mod api/go.sum ./api/
RUN cd racing && go mod download
RUN cd api && go mod download

COPY racing ./racing
COPY api ./api

RUN cd racing && CGO_ENABLED=1 go build -ldflags '-s -w -extldflags "-static"' -o /out/racing .
RUN cd api && CGO_ENABLED=0 go build -ldflags '-s -w' -o /out/api .

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/racing /usr/local/bin/racing
COPY --from=build /out/api /usr/local/bin/api

# racing opens its SQLite file at the relative path ./db/racing.db, so the
# working directory has to hold db/racing.db.
WORKDIR /app
COPY --from=build /src/racing/db/racing.db ./db/racing.db

# 9000: racing gRPC. 8000: api REST gateway.
EXPOSE 9000 8000

# No default ENTRYPOINT/CMD: this image holds both binaries, and the
# caller picks which one to run, e.g.
#   docker run -p 9000:9000 <image> racing
#   docker run -p 8000:8000 <image> api --api-endpoint=0.0.0.0:8000 --grpc-endpoint=<racing-host>:9000
# api defaults to listening on localhost:8000, which is unreachable from
# outside the container, so --api-endpoint must be overridden.
