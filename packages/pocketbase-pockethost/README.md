# pocketbase-pockethost

Ultra-simple PocketBase and Pockethost automation CLI.

Main commands:

- `init`
- `install`
- `dev`
- `test`
- `doctor`
- `health`
- `deploy`
- `sftp:deploy`
- `workflow:install`
- `migration:new`
- `hooks:new`

The default project model is zero-build:

- edit `pb_public/index.html`
- edit `pb_public/assets/site.css`
- keep hooks and migrations optional

Deployments use SFTP on `ftp.pockethost.io:2222` with an Ed25519 private key. Configure `POCKETHOST_SFTP_USERNAME`, `POCKETHOST_SFTP_PRIVATE_KEY` or `POCKETHOST_SFTP_PRIVATE_KEY_PATH`, and `POCKETHOST_INSTANCE_NAME` before deploying.
