FROM node:24-bookworm-slim AS frontend
WORKDIR /build/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build
FROM golang:1.27.3-bookworm AS gateway
WORKDIR /build/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/*.go ./
RUN CGO_ENABLED=0 go build -trimpath -o gateway .
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/* && useradd --create-home --uid 10001 agent
WORKDIR /app/backend
COPY --from=gateway /build/backend/gateway ./gateway
COPY --from=frontend /build/frontend/dist /app/frontend/dist
COPY backend/openapi.json ./openapi.json
ENV API_HOST=0.0.0.0 API_PORT=9400
USER agent
EXPOSE 9400
CMD ["./gateway","-static","/app/frontend/dist","-schema","/app/backend/openapi.json"]
