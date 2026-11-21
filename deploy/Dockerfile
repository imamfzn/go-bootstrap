# One image, three binaries. Choose the binary with the container command:
#   /bin/api                  Cloud Run service (the default)
#   /bin/job <name>           Cloud Run Job
#   /bin/worker               long-running consumer
#
# Build from the repository root:
#   docker build -f deploy/Dockerfile -t go-bootstrap .

FROM golang:1.27 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ ./cmd/...

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/ /bin/

USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/bin/api"]
