# Stage 1: Build the frontend app
FROM node:20-alpine3.21 AS build

WORKDIR /app

COPY ./frontend/app/package.json ./frontend/app/pnpm-lock.yaml ./
RUN corepack pnpm@9.15.4 --version
RUN corepack pnpm@9.15.4 install --frozen-lockfile && corepack pnpm@9.15.4 store prune

COPY ./frontend/app ./
RUN npm run build

# Stage 2: Build the static fileserver
FROM golang:1.24-alpine3.21 AS staticfileserver

WORKDIR /app

COPY go.mod go.sum ./
COPY ./cmd/hatchet-staticfileserver/ ./cmd/hatchet-staticfileserver/

RUN go build -o hatchet-staticfileserver ./cmd/hatchet-staticfileserver/main.go
RUN chmod +x ./hatchet-staticfileserver

# Stage 3: Run the static fileserver
FROM alpine:3.21

WORKDIR /app

COPY --from=build /app/dist ./dist
COPY --from=staticfileserver /app/hatchet-staticfileserver ./hatchet-staticfileserver

EXPOSE 3000

CMD ["/app/hatchet-staticfileserver", "-static-asset-dir", "/app/dist"]
