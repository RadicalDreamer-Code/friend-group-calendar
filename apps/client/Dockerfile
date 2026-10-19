# Development: Expo's Metro dev server for the web app, with hot reload.
# The source is bind-mounted by compose.yaml; node_modules stay in the image.
FROM node:24-alpine AS dev
WORKDIR /app
ENV EXPO_NO_TELEMETRY=1
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ENV WEB_PORT=8082
EXPOSE 8082
CMD ["sh", "-c", "npx expo start --web --port ${WEB_PORT}"]
