import fs from "node:fs/promises";
import path from "node:path";
import { CONFIG_FILE, DEFAULT_BRANCH_ENVIRONMENT_MAP, VERSION_FILE } from "./constants.js";
import { CommandError } from "./errors.js";
import { fileExists, readJson, readText, slugify } from "./utils.js";

export async function findProjectRoot(startDir = process.cwd()) {
  let currentDir = path.resolve(startDir);

  while (true) {
    const configPath = path.join(currentDir, CONFIG_FILE);
    const versionPath = path.join(currentDir, VERSION_FILE);

    if (await fileExists(configPath) || await fileExists(versionPath)) {
      return currentDir;
    }

    const parentDir = path.dirname(currentDir);
    if (parentDir === currentDir) {
      throw new CommandError(`Could not find ${CONFIG_FILE} from ${startDir}`);
    }

    currentDir = parentDir;
  }
}

export async function loadProject(projectRoot) {
  const configPath = path.join(projectRoot, CONFIG_FILE);
  const versionPath = path.join(projectRoot, VERSION_FILE);

  if (!(await fileExists(configPath))) {
    throw new CommandError(`Missing ${CONFIG_FILE} in ${projectRoot}`);
  }

  if (!(await fileExists(versionPath))) {
    throw new CommandError(`Missing ${VERSION_FILE} in ${projectRoot}`);
  }

  const config = await readJson(configPath);
  const version = (await readText(versionPath)).trim();

  return {
    projectRoot,
    configPath,
    versionPath,
    config,
    version
  };
}

export function getBranchEnvironmentMap(config) {
  return {
    ...DEFAULT_BRANCH_ENVIRONMENT_MAP,
    ...(config.pockethost?.branchEnvironmentMap || {})
  };
}

export async function detectCurrentBranch(projectRoot) {
  const gitHeadPath = path.join(projectRoot, ".git", "HEAD");

  if (!(await fileExists(gitHeadPath))) {
    return null;
  }

  const head = (await fs.readFile(gitHeadPath, "utf8")).trim();

  if (head.startsWith("ref: ")) {
    return head.split("/").pop() || null;
  }

  return null;
}

export async function resolveEnvironmentName(project, options = {}) {
  if (options.environment) {
    return options.environment;
  }

  if (process.env.POCKETHOST_ENVIRONMENT) {
    return process.env.POCKETHOST_ENVIRONMENT;
  }

  if (process.env.GITHUB_REF_NAME) {
    const mapped = getBranchEnvironmentMap(project.config)[process.env.GITHUB_REF_NAME];
    if (mapped) {
      return mapped;
    }

    if (options.strictBranchMapping === true) {
      throw new CommandError(`Unsupported branch '${process.env.GITHUB_REF_NAME}'. Use main, master, staging, or pass --env explicitly.`);
    }
  }

  const currentBranch = await detectCurrentBranch(project.projectRoot);
  if (currentBranch) {
    const mapped = getBranchEnvironmentMap(project.config)[currentBranch];
    if (mapped) {
      return mapped;
    }

    if (options.strictBranchMapping === true) {
      throw new CommandError(`Unsupported branch '${currentBranch}'. Use main, master, staging, or pass --env explicitly.`);
    }
  }

  return "production";
}

export function resolveEnvironmentConfig(project, environmentName) {
  return project.config.pockethost?.environments?.[environmentName] || {};
}

export function resolveInstanceName(project, environmentName) {
  const config = resolveEnvironmentConfig(project, environmentName);
  return process.env.POCKETHOST_INSTANCE_NAME
    || config.instanceName
    || process.env.POCKETHOST_TENANT_ID
    || config.tenantId
    || "";
}

export const resolveTenantId = resolveInstanceName;

export function resolveHealthcheckBaseUrl(project, environmentName, instanceName) {
  const environmentConfig = resolveEnvironmentConfig(project, environmentName);
  const rootBaseUrl = project.config.pockethost?.healthcheckBaseUrl || "";
  const environmentBaseUrl = environmentConfig.healthcheckBaseUrl || "";

  if (process.env.HEALTHCHECK_BASE_URL) {
    return process.env.HEALTHCHECK_BASE_URL;
  }

  if (environmentBaseUrl) {
    return environmentBaseUrl;
  }

  if (rootBaseUrl) {
    return rootBaseUrl;
  }

  if (instanceName) {
    return `https://${instanceName}.pockethost.io`;
  }

  return "";
}

export function resolveProjectName(targetDir) {
  return slugify(path.basename(path.resolve(targetDir)));
}

export async function detectProjectSurface(projectRoot) {
  const pbPublic = await fileExists(path.join(projectRoot, "pb_public"));
  const pbHooks = await directoryHasMeaningfulFiles(path.join(projectRoot, "pb_hooks"));
  const pbMigrations = await directoryHasMeaningfulFiles(path.join(projectRoot, "pb_migrations"));
  const packageJson = await fileExists(path.join(projectRoot, "package.json"));

  return {
    pbPublic,
    pbHooks,
    pbMigrations,
    packageJson
  };
}

async function directoryHasMeaningfulFiles(dirPath) {
  if (!(await fileExists(dirPath))) {
    return false;
  }

  const entries = await fs.readdir(dirPath, { withFileTypes: true });

  for (const entry of entries) {
    if (entry.name === ".gitkeep" || entry.name === ".DS_Store") {
      continue;
    }

    if (entry.isFile()) {
      return true;
    }

    if (entry.isDirectory()) {
      const hasNestedFiles = await directoryHasMeaningfulFiles(path.join(dirPath, entry.name));
      if (hasNestedFiles) {
        return true;
      }
    }
  }

  return false;
}
