FROM golang:1.14.4 AS build

WORKDIR $GOPATH/src/github.com/dnys1/grpc-mongo

COPY . .
RUN go mod download

WORKDIR $GOPATH/src/github.com/dnys1/grpc-mongo

RUN CGO_ENABLED=0 GOOS=linux go build -a -o /server .

FROM alpine:3.12.0

COPY --from=build /server /usr/local/bin/server

EXPOSE 8081
EXPOSE 50051

ENTRYPOINT ["/usr/local/bin/server", "--grpc-host", "0.0.0.0", "--db-host", "blog_db"]