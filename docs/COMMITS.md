# Conventional Commits

CloudPulse uses [Conventional Commits](https://www.conventionalcommits.org/) enforced by Commitlint.

## Format

```
<type>(<optional scope>): <description>

[optional body]
```

### Examples

```
feat(collector): add Redis ingestion queue
fix(api-gateway): allow Vercel preview origins in CORS
docs(adr): add ADR-001 database strategy
ci: add GitHub Actions workflow for Go services
chore(web): bump Next.js to 15.3
```

## Local check

```bash
npm install
npx commitlint --last
```

## Git hook (optional)

```bash
npx husky init
echo "npx --no -- commitlint --edit \$1" > .husky/commit-msg
```
