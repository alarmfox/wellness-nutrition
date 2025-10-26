FROM golang:1.24.9 AS build

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o server cmd/server/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o seed cmd/seed/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o migrate cmd/migrations/migrate.go
RUN CGO_ENABLED=0 GOOS=linux go build -o cleanup cmd/cleanup/main.go

FROM alpine

WORKDIR /app

RUN apk add --no-cache tz

COPY --from=build /app/server /app/seed /app/migrate /app/cleanup .

CMD ["/app/server"]
