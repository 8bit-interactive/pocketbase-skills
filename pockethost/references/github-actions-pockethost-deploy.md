# GitHub Actions SFTP Deployment for Pockethost

Use this reference when a repository needs a standard GitHub Actions deployment to Pockethost.

## Connection contract

Pockethost uses SFTP for file deployment:

| Setting | Value |
| --- | --- |
| Protocol | SFTP |
| Host | `ftp.pockethost.io` |
| Port | `2222` |
| Username | Pockethost account email |
| Authentication | Ed25519 SSH private key |
| Password | Not used |

Register the public key under **Account → Keys**. Keep the private key only in a local key file or a GitHub secret.

## Project conventions

- `main` → GitHub Environment `production`
- `master` → GitHub Environment `production`
- `staging` → GitHub Environment `staging`
- `pb_public/` is the main static site surface
- `pb_hooks/` and `pb_migrations/` are optional
- `.pb_version` pins the PocketBase version
- `.pb_config.json` is the single explicit project config file
- `POCKETHOST_INSTANCE_NAME` is the instance subdomain, not a UUID

## GitHub Environment configuration

Configure both `production` and `staging` with:

- `POCKETHOST_SFTP_USERNAME` as an environment secret
- `POCKETHOST_SFTP_PRIVATE_KEY` as an environment secret
- `POCKETHOST_INSTANCE_NAME` as an environment variable or secret

Optional:

- `HEALTHCHECK_BASE_URL` as an environment variable when the public URL is not the default instance URL

For the demo in this repository, use:

- `production`: `pocketbase-pockethost-skills`
- `staging`: `pocketbase-pockethost-skills-staging`

## Reusable workflow

The public workflow is available at:

```yaml
jobs:
  deploy:
    uses: 8bit-interactive/pocketbase-pockethost-skills/.github/workflows/pockethost-deploy.yml@v0.2.0
    with:
      working-directory: .
    secrets: inherit
```

The workflow uses `wlixcc/SFTP-Deploy-Action@v1.2.6` with `sftp_only: true` and uploads `pb_public`, `pb_hooks`, and `pb_migrations` when present. It never sends an FTP password and does not delete remote files by default.

## Local CLI deployment

For local deployment, use the package CLI:

```bash
export POCKETHOST_SFTP_USERNAME="you@example.com"
export POCKETHOST_SFTP_PRIVATE_KEY_PATH="$HOME/.ssh/pockethost_ed25519"
export POCKETHOST_INSTANCE_NAME="my-instance"
npx pocketbase-pockethost doctor --strict --for deploy
npx pocketbase-pockethost deploy
```

In CI, use `POCKETHOST_SFTP_PRIVATE_KEY` for the multiline private-key secret instead of `POCKETHOST_SFTP_PRIVATE_KEY_PATH`.

## Migration notes

Replace the old `POCKETHOST_FTP_USERNAME`, `POCKETHOST_FTP_PASSWORD`, and port 21 configuration with the SFTP settings above. `ftp:deploy` remains only as a deprecated CLI alias; it now uses SFTP and does not connect through FTP.

Do not use `secure: false`, `basic-ftp`, or `SamKirkland/FTP-Deploy-Action` for new Pockethost deployments.

## Troubleshooting

- `Permission denied (publickey)`: verify that the public Ed25519 key is registered and that the username is the Pockethost email.
- `Connection refused`: verify port `2222`, not `21` or `22`.
- `Invalid format`: preserve the complete multiline private key, including its final newline, in the GitHub secret.
- `Missing instance`: use the Pockethost subdomain in `POCKETHOST_INSTANCE_NAME`.

## References

- [Pockethost SFTP file access documentation](https://pockethost.io/docs/ftp)
- [Pockethost FTPS sunset announcement](https://pockethost.io/blog/ftps-sunset)
