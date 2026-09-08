# pocketbase-pockethost

Ultra-simple PocketBase and Pockethost automation CLI.

This Node package is a transitional compatibility CLI. New projects should use the statically linked Go `pocketbase-pockethost` binary for PocketHost SFTP deployment.

Main commands:

- `init`
- `install`
- `dev`
- `test`
- `doctor`
- `health`
- `deploy`
- `deploy` (legacy FTP compatibility)
- `workflow:install`
- `migration:new`
- `hooks:new`

The default project model is zero-build:

- edit `pb_public/index.html`
- edit `pb_public/assets/site.css`
- keep hooks and migrations optional
