# Governance

This document describes how the CloudPulse open-source project is governed. It is
inspired by mature OSS projects (including Grafana and Gitea) and may evolve as the
community grows.

## Project mission

CloudPulse aims to provide an accessible, cloud-native observability platform for
uptime, latency, and system health — with a clear path from development to production.

## Roles

### Users

Anyone who deploys or evaluates CloudPulse. Feedback via issues and discussions shapes
the roadmap.

### Contributors

Individuals who submit issues, documentation, code, or reviews under the
[Contributor License Agreement implied by Apache 2.0](LICENSE) and
[CONTRIBUTING.md](CONTRIBUTING.md).

### Maintainers

Trusted contributors with merge rights who:

- Triage issues and pull requests
- Enforce the [Code of Conduct](CODE_OF_CONDUCT.md)
- Release versions and security advisories
- Guide technical direction in line with the mission

### Project lead

**Durga Prasad Raju Nadimpalli** serves as the founding maintainer and final decision
maker when consensus cannot be reached on scope, releases, or security response.

## Decision making

1. **Routine changes** — Discussed in PRs; merged by maintainers after review and CI.
2. **Significant features** — Should have an issue or design note for feedback before
   large implementation.
3. **Breaking changes** — Require explicit callouts in PRs and release notes.
4. **Disputes** — Discussed openly in issues/PRs; escalated to the project lead if
   unresolved after good-faith discussion.

## Releases

- **Pre-1.0:** Rapid iteration on `main`; tags may mark milestones.
- **1.0+:** Semantic versioning (SemVer); changelog entries for user-visible changes.
- **Security:** Handled per [SECURITY.md](SECURITY.md), independent of feature releases.

## Becoming a maintainer

Maintainer status is granted by existing maintainers based on:

- Sustained, high-quality contributions
- Constructive review participation
- Alignment with project values and code quality standards

There is no fixed timeline; nominations may be raised in private among maintainers and
confirmed with the project lead.

## License and intellectual property

- The project is licensed under [Apache License 2.0](LICENSE).
- Contributors retain copyright; contributions are licensed under the same terms.
- See [NOTICE](NOTICE) and [AUTHORS.md](AUTHORS.md) for attribution.

## Amendments

Changes to this governance document are proposed via pull request and require approval
from the project lead and at least one other maintainer (when multiple maintainers exist).
