# Verification · 2026-11-10

Verified locally on Windows with Node.js 24.22.2 and project-local Go 1.27.3. Gin is pinned to 1.12.0 and the TypeScript Claude Agent SDK to 0.3.319. Previous editions and their private environment files were not changed or copied. Tests used placeholder model keys or injected SDK iterators; no paid inference ran.

- **46 automated tests passed:** 20 Node service/SDK tests, 19 React/API client tests and 7 Go gateway tests. Go tests also include multiple input-validation cases inside their test functions.
- Node tests cover internal-token gating before inference, direct unauthenticated health rejection, SDK options/projection/final/error behavior, cancellation/deadline, isolated runtime cleanup, service/cloud credential filtering, UTF-8 streaming and valid keep-alive requests after oversized input rejection.
- Go tests cover actual body/media/JSON/prompt/origin validation before forwarding, normalized prompt plus server-owned token, refusal to forward browser cookies/origin as credentials, progressive streaming before upstream completion, real request-context cancellation, static/dotfile/API confinement, token/service failures and explicit trusted HTTPS configuration.
- React tests cover all five views, retained prompt/answer, Stream/Simple, duplicate locking/cancellation, partial/final/error handling, source walkthroughs, theme, safe highlighting/Markdown and fragmented Unicode/error/incomplete SSE.
- TypeScript checks passed for React and the Node service. Go source was formatted, compiled and tested; Go module checksums passed `go mod verify`.
- Full production build passed: generated sixteen real-source excerpts/SVG/PlantUML, compiled Node, built React and compiled the Windows Gin executable.
- Production HTTP checks passed through the real Gin → Node path: health, input rejection, missing-key query/SSE, schema/reference, compiled HTML/JavaScript/SVG, unknown APIs/dotfiles and server-controlled token forwarding despite a spoofed browser header. Direct unauthenticated Node health/query requests returned 401.
- Chrome checks passed against the compiled production UI: all five views, retained prompt, missing-key stream feedback, sixteen colored Go/TypeScript excerpts, loaded native-size UML, internal diagram scrolling, persisted theme and no page overflow at 390px in every view. Screenshots are in ignored verification/.
- The documented npm run dev launcher started Node, Gin and Vite in readiness order and passed the same Chrome checks on port 6100 independently of the production launcher.
- The native bundled Windows CLI ran offline with --version and reported 2.1.319 (Claude Code).
- Re-running local environment configuration preserved both existing server files byte-for-byte, without printing credentials. Setup generated paired random local service tokens; the model key remains a placeholder until the user edits it.
- Docker Compose configuration validated with `docker compose config --quiet`, without exposing resolved secrets.

## Dependencies and limits

Shipped Node and frontend dependency audits reported **zero vulnerabilities**. The complete Node development dependency installation reported **one low-severity esbuild advisory**; no forced incompatible update was applied. Go module verification establishes checksum integrity, not absence of vulnerabilities; a separate Go vulnerability scan was not performed.

The official Go archive was checked against its published SHA-256 and installed only under this edition's ignored .tools directory. Module/build caches remain local and no system PATH was changed. A cold Go compile can take several minutes; setup now builds Gin once and the launcher allows extra startup time with readiness feedback.

Docker CLI is installed but its Linux engine pipe was unavailable, so neither image nor the live Compose stack was executed. No hosting/cloud resource was created. Linux builds, hosted access control/TLS/private networking, actual provider authentication/model access/credits and real model cancellation remain untested. Hard termination can interrupt cleanup. Environment filtering and temporary directories are not a network or metadata sandbox. The internal token is distinct from end-user authentication.

## Reproduce

```powershell
npm ci
npm run setup
npm run typecheck
npm test
npm run build
```

With a placeholder model key, run npm start, then npm run verify:http in another terminal. For production browser checks set VERIFY_BASE_URL=http://127.0.0.1:9400 before npm run verify:browser. Development browser checks default to http://127.0.0.1:6100 with npm run dev. Installed Chrome is required. Both verification scripts reject a configured model key to prevent unintended paid inference.
