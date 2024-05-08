# syntax=docker/dockerfile:1.4
FROM --platform=$BUILDPLATFORM golang:1.22 AS builder

WORKDIR /code

ENV CGO_ENABLED 0
ENV GOPATH /go
ENV GOCACHE /go-build

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod/cache \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod/cache \
    --mount=type=cache,target=/go-build \
    go build -o bin/backend .

CMD ["/code/bin/backend"]

FROM builder AS dev-envs

RUN <<EOF
apt-get update
apt-get install git
EOF

RUN <<EOF
adduser --system --group docker
EOF

# install Docker tools (cli, buildx, compose)
COPY --from=gloursdocker/docker / /

CMD ["go", "run", "."]

FROM scratch
COPY --from=builder /code/bin/backend /usr/local/bin/backend
CMD ["/usr/local/bin/backend"]