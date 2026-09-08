# GitHub Actions deployment for PocketHost

The recommended workflow downloads a pinned `pocketbase-pockethost` release and runs the same commands used locally:

1. check out the repository;
2. install the pinned binary and verify its checksum;
3. run `pocketbase-pockethost doctor`;
4. run `pocketbase-pockethost validate`;
5. run `pocketbase-pockethost deploy`;
6. run `pocketbase-pockethost health`.

## Environment configuration

Create GitHub Environments named `production` and `staging`.

Required secrets:

- `POCKETHOST_SFTP_USERNAME`
- `POCKETHOST_SFTP_PRIVATE_KEY`

Required variables:

- `POCKETHOST_INSTANCE_NAME`
- `POCKETHOST_SFTP_HOST_KEY`

Optional variables:

- `HEALTHCHECK_BASE_URL`
- `POCKETHOST_SFTP_HOST`
- `POCKETHOST_SFTP_PORT`

The private key must be an Ed25519 OpenSSH key registered in PocketHost Account → Keys. The host key variable must contain the documented PocketHost host fingerprint or an authorized public-key line. The CLI rejects unpinned SFTP hosts.

## Project conventions

- `main` → `production`
- `master` → `production`
- `staging` → `staging`
- `.pb_version` pins the PocketBase release;
- `.pb_config.json` is the only project-specific configuration file.

The generated workflow is intentionally small. It delegates validation, checksum verification, SFTP synchronization, and health checks to the binary so local and CI deployments share the same behavior.

## Synchronization behavior

The CLI maintains `.ftp-deploy-sync-state.json` in the remote instance directory for compatibility with the established FTP-Deploy-Action manifest format. It uploads new and changed files and removes only files previously managed by the manifest. Unknown remote files and PocketBase data remain untouched.
