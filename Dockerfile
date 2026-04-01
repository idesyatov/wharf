FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
RUN CGO_ENABLED=0 go build -buildvcs=false \
    -ldflags "-s -w \
    -X github.com/idesyatov/wharf/internal/version.Version=${VERSION} \
    -X github.com/idesyatov/wharf/internal/version.Commit=${COMMIT} \
    -X github.com/idesyatov/wharf/internal/version.BuildDate=${BUILD_DATE}" \
    -o /wharf ./cmd/wharf

FROM alpine:3.20
RUN apk add --no-cache docker-cli docker-cli-compose
COPY --from=builder /wharf /usr/local/bin/wharf
ENTRYPOINT ["wharf"]
