FROM golang:1.20-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o pacman main.go

FROM alpine:3.18

RUN apk add --no-cache curl python3

WORKDIR /app

COPY --from=builder /app/pacman .
COPY index.html ./
COPY static/ ./static/

EXPOSE 8080

CMD ["./pacman"]
