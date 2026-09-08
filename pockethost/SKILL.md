---
name: pockethost
description: Use when setting up or updating Pockethost hosting and deployment workflows, especially GitHub Actions deployment, SFTP deployment, branch-based staging and production, and the pocketbase-pockethost CLI.
---

# Pockethost

## Overview

Use this skill for hosted deployment and runtime guidance specific to Pockethost.

The preferred operational surface is now the `pocketbase-pockethost` CLI.

Default assumptions:

- zero-build small site first
- `pb_public/` is the normal user-editable surface
- `pb_hooks/` and `pb_migrations/` are supported but optional
- GitHub-first deployment, with manual SFTP as the official file-transfer path

## Workflow

1. Confirm the project follows the standard layout: `pb_public`, `pb_hooks`, `pb_migrations`.
2. Prefer `npx pocketbase-pockethost init` for new projects.
3. Prefer `npx pocketbase-pockethost workflow:install` over hand-editing workflows.
4. Prefer `npx pocketbase-pockethost doctor`, `health`, `test`, and `deploy` over long local shell glue.
5. Use GitHub Environments named `production` and `staging`.
6. Store `POCKETHOST_SFTP_USERNAME` and `POCKETHOST_SFTP_PRIVATE_KEY` as environment secrets.
7. Store `POCKETHOST_INSTANCE_NAME` as an environment variable or secret.
8. Connect to `ftp.pockethost.io` on port `2222` with an Ed25519 key.
9. Keep `.pb_version` as the source of truth for the PocketBase version.
10. For PocketBase application logic, hooks, SPA routing, or local migration validation, use `$pocketbase`.

## Deployment Rules

- Deploy `main` to `production`.
- Deploy `master` to `production`.
- Deploy `staging` to `staging`.
- Default to a local GitHub workflow generated into the consuming repository.
- Treat manual SFTP deploy as a first-class non-GitHub path.
- Keep the SFTP secret names identical across environments.
- Resolve `POCKETHOST_INSTANCE_NAME` from environment vars before falling back to environment secrets.
- For small static sites, assume the main user-editable files are `pb_public/index.html` and `pb_public/assets/site.css`.
- Keep the SPA routing mount separate from asset directories. If the frontend uses `/page`, bundles still belong under `pb_public/assets` or `pb_public/dist`.

## References

- [references/github-actions-pockethost-deploy.md](references/github-actions-pockethost-deploy.md): Detailed setup notes and workflow behavior.
- [assets/github-actions-pockethost-deploy.yml](assets/github-actions-pockethost-deploy.yml): GitHub Actions SFTP workflow template material.
- [assets/Makefile](assets/Makefile): Legacy Makefile reference during the CLI transition.
- [../packages/pocketbase-pockethost/package.json](../packages/pocketbase-pockethost/package.json): CLI package entrypoint.
- `$pocketbase`: PocketBase runtime, testing, and local migration validation guidance.
