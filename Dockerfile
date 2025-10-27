FROM golang:1.24.9 AS build

WORKDIR /app

# Install minifier
RUN go install github.com/tdewolff/minify/v2/cmd/minify@latest

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Minify static assets before compilation
RUN minify -r -o ./static cmd/server/static
RUN cp -r ./static/static/* cmd/server/static && rm -rf ./static

# Build binaries
RUN CGO_ENABLED=0 GOOS=linux go build -o server cmd/server/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o seed cmd/seed/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o migrate cmd/migrations/migrate.go
RUN CGO_ENABLED=0 GOOS=linux go build -o cleanup cmd/cleanup/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o reminder cmd/reminder/main.go

FROM alpine

WORKDIR /app

RUN apk add --no-cache tz

COPY --from=build /app/server /app/seed /app/migrate /app/cleanup /app/reminder .

CMD ["/app/server"]
