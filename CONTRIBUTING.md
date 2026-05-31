# Contributing to CloudPulse

Thank you for your interest in contributing to CloudPulse. This document explains
how to participate in a way that keeps the project healthy, reviewable, and
welcoming for everyone.

## Table of contents

- [Code of conduct](#code-of-conduct)
- [Ways to contribute](#ways-to-contribute)
- [Development setup](#development-setup)
- [Branching and commits](#branching-and-commits)
- [Pull request process](#pull-request-process)
- [Coding standards](#coding-standards)
- [License](#license)

## Code of conduct

This project follows our [Code of Conduct](CODE_OF_CONDUCT.md). By participating,
you agree to uphold a respectful, inclusive environment.

## Ways to contribute

You do not need to write code to help:

- **Report bugs** — Use the bug report template with reproduction steps.
- **Suggest features** — Open a discussion or feature request with context and use cases.
- **Improve docs** — Fix typos, clarify setup steps, or add examples.
- **Submit code** — Bug fixes, tests, API improvements, and UI polish are all welcome.
- **Review PRs** — Thoughtful reviews are as valuable as new features.

## Development setup

### Prerequisites

- Go 1.22+
- Node.js 20+ and npm
- Docker & Docker Compose (recommended for full stack)
- GNU Make (optional; see `Makefile`)

### Quick start

```bash
git clone https://github.com/yourusername/cloudpulse.git
cd cloudpulse
cp .env.example .env

# Full stack
make dev-build

# Or run services separately
make backend-run          # API on :8080
cd frontend && npm install && npm run dev   # UI on :5173
```

### Verify your changes

```bash
# Backend
cd backend && go test ./... && go vet ./...

# Frontend
cd frontend && npm run lint && npm run build
```

## Branching and commits

- Branch from `main` using descriptive names: `feat/uptime-scheduler`, `fix/health-cors`.
- Keep commits focused; one logical change per commit when possible.
- Write commit messages in the imperative mood:

  ```
  Add health check endpoint for readiness probes

  - Expose GET /ready for Kubernetes
  - Document env vars in README
  ```

- Reference issues when applicable: `Fixes #42`.

## Pull request process

1. **Fork** the repository and create a branch from `main`.
2. **Implement** your change with tests where behavior is non-trivial.
3. **Run** lint and build commands locally (see above).
4. **Update** documentation if you change APIs, config, or setup steps.
5. **Open a PR** with:
   - A clear title and summary of *why* the change is needed
   - Screenshots for UI changes
   - Notes for reviewers (breaking changes, follow-ups)
6. **Respond** to review feedback; maintainers may request changes before merge.

Draft PRs are welcome for early feedback on large work.

### Review expectations

- PRs require at least one approval from a maintainer before merge.
- CI must pass (lint, build, tests when present).
- We may squash-merge to keep history readable.

## Coding standards

### Go (backend)

- Follow standard Go conventions (`gofmt`, `go vet`).
- Place application code under `cmd/`, `internal/`, and `pkg/` as documented in README.
- Prefer explicit error handling; avoid panics in library code.
- Add Apache 2.0 license headers to new source files (see existing files).

### TypeScript / React (frontend)

- **Strict TypeScript** — do not disable strict checks without discussion.
- Run `npm run lint` before pushing.
- Use Tailwind for styling; keep components accessible (labels, contrast, keyboard).
- Add license headers to new `.ts` / `.tsx` files.

### General

- No secrets in commits (`.env`, keys, tokens).
- Prefer small, reviewable PRs over large monolithic changes.
- Discuss architectural changes in an issue before investing significant effort.

## License

By contributing to CloudPulse, you agree that your contributions will be licensed
under the [Apache License 2.0](LICENSE). You retain copyright to your contributions;
see the NOTICE file for project attribution.

If you include third-party code, ensure it is compatible with Apache 2.0 and document
the origin in your PR.

---

Questions? Open a [GitHub Discussion](https://github.com/durgaprasadraju/cloudpulse/discussions)
or tag **@yourusername** in an issue.

**Durga Prasad Raju Nadimpalli** — Project lead
