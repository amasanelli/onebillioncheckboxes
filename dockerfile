FROM golang:alpine as builder

RUN mkdir /app

WORKDIR /app

COPY ./ ./

RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o onebillioncheckboxes

RUN chmod +x ./onebillioncheckboxes

####

FROM golang:alpine

WORKDIR /app

COPY ./templates ./templates

COPY ./public ./public

COPY --from=builder /app/onebillioncheckboxes /app/onebillioncheckboxes

EXPOSE 80

ENTRYPOINT [ "/app/onebillioncheckboxes" ]