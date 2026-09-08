# PocketBase Pockethost Platform

This repository now has one main goal: make PocketBase and Pockethost projects ultra-simple.

The main user-facing product is the npm package:

- `pocketbase-pockethost`

It is designed for:

- small zero-build sites served from `pb_public`
- PocketBase projects that still need hooks and migrations
- GitHub-based deploys
- manual SFTP deploys for people not using GitHub

## What Lives In This Repository

### 1. Skills

These folders keep the Codex guidance:

- `pocketbase/`
- `pockethost/`

They still document conventions, but the preferred operational surface is now the CLI.

### 2. GitHub Workflow Templates

The repository still ships workflow guidance and templates for Pockethost deploys.

The preferred model is:

- generate a local workflow in the consuming repo
- let that workflow call the CLI
- keep branch mapping convention-based

### 3. Node CLI + Library

The new automation core lives in:

- [packages/pocketbase-pockethost/package.json](/Users/evaisse/Sites/projects/pocketbase-pockethost-skills/packages/pocketbase-pockethost/package.json)

This package provides:

- project scaffolding
- PocketBase binary management through `.pb_version`
- local development commands
- migration and hook generators
- health checks
- GitHub workflow generation
- manual SFTP deploys

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

The v1 command surface is:

- `npx pocketbase-pockethost init`
- `npx pocketbase-pockethost install`
- `npx pocketbase-pockethost dev`
- `npx pocketbase-pockethost test`
- `npx pocketbase-pockethost doctor`
- `npx pocketbase-pockethost health`
- `npx pocketbase-pockethost deploy`
- `npx pocketbase-pockethost sftp:deploy`
- `npx pocketbase-pockethost workflow:install`
- `npx pocketbase-pockethost migration:new <name>`
- `npx pocketbase-pockethost hooks:new <name>`

## Repository Direction

The direction is now:

- CLI first
- convention over configuration
- one config file: `.pb_config.json`
- one PocketBase version file: `.pb_version`
- Node/NPM as the only required user dependency

## Pockethost SFTP

Pockethost deployments use SFTP on `ftp.pockethost.io:2222`.

Required GitHub Environment configuration:

- `POCKETHOST_SFTP_USERNAME`: the Pockethost account email, as a secret
- `POCKETHOST_SFTP_PRIVATE_KEY`: an Ed25519 private key registered under Account → Keys, as a secret
- `POCKETHOST_INSTANCE_NAME`: the instance subdomain, as a variable or secret

The reusable workflow is public and can be consumed from another repository:

```yaml
jobs:
  deploy:
    uses: 8bit-interactive/pocketbase-pockethost-skills/.github/workflows/pockethost-deploy.yml@v0.2.0
    with:
      working-directory: .
    secrets: inherit
```

Create a project quickly with `npx pocketbase-pockethost init`, then run `npx pocketbase-pockethost workflow:install`.

Legacy copy-paste assets such as long `Makefile`-driven flows are kept only as transitional material while the CLI becomes the default path.
