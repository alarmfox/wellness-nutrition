FROM golang:1.20.4-alpine as build

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o survey

FROM alpine 

WORKDIR /app

COPY --from=build /app/survey .

ENTRYPOINT ["/app/survey"]
