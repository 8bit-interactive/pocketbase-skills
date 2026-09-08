import fs from "node:fs/promises";
import path from "node:path";
import SftpClient from "ssh2-sftp-client";
import { CommandError } from "./errors.js";
import { detectProjectSurface, resolveEnvironmentName, resolveHealthcheckBaseUrl, resolveInstanceName } from "./project.js";

async function deployDirectory(client, localDir, remoteDir) {
  await client.uploadDir(localDir, remoteDir);
}

async function resolvePrivateKey() {
  if (process.env.POCKETHOST_SFTP_PRIVATE_KEY) {
    return process.env.POCKETHOST_SFTP_PRIVATE_KEY.replace(/\\r\\n/g, "\\n");
  }

  if (process.env.POCKETHOST_SFTP_PRIVATE_KEY_PATH) {
    return fs.readFile(process.env.POCKETHOST_SFTP_PRIVATE_KEY_PATH, "utf8");
  }

  return "";
}

function resolveSftpPort() {
  const value = process.env.POCKETHOST_SFTP_PORT || "2222";
  const port = Number(value);

  if (!Number.isInteger(port) || port <= 0) {
    throw new CommandError(`Invalid POCKETHOST_SFTP_PORT: ${value}`);
  }

  return port;
}

export async function runHealthcheck(project, environmentName) {
  const instanceName = resolveInstanceName(project, environmentName);
  const baseUrl = resolveHealthcheckBaseUrl(project, environmentName, instanceName);
  const indexHtml = await fs.readFile(path.join(project.projectRoot, "pb_public", "index.html"), "utf8");
  const expectedHeadingMatch = indexHtml.match(/<h1>(.*?)<\/h1>/is);
  const expectedHeading = expectedHeadingMatch ? expectedHeadingMatch[1].trim() : "";

  if (!baseUrl) {
    throw new CommandError(`Missing healthcheck base URL for environment '${environmentName}'. Set HEALTHCHECK_BASE_URL or POCKETHOST_INSTANCE_NAME.`);
  }

  if (!expectedHeading) {
    throw new CommandError("Could not extract an <h1> marker from pb_public/index.html.");
  }

  const response = await fetch(baseUrl, {
    headers: {
      "User-Agent": "pocketbase-pockethost"
    }
  });

  if (!response.ok) {
    throw new CommandError(`Healthcheck failed for ${baseUrl}: ${response.status} ${response.statusText}`);
  }

  const html = await response.text();
  if (!html.includes(expectedHeading)) {
    throw new CommandError(`Healthcheck failed for ${baseUrl}: expected page marker '${expectedHeading}' was not found.`);
  }
}

export async function deployProject(project, options = {}) {
  const environmentName = await resolveEnvironmentName(project, options);
  const instanceName = resolveInstanceName(project, environmentName);
  const sftpUsername = process.env.POCKETHOST_SFTP_USERNAME || "";
  const sftpHost = process.env.POCKETHOST_SFTP_HOST || "ftp.pockethost.io";
  const sftpPort = resolveSftpPort();
  const dryRun = options.dryRun === true;
  const surface = await detectProjectSurface(project.projectRoot);
  const sftpPrivateKey = dryRun ? "dry-run" : await resolvePrivateKey();

  if (!instanceName) {
    throw new CommandError(`Missing POCKETHOST_INSTANCE_NAME for environment '${environmentName}'. Configure the Pockethost instance subdomain.`);
  }

  if (!dryRun && !sftpUsername) {
    throw new CommandError(`Missing POCKETHOST_SFTP_USERNAME for environment '${environmentName}'.`);
  }

  if (!dryRun && !sftpPrivateKey) {
    throw new CommandError(`Missing POCKETHOST_SFTP_PRIVATE_KEY or POCKETHOST_SFTP_PRIVATE_KEY_PATH for environment '${environmentName}'.`);
  }

  const publicDir = `/${instanceName}/pb_public`;
  const hooksDir = `/${instanceName}/pb_hooks`;
  const migrationsDir = `/${instanceName}/pb_migrations`;

  if (dryRun) {
    return {
      environmentName,
      instanceName,
      sftpHost,
      sftpPort,
      publicDir,
      hooksDir,
      migrationsDir,
      surface
    };
  }

  const client = new SftpClient();

  try {
    await client.connect({
      host: sftpHost,
      port: sftpPort,
      username: sftpUsername,
      privateKey: sftpPrivateKey
    });

    if (surface.pbPublic) {
      console.log(`Uploading pb_public -> ${publicDir}`);
      await deployDirectory(client, path.join(project.projectRoot, "pb_public"), publicDir);
    }

    if (surface.pbHooks) {
      console.log(`Uploading pb_hooks -> ${hooksDir}`);
      await deployDirectory(client, path.join(project.projectRoot, "pb_hooks"), hooksDir);
    }

    if (surface.pbMigrations) {
      console.log(`Uploading pb_migrations -> ${migrationsDir}`);
      await deployDirectory(client, path.join(project.projectRoot, "pb_migrations"), migrationsDir);
    }
  } finally {
    await client.end();
  }

  if (surface.pbPublic) {
    await runHealthcheck(project, environmentName);
  }

  return {
    environmentName,
    instanceName,
    sftpHost,
    sftpPort,
    publicDir,
    hooksDir,
    migrationsDir,
    surface
  };
}
