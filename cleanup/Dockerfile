FROM golang:1.19.6-alpine3.17 as build

WORKDIR /app

COPY cleanup ./

RUN go build -o app main.go

FROM alpine

WORKDIR  /app
COPY --from=build /app/app ./cleanup

RUN echo ' 0 0 1 1-12/1 * /app/cleanup --cmd=events --database-url=$DATABASE_URL' >> /etc/crontabs/root 
RUN echo ' 0 0 1 1-12/3 * /app/cleanup --cmd=slots --database-url=$DATABASE_URL'>> /etc/crontabs/root

CMD ["/usr/sbin/crond", "-f"]
