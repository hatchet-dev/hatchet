# This expects the hatchet-lite image to be built and available on the machine
# -------------------
ARG HATCHET_API_IMAGE

# Stage 1: copy from the existing Go built image
FROM $HATCHET_API_IMAGE as api-binary-base

# Stage 2: build the frontend
FROM node:18-alpine as frontend-build

WORKDIR /app

COPY ./frontend/app/package.json ./frontend/app/pnpm-lock.yaml ./
RUN corepack pnpm@9.15.4 --version
RUN corepack pnpm@9.15.4 install --frozen-lockfile && corepack pnpm@9.15.4 store prune

COPY ./frontend/app ./
RUN npm run build

# Stage 3: run in nginx alpine image
FROM nginx:alpine

ARG APP_TARGET=client

COPY --from=api-binary-base /hatchet/hatchet-api ./hatchet-api
COPY ./build/package/dashboard-entrypoint.sh ./entrypoint.sh
COPY ./build/package/dashboard-nginx.conf /etc/nginx/nginx.conf

RUN rm -rf /usr/share/nginx/html/*
COPY --from=frontend-build /app/dist /usr/share/nginx/html

# Make entrypoint script executable
RUN chmod +x ./entrypoint.sh

EXPOSE 80

# Run the entrypoint script
CMD ["./entrypoint.sh"]
