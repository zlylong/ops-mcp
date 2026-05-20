FROM node:20-bookworm AS build
WORKDIR /app
COPY frontend/package*.json ./
RUN npm ci
COPY frontend ./
ARG VITE_APP_VERSION=0.1.0
ENV VITE_APP_VERSION=$VITE_APP_VERSION
RUN npm run build

FROM nginx:1.27-alpine
COPY frontend/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
