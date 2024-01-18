# Base Go environment
# -------------------
FROM golang:1.21-alpine as base
WORKDIR /hatchet

# curl is needed for things like signaling cloudsql proxy container to stop after a migration
RUN apk update && apk add --no-cache curl

COPY go.mod go.sum ./

RUN go mod download

RUN go run github.com/steebchen/prisma-client-go prefetch

COPY /prisma ./prisma

CMD go run github.com/steebchen/prisma-client-go migrate deploy
