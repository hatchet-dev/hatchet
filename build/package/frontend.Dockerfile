FROM node:18-alpine as build

WORKDIR /app
COPY ./frontend/app/package.json ./frontend/app/package-lock.json ./
RUN npm install
COPY ./frontend/app ./
RUN npm run build

# Stage 2: Serve the built app with NGINX
FROM nginx:alpine

ARG APP_TARGET=client

COPY ./build/package/nginx.conf /etc/nginx/nginx.conf
RUN rm -rf /usr/share/nginx/html/*
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
