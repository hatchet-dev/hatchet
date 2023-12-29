# Base Go environment
# -------------------
FROM golang:1.21-alpine as base
WORKDIR /hatchet

COPY go.mod go.sum ./

RUN go mod download

RUN go run github.com/steebchen/prisma-client-go prefetch

COPY /prisma ./prisma

CMD go run github.com/steebchen/prisma-client-go migrate deploy
