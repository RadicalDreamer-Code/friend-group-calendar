FROM node:24-alpine AS base
WORKDIR /app
ENV EXPO_NO_TELEMETRY=1
COPY package.json package-lock.json ./
RUN npm ci

# Development: Expo's Metro dev server for the web app, with hot reload.
# The source is bind-mounted by compose.yaml; node_modules stay in the image.
FROM base AS dev
COPY . .
ENV WEB_PORT=8082
EXPOSE 8082
CMD ["sh", "-c", "npx expo start --web --port ${WEB_PORT}"]

# Production build: the web app as static files in /app/dist.
FROM base AS build
COPY . .
# Where the app finds the API, e.g. https://api.example.com. Expo bakes
# EXPO_PUBLIC_* variables into the bundle, so it is set at build time.
ARG EXPO_PUBLIC_API_URL
RUN test -n "$EXPO_PUBLIC_API_URL" || { echo "Set the EXPO_PUBLIC_API_URL build argument." >&2; exit 1; }
RUN npx expo export --platform web

# Production: Caddy serves the files over HTTPS (configured in deploy/Caddyfile).
FROM caddy:2.11-alpine AS prod
COPY --from=build /app/dist /srv
