---
name: pockethost
description: Use when setting up or updating PocketHost hosting and deployment workflows, especially SFTP deployment, branch-based staging and production, the pocketbase-pockethost CLI, and post-deploy health checks.
---

# PocketHost

Use this skill for hosted deployment and runtime guidance specific to PocketHost.

The preferred operational surface is the cross-platform `pocketbase-pockethost` binary. It uses the PocketBase project conventions already used by this repository:

- `pb_public/` for static files
- `pb_hooks/` for JavaScript hooks
- `pb_migrations/` for migrations
- `.pb_version` for the exact PocketBase version
- `.pb_config.json` for environment mapping and instance names

## Workflow

1. Confirm the project follows the standard layout.
2. Use `pocketbase-pockethost init` for a new project.
3. Use `pocketbase-pockethost doctor` before deployment.
4. Use `pocketbase-pockethost validate` to test migrations in a temporary data directory.
5. Use `pocketbase-pockethost deploy --dry-run` to inspect the SFTP diff.
6. Use `pocketbase-pockethost deploy` for the actual upload.
7. Use `pocketbase-pockethost health` after deployment.

## SFTP configuration

PocketHost deployments use SFTP only:

- host: `ftp.pockethost.io`
- port: `2222`
- username: `POCKETHOST_SFTP_USERNAME`
- private key: `POCKETHOST_SFTP_PRIVATE_KEY_PATH` or `POCKETHOST_SFTP_PRIVATE_KEY`
- instance: `POCKETHOST_INSTANCE_NAME`
- pinned host key: `POCKETHOST_SFTP_HOST_KEY`

Generate a local Ed25519 key with `pocketbase-pockethost key generate`, then register the public key in PocketHost Account → Keys. Never commit private keys.

The deployment synchronizes only `pb_public`, `pb_hooks`, and `pb_migrations`. It never manages `pb_data`. The remote sync manifest tracks managed files and permits safe deletion of files removed from the project; `--clean-slate --yes` is required for a full cleanup of the managed directories.

For GitHub Actions, store the username and private key as environment secrets and the instance name and host fingerprint as environment variables. Keep `main` and `master` mapped to `production`, and `staging` mapped to `staging`.

## References

- [references/github-actions-pockethost-deploy.md](references/github-actions-pockethost-deploy.md): workflow setup and secrets.
- [assets/Makefile](assets/Makefile): transitional compatibility commands.
- `$pocketbase`: PocketBase runtime, hooks, routing, and migration guidance.
