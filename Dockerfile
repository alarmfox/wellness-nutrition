FROM golang:1.24.9 AS build

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o server cmd/server/main.go

FROM alpine

WORKDIR /app

COPY --from=build /app/server .

CMD ["/app/server"]
