# Build the static binary.
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/vetkitten ./cmd/vetkitten

# Run it from a minimal image. GitHub container actions run as root and
# mount the workspace and the step summary file, so the image declares
# no USER.
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/vetkitten /vetkitten
ENTRYPOINT ["/vetkitten"]
