# Candella Ecosystem

One platform, every retail business — a single codebase intended to serve multiple
retail verticals (POS, Pharmacy ERP, Garments, Cosmetics, Inventory, CRM, and more)
instead of a separate app per industry.

**Status:** Phase 1 — project setup. Frontend and backend scaffolds exist and talk to
each other; no business features yet.

## Quick start

See [`SETUP.md`](SETUP.md) for the full, verified step-by-step flow (prerequisites,
install, environment variables, running both apps). The short version:

```bash
pnpm install
pnpm --filter frontend dev      # http://localhost:3000
cd backend && go run ./cmd/server   # http://localhost:8080
```

## Structure

- [`frontend/`](frontend/) — Next.js 16 app
- [`backend/`](backend/) — Go API (Chi router)
- [`packages/`](packages/) — shared TypeScript config and types
- [`docs/`](docs/) — architecture, engineering standards, roadmap
- [`SETUP.md`](SETUP.md) — the full setup flow, written to stay copy-paste accurate

## Documentation

Start with [`docs/README.md`](docs/README.md) for how the docs are organized, and
[`docs/architecture/00_overview.md`](docs/architecture/00_overview.md) for the system
design and the reasoning behind each technology choice.
