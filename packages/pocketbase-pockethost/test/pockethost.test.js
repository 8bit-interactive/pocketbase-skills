import assert from "node:assert/strict";
import fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { deployProject } from "../src/deploy.js";
import { resolveInstanceName } from "../src/project.js";
import { renderDeployWorkflow } from "../src/workflow.js";

test("resolves the configured Pockethost instance name", () => {
  const project = {
    config: {
      pockethost: {
        environments: {
          staging: {
            instanceName: "staging-instance"
          }
        }
      }
    }
  };

  assert.equal(resolveInstanceName(project, "staging"), "staging-instance");
});

test("keeps legacy tenant configuration as a fallback", () => {
  const project = {
    config: {
      pockethost: {
        environments: {
          production: {
            tenantId: "legacy-instance"
          }
        }
      }
    }
  };

  assert.equal(resolveInstanceName(project, "production"), "legacy-instance");
});

test("dry-run resolves SFTP deployment paths without credentials", async () => {
  const projectRoot = await fs.mkdtemp(path.join(os.tmpdir(), "pockethost-sftp-test-"));
  await fs.mkdir(path.join(projectRoot, "pb_public"));

  const result = await deployProject({
    projectRoot,
    config: {
      pockethost: {
        environments: {
          staging: {
            instanceName: "staging-instance"
          }
        }
      }
    }
  }, {
    environment: "staging",
    dryRun: true
  });

  assert.equal(result.sftpHost, "ftp.pockethost.io");
  assert.equal(result.sftpPort, 2222);
  assert.equal(result.publicDir, "/staging-instance/pb_public");
  assert.equal(result.hooksDir, "/staging-instance/pb_hooks");
  assert.equal(result.migrationsDir, "/staging-instance/pb_migrations");
});

test("generated workflows use SFTP credentials and port", () => {
  const workflow = renderDeployWorkflow();

  assert.match(workflow, /POCKETHOST_SFTP_USERNAME/);
  assert.match(workflow, /POCKETHOST_SFTP_PRIVATE_KEY/);
  assert.match(workflow, /POCKETHOST_INSTANCE_NAME/);
  assert.doesNotMatch(workflow, /POCKETHOST_FTP_PASSWORD/);
  assert.doesNotMatch(workflow, /FTP-Deploy-Action/);
});
