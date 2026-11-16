FROM node:24-bookworm-slim AS build
WORKDIR /app/agent-service
COPY agent-service/package.json agent-service/package-lock.json ./
RUN npm ci
COPY agent-service/tsconfig.json ./
COPY agent-service/src ./src
RUN npm run build && npm prune --omit=dev
FROM node:24-bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends bash git ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app/agent-service
COPY --from=build /app/agent-service/dist ./dist
COPY --from=build /app/agent-service/node_modules ./node_modules
COPY agent-service/package.json ./
ENV AGENT_HOST=0.0.0.0 AGENT_PORT=9401 NODE_ENV=production
USER node
EXPOSE 9401
CMD ["node","dist/server.js"]
