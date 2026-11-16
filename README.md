# The Smallest Agent · React + Go/Gin + Node Agent Service

Independent edition 19: **React/TypeScript → Go/Gin gateway → private Node.js service → TypeScript Claude Agent SDK → Claude API**. It retains five working lesson tabs, large readable fonts, pastel cards, persistent theme, colored code, safe Markdown and the sixteen-step source-linked UML walkthrough.

## Run on Windows

Install **Node.js 24**. Go 1.27 is supported from PATH; if absent, setup downloads official Go 1.27.3 into this edition's ignored .tools directory and verifies its SHA-256. Windows 10/11's tar command extracts the archive. No system PATH change, Python, virtual environment or cloud account is needed.

From **Lesson 0**:

```powershell
cd .\19-smallest-agent-react-go-gin-node
npm ci
npm run setup
notepad agent-service\.env
```

Replace ANTHROPIC_API_KEY with your own key, save, then:

```powershell
npm run dev
```

Wait for Vite, then open **http://127.0.0.1:6100**. Gin runs at **9400** and Node at **9401**. First Go compilation can take longer. The launcher waits for Node readiness, then authenticated Gin health, then starts Vite. Ctrl+C stops its own child services.

## Configuration: two server files

Setup creates missing agent-service/.env and backend/.env with one matching random internal-service token. It never prints the token or overwrites an existing file. It also compiles Gin once so the initial development start can use the warmed Go cache. The model key remains a placeholder until you edit it. No credentials are copied from previous editions.

**agent-service/.env** owns ANTHROPIC_API_KEY, CLAUDE_MODEL (blank uses the SDK default), AGENT_SYSTEM_PROMPT, AGENT_MAX_TURNS (integer 1–10), AGENT_HOST/AGENT_PORT and AGENT_SERVICE_TOKEN. Node loads the file relative to its module; existing process environment takes precedence. Restart after changes.

**backend/.env** owns API_HOST/API_PORT, AGENT_SERVICE_URL, the same AGENT_SERVICE_TOKEN and TRUST_PROXY=0. It contains no model key. Gin loads this file from its backend working directory. A token mismatch returns safe unavailable feedback and prevents startup readiness. If you create these files manually from the examples, replace the token placeholder with the same long random value in both. Only setup-generated files provide this automatically.

Node binds to loopback by default. Only Gin attaches the private service header; browser headers, cookies and tokens are not forwarded as credentials. The launcher removes ambient model variables from the Go child's environment. Never put model or service keys in VITE_ variables, public assets or source code. Key presence does not establish provider authentication, model access or credits.

Separate terminals can run:

```powershell
npm run dev:agent
```

```powershell
npm run dev:backend
```

```powershell
npm run dev:frontend
```

The combined launcher uses the documented default ports. If customizing ports or moving services, use separate commands and update the Vite proxy/service URL consistently.

## Build and run production locally

```powershell
npm run typecheck
npm test
npm run build
```

Stop development, then:

```powershell
npm start
```

Open **http://127.0.0.1:9400**. The launcher starts compiled Node plus the native Gin binary; Gin serves frontend/dist and its API from one origin. There is no Vite server in this mode. Build regenerates actual-source excerpts and diagrams, compiles the independent React and Node projects and creates backend/bin/gateway.exe on Windows (gateway on Linux).

## Application and API

- The lab includes Hello world!, example prompts, Stream/Simple, a trimmed counter, Ctrl+Enter, immediate duplicate locking, elapsed time, Cancel, safe SDK message JSON, final answer/turns/duration/cost/error, copying and Markdown download. Prompt and results persist across lesson navigation.
- The learning/setup/architecture/source sections explain this exact service boundary. Each of sixteen UML steps has a generated syntax-colored Go/TypeScript excerpt with source path, line and explanation. The readable native-size diagram scrolls internally and downloads as SVG/PlantUML.
- GET /api/health checks the authenticated Node service and returns readiness plus model-key presence.
- POST /api/agent/query accepts JSON `{ "prompt": "Hello world!" }` and returns the normalized final SDK result. POST /api/agent/stream relays real UTF-8 SSE message/done/error events.
- /api/docs is a custom reference; /api/openapi.json is the schema. These are served by Gin.

Gin validates origin, exact JSON media type, actual 32 KiB body bound and trimmed 1–4000 character prompts before forwarding a normalized request. It uses a fixed configured upstream, attaches its own internal token and propagates the browser request context. The service independently validates input and compares tokens timing-safely. It flushes SSE immediately; Gin flushes each received chunk. Browser cancellation closes the upstream response and aborts the SDK controller. Node's 180-second application deadline is below Go's 185-second context and 190-second client limit. Final result plus done is required; completion never overrides an SDK error.

Only Node imports the official CLI-backed Agent SDK. Tools, settings, skills, MCP, plugins and persisted sessions are disabled. Each run owns fresh temporary work/home/config directories and closes/cleans them on completion or cancellation. Safe public projections omit thinking, private paths/session IDs and tool payloads; exact configured-key occurrences are redacted. Internal/cloud credentials are excluded from the CLI child environment. This is not a metadata or network sandbox. Unreported cost remains null. SDK messages are distinct from token deltas. Highlight.js colors code and DOMPurify sanitizes output.

For hosting, see DEPLOYMENT.md. The internal token authenticates Gin to Node; it is separate from end-user authentication and public API access control. Set TRUST_PROXY only behind a controlled HTTPS proxy. VERIFICATION.md records actual checks and limits; BUILD-PROMPT.txt matches the standalone prompt in Lesson 0.

Sources: [Gin documentation](https://gin-gonic.com/en/docs/), [Go HTTP](https://pkg.go.dev/net/http), [SDK hosting](https://code.claude.com/docs/en/agent-sdk/hosting).
