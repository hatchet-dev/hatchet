# Base Go environment
# -------------------
FROM golang:1.21-alpine as base
WORKDIR /hatchet

RUN apk update && apk add --no-cache gcc musl-dev git

COPY go.mod go.sum ./
COPY /prisma ./prisma

RUN go install github.com/steebchen/prisma-client-go@v0.31.2

# prefetch the binaries, so that they will be cached and not downloaded on each change
RUN go run github.com/steebchen/prisma-client-go prefetch

CMD go run github.com/steebchen/prisma-client-go migrate deploy