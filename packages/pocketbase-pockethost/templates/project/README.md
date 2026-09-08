# __PROJECT_TITLE__

This project uses the `pocketbase-pockethost` project conventions. New deployments should install the Go `pocketbase-pockethost` binary.

Most users should only edit:

- `pb_public/index.html`
- `pb_public/assets/site.css`

Quick start:

1. `npm install`
2. `npm run check`
3. `npm run dev`
4. edit `pb_public/index.html`
5. push `staging`
6. push `main`

GitHub Environments:

- `staging`
- `production`

Required GitHub Environment values:

- `POCKETHOST_SFTP_USERNAME` as a secret
- `POCKETHOST_SFTP_PRIVATE_KEY` as a secret
- `POCKETHOST_SFTP_HOST_KEY` as a variable
- `POCKETHOST_INSTANCE_NAME` as a variable or secret
