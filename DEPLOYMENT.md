# Deploy the React + Go/Gin + Node service edition

This edition has two server processes: the Gin gateway and the private Node Agent Service. Production React files are served by Gin. No hosting resources or containers have been created during implementation.

## Build for the intended operating system

Use Node.js 24 and Go 1.27 (the local installer pins official Go 1.27.3 with checksum verification). Run npm ci, npm run setup, npm test, npm run typecheck and npm run build. The project-local Go module/build caches live under .tools. A Windows Go binary and Windows SDK native executable cannot run on a Linux host; build and install dependencies on the target OS.

The gateway artifact consists of backend/bin/gateway (or gateway.exe), frontend/dist and backend/openapi.json. Run it from backend, or supply -static and -schema absolute paths. Its private configuration is backend/.env or injected environment.

The Node artifact needs agent-service/dist, its package.json and the complete production node_modules including the SDK's platform-specific optional executable. Install/build on the target OS; do not use --omit=optional or remove the native CLI. The runtime needs writable OS temporary storage, native subprocess execution and outbound HTTPS to Anthropic. Debian/glibc Node 24 with Bash, Git and CA certificates is provided in the optional Dockerfile. The native executable alone is over 300 MB.

## Private communication and public access

- Expose Gin through your intended HTTPS ingress. Keep Node's port private to the gateway/network. If deploying on separate hosts, use a controlled private network or authenticated TLS; the shared token alone does not encrypt plaintext HTTP across an untrusted network.
- Inject a long random AGENT_SERVICE_TOKEN with the same value in both server environments. Inject ANTHROPIC_API_KEY only into Node. Rotate paired service tokens together. Never copy them into React, build output, source control or logs.
- Configure Gin's fixed AGENT_SERVICE_URL, API host/port and static/schema paths. Configure Node's AGENT_HOST/AGENT_PORT. For containers bind to 0.0.0.0 inside their private network; loopback is the local default.
- Add end-user authentication/access control at the public gateway/ingress before intentionally exposing paid inference. Same-origin checks do not authenticate direct callers. The internal token proves service-to-service access, not user identity or a per-user budget.
- Leave TRUST_PROXY=0 unless the gateway is reachable only through a controlled proxy that overwrites X-Forwarded-Proto and preserves the intended Host. Gin does not trust forwarded Host for origin selection.
- Give the proxy sufficient streaming/idle limits for the 180-second Node application deadline plus cleanup and disable SSE buffering where applicable. Concurrency, process/file-descriptor and memory use must be measured; this lesson does not provide durable sessions or a job queue.

Run Node with `node agent-service/dist/server.js` from the project root, and the compiled Gin binary with backend as working directory. A process manager can supervise both. Graceful shutdown is bounded; hard termination can interrupt cleanup. A removed agent key or mismatched internal token must produce safe unavailable/error feedback rather than exposing raw credentials or provider diagnostics.

## Optional local containers

The Compose file builds React into the Gin image and installs Linux SDK dependencies into a separate Node image. Node has no published host port; Gin's local example maps only loopback port 9400. Runtime .env files are supplied separately and excluded from image build context. Both runtime images use non-root users.

After setup and setting your own Node model key, with Docker Desktop's Linux engine running:

```powershell
docker compose up --build
```

Open **http://127.0.0.1:9400**. Compose overrides local loopback bind addresses inside containers and changes the gateway URL to the private agent service DNS name. Only the gateway is host-published. These are local deployment commands for the user; no container has been started automatically.

## Verify the deployed result

Check health, all five views, syntax highlighting, UML downloads and API reference. Verify unauthorized callers cannot reach either intended boundary. Test mismatched/revoked service tokens and missing/revoked model keys without exposing their values. Then run real Simple/Stream with your own key (paid provider calls), requiring actual SDK messages, final plus done and reported metrics. Check that events arrive progressively across both hops.

Test cancellation, deadline, service outages, restarts and concurrent requests. Observe Go request contexts, Node controller cleanup, CPU/memory/temporary files, provider spending and transport buffering. Local injected tests do not prove hosted inference or network controls. VERIFICATION.md records actual evidence and untested behavior.
