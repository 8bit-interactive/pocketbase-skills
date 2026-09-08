# PocketBase Pockethost Platform

This repository now has one main goal: make PocketBase and Pockethost projects ultra-simple.

The main user-facing product is the cross-platform binary:

- `pocketbase-pockethost`

It is designed for:

- small zero-build sites served from `pb_public`
- PocketBase projects that still need hooks and migrations
- GitHub-based deploys
- SFTP deploys for local use and GitHub Actions

## What Lives In This Repository

### 1. Skills

These folders keep the Codex guidance:

- `skills/pocketbase/`
- `skills/pockethost/`

They still document conventions, but the preferred operational surface is now the CLI.

### 2. GitHub Workflow Templates

The repository still ships workflow guidance and templates for Pockethost deploys.

The preferred model is:

- generate a local workflow in the consuming repo
- let that workflow call the CLI
- keep branch mapping convention-based

### 3. Go CLI

The deployment automation core lives in:

- [cli/cmd/pocketbase-pockethost/main.go](cli/cmd/pocketbase-pockethost/main.go)

This package provides:

- project scaffolding
- PocketBase binary downloads through `.pb_version`
- isolated migration validation
- Ed25519 key generation
- health checks
- SFTP synchronization with a remote state manifest
- GitHub workflow generation

### 4. Hosted demonstration

The [demo/](demo/) directory is a zero-build site deployed by [demo-pockethost-deploy.yml](.github/workflows/demo-pockethost-deploy.yml). Configure the `staging` and `production` GitHub Environments to use it as a live deployment example.

## Default Project Model

The default generated project keeps the existing PocketBase layout:

- `pb_public/`
- `pb_hooks/`
- `pb_migrations/`

The zero-build default is intentionally small:

- edit `pb_public/index.html`
- edit `pb_public/assets/site.css`
- use `npm run dev`
- push `staging`
- then push `main`

## CLI Commands

The binary command surface is:

- `pocketbase-pockethost init`
- `pocketbase-pockethost key generate`
- `pocketbase-pockethost pocketbase install`
- `pocketbase-pockethost validate`
- `pocketbase-pockethost doctor`
- `pocketbase-pockethost deploy --dry-run`
- `pocketbase-pockethost deploy`
- `pocketbase-pockethost health`
- `pocketbase-pockethost workflow install`

## Repository Direction

The direction is now:

- CLI first
- convention over configuration
- one config file: `.pb_config.json`
- one PocketBase version file: `.pb_version`
- one statically linked Go binary as the only required user dependency
- SFTP only, using `ftp.pockethost.io:2222`
- no remote management of `pb_data`

Legacy copy-paste assets such as long `Makefile`-driven flows are kept only as transitional material while the CLI becomes the default path.
