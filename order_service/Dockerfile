
FROM golang:1.23.6  AS builder


WORKDIR /app


COPY go.mod go.sum ./

RUN go mod download

COPY . .


RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server cmd/main.go


FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/server /app/server

COPY .env .env

COPY migrations ./migrations


EXPOSE 8082

ENTRYPOINT ["/app/server"]