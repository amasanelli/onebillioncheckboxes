FROM golang:1.21.1-alpine as builder

RUN mkdir /app

WORKDIR /app

COPY ./cmd/websockets ./

COPY ./internal ./internal

COPY ./go.mod ./go.mod

COPY ./go.sum ./go.sum

RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o websockets

RUN chmod +x ./websockets

####

FROM scratch

WORKDIR /app

COPY --from=builder /app/websockets /app/websockets

EXPOSE 80

ENTRYPOINT [ "/app/websockets" ]