# Frontend

Independent Next.js + TypeScript application for the local Code Risk Analyzer experience.

## Run locally

```bash
cd frontend
npm install
npm run dev
```

Open the URL printed by Next.js. Browser API requests use relative `/api/...` paths only. `next.config.ts` proxies those paths server-side to the Go API at `127.0.0.1:8080`, so frontend code does not call an external analysis service or directly access a repository.

The current UI is an honest foundation page. It does not display mock analysis results or provide a repository-analysis form until those backend capabilities exist.

## Quality commands

```bash
npm run typecheck
npm test
npm run build
```

Start the Go API separately from `../backend` before selecting **Check local API health**.
