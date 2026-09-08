# Pockethost skills demo

This zero-build site demonstrates the public reusable SFTP workflow from the parent repository.

Deployment mapping:

- `staging` → `https://pocketbase-pockethost-skills-staging.pockethost.io`
- `main` → `https://pocketbase-pockethost-skills.pockethost.io`

The workflow expects these values in the selected GitHub Environment:

- `POCKETHOST_SFTP_USERNAME` as a secret
- `POCKETHOST_SFTP_PRIVATE_KEY` as a secret
- `POCKETHOST_INSTANCE_NAME` as a variable or secret
