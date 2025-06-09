FROM node:18-alpine AS build

WORKDIR /app

COPY ./frontend/app/package.json ./frontend/app/pnpm-lock.yaml ./
RUN corepack pnpm@9.15.4 --version
RUN corepack pnpm@9.15.4 install --frozen-lockfile && corepack pnpm@9.15.4 store prune

COPY ./frontend/app ./
RUN npm run build

# Stage 2: Serve the built app with NGINX
FROM nginxinc/nginx-unprivileged:alpine

USER root
RUN rm -rf /usr/share/nginx/html/*
USER nginx

ARG APP_TARGET=client

COPY ./build/package/nginx.conf /etc/nginx/nginx.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
