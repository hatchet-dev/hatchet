# This expects the hatchet-lite image to be built and available on the machine
# -------------------
ARG HATCHET_LITE_IMAGE
ARG HATCHET_ADMIN_IMAGE

# Stage 1: copy from the existing Go built image
FROM $HATCHET_LITE_IMAGE as lite-binary-base

FROM $HATCHET_ADMIN_IMAGE as admin-binary-base

# Stage 2: build the frontend
FROM node:18-alpine as frontend-build

WORKDIR /app

COPY ./frontend/app/package.json ./frontend/app/pnpm-lock.yaml ./
RUN corepack pnpm --version
RUN corepack pnpm install --frozen-lockfile && corepack pnpm store prune

COPY ./frontend/app ./
RUN npm run build

# Stage 3: run in rabbitmq alpine image
FROM rabbitmq:alpine as rabbitmq

# install bash via apk
RUN apk update && apk add --no-cache bash gcc musl-dev openssl bash ca-certificates curl postgresql-client

RUN curl -sSf https://atlasgo.sh | sh

COPY --from=lite-binary-base /hatchet/hatchet-lite ./hatchet-lite
COPY --from=admin-binary-base /hatchet/hatchet-admin ./hatchet-admin
COPY --from=frontend-build /app/dist ./static-assets

# Copy entrypoint script
COPY ./hack/db/atlas-apply.sh ./atlas-apply.sh
COPY ./hack/lite/start.sh ./entrypoint.sh
COPY ./sql/migrations ./sql/migrations

ENV LITE_STATIC_ASSET_DIR=/static-assets
ENV LITE_FRONTEND_PORT=8081
ENV LITE_RUNTIME_PORT=8888

# Make entrypoint script executable
RUN chmod +x ./entrypoint.sh
RUN chmod +x ./atlas-apply.sh

EXPOSE 8888 7070

# Run the entrypoint script
CMD ["./entrypoint.sh"]
