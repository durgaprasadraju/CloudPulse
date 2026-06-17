# GitHub setup: push and branch protection

## 1. Initialize and push

```bash
git init
git add .
git commit -m "chore: init CloudPulse monorepo with services, web, and infra"
git branch -M main
git remote add origin https://github.com/durgaprasadraju/CloudPulse.git
git push -u origin main
```

For feature work:

```bash
git checkout -b develop
git push -u origin develop
```

## 2. Install GitHub CLI (Windows)

```powershell
winget install GitHub.cli
gh auth login
```

## 3. Branch protection (main)

```bash
gh api repos/durgaprasadraju/CloudPulse/branches/main/protection \
  --method PUT \
  --field required_status_checks='{"strict":true,"contexts":["ci"]}' \
  --field enforce_admins=true \
  --field required_pull_request_reviews='{"required_approving_review_count":1}' \
  --field restrictions=null
```

## 4. Branch protection (develop)

```bash
gh api repos/durgaprasadraju/CloudPulse/branches/develop/protection \
  --method PUT \
  --field required_status_checks='{"strict":true,"contexts":["ci"]}' \
  --field enforce_admins=false \
  --field required_pull_request_reviews='{"required_approving_review_count":1}' \
  --field restrictions=null
```

## 5. Required checks

Enable **Settings → Branches → Require status checks** for workflow `CI` after the first successful run.

## 6. Vercel

Connect the `web/` directory as the root for the Next.js project (`app.cloudpulse.live`).
