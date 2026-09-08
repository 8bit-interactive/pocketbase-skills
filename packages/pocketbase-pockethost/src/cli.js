import fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { parseArgs } from "node:util";
import { fileURLToPath } from "node:url";
import { CommandError } from "./errors.js";
import { runDoctor } from "./doctor.js";
import { loadProject, findProjectRoot, resolveEnvironmentName } from "./project.js";
import { deployProject, runHealthcheck } from "./deploy.js";
import { ensurePocketBaseInstalled, defaultDevArgs, runPocketBase, runPocketBaseMigrate } from "./pocketbase.js";
import { scaffoldProject } from "./scaffold.js";
import { installWorkflow } from "./workflow.js";
import { runCommand, slugify, timestampForMigration, writeText } from "./utils.js";

const CURRENT_DIR = path.dirname(fileURLToPath(import.meta.url));
const PACKAGE_JSON_PATH = path.resolve(CURRENT_DIR, "../package.json");

function printHelp() {
  console.log(`pocketbase-pockethost

Usage:
  pocketbase-pockethost init [directory]
  pocketbase-pockethost install
  pocketbase-pockethost dev [--http 127.0.0.1:8090]
  pocketbase-pockethost test
  pocketbase-pockethost doctor [--strict] [--for deploy] [--env production]
  pocketbase-pockethost health [--env production]
  pocketbase-pockethost deploy [--env production] [--dry-run]
  pocketbase-pockethost sftp:deploy [--env production] [--dry-run]
  pocketbase-pockethost ftp:deploy [--env production] [--dry-run] (deprecated alias)
  pocketbase-pockethost workflow:install [--force]
  pocketbase-pockethost migration:new <name>
  pocketbase-pockethost hooks:new <name>
`);
}

async function readPackageVersion() {
  const pkg = JSON.parse(await fs.readFile(PACKAGE_JSON_PATH, "utf8"));
  return pkg.version;
}

async function loadCurrentProject() {
  const projectRoot = await findProjectRoot(process.cwd());
  return loadProject(projectRoot);
}

async function commandInit(positionals, options) {
  const targetDir = positionals[0] ? path.resolve(positionals[0]) : process.cwd();
  const packageVersion = await readPackageVersion();
  const result = await scaffoldProject(targetDir, packageVersion, { force: options.force === true });

  console.log(`Scaffolded project in ${result}`);
  console.log("Next steps:");
  console.log("1. npm install");
  console.log("2. npm run check");
  console.log("3. Edit pb_public/index.html");
  console.log("4. npm run dev");
  console.log("5. Push staging, then main");
}

async function commandInstall() {
  const project = await loadCurrentProject();
  const binaryPath = await ensurePocketBaseInstalled(project.projectRoot, project.version);
  console.log(`PocketBase installed at ${binaryPath}`);
}

async function commandDev(options) {
  const project = await loadCurrentProject();
  const httpAddress = options.http || "127.0.0.1:8090";
  await runPocketBase(project.projectRoot, project.version, defaultDevArgs(project.projectRoot, httpAddress));
}

async function commandTest() {
  const project = await loadCurrentProject();
  const doctorOptions = { strict: true, forDeploy: false };

  await runDoctor(project, doctorOptions);

  const tempDir = await fs.mkdtemp(path.join(os.tmpdir(), "pb-test-"));
  const dataDir = path.join(tempDir, "pb_data");
  await fs.mkdir(dataDir, { recursive: true });

  console.log("Running migration smoke test...");
  await runPocketBaseMigrate(project.projectRoot, project.version, dataDir);

  if (project.config.tests?.lintJs) {
    await runCommand("npm", ["run", "lint:js"], { cwd: project.projectRoot, env: process.env });
  } else {
    console.log("Skipping JS lint: disabled.");
  }

  if (project.config.tests?.lintCss) {
    await runCommand("npm", ["run", "lint:css"], { cwd: project.projectRoot, env: process.env });
  } else {
    console.log("Skipping CSS lint: disabled.");
  }

  if (project.config.tests?.unit) {
    await runCommand("npm", ["run", "test:unit"], { cwd: project.projectRoot, env: process.env });
  } else {
    console.log("Skipping unit tests: disabled.");
  }

  if (project.config.tests?.e2e) {
    await runCommand("npm", ["run", "test:e2e"], { cwd: project.projectRoot, env: process.env });
  } else {
    console.log("Skipping Playwright e2e: disabled.");
  }

  console.log("Test run passed.");
}

async function commandDoctor(options) {
  const project = await loadCurrentProject();
  const forDeploy = options.for === "deploy";
  await runDoctor(project, {
    strict: options.strict === true,
    forDeploy,
    environment: options.env,
    strictBranchMapping: forDeploy
  });
}

async function commandHealth(options) {
  const project = await loadCurrentProject();
  const environmentName = await resolveEnvironmentName(project, {
    environment: options.env,
    strictBranchMapping: true
  });

  await runHealthcheck(project, environmentName);
  console.log(`Healthcheck passed for environment '${environmentName}'.`);
}

async function commandDeploy(options) {
  const project = await loadCurrentProject();
  const result = await deployProject(project, {
    environment: options.env,
    dryRun: options["dry-run"] === true,
    strictBranchMapping: true
  });

  console.log(`Deployment target environment: ${result.environmentName}`);
  console.log(`SFTP host: ${result.sftpHost}:${result.sftpPort}`);
  console.log(`Pockethost instance: ${result.instanceName}`);
  console.log(`pb_public -> ${result.publicDir}`);

  if (result.surface.pbHooks) {
    console.log(`pb_hooks -> ${result.hooksDir}`);
  }

  if (result.surface.pbMigrations) {
    console.log(`pb_migrations -> ${result.migrationsDir}`);
  }

  if (options["dry-run"] === true) {
    console.log("Dry run completed. No files were uploaded.");
  } else {
    console.log("Deployment completed successfully.");
  }
}

async function commandWorkflowInstall(options) {
  const project = await loadCurrentProject();
  const workflowPath = await installWorkflow(project.projectRoot, {
    force: options.force === true
  });

  console.log(`Installed workflow at ${workflowPath}`);
}

async function commandMigrationNew(positionals) {
  const name = positionals[0];
  if (!name) {
    throw new CommandError("migration:new requires a name.");
  }

  const project = await loadCurrentProject();
  const slug = slugify(name).replace(/-/g, "_");
  const filename = `${timestampForMigration()}_${slug}.js`;
  const targetPath = path.join(project.projectRoot, "pb_migrations", filename);

  const content = `/// <reference path="../pb_data/types.d.ts" />

migrate((db) => {
  // TODO: apply migration "${name}"
}, (db) => {
  // TODO: rollback migration "${name}"
})
`;

  await writeText(targetPath, content);
  console.log(`Created migration ${targetPath}`);
}

async function commandHooksNew(positionals) {
  const name = positionals[0];
  if (!name) {
    throw new CommandError("hooks:new requires a name.");
  }

  const project = await loadCurrentProject();
  const slug = slugify(name);
  const targetPath = path.join(project.projectRoot, "pb_hooks", `${slug}.js`);
  const content = `function register${slug.replace(/(^|-)([a-z])/g, (_match, _separator, letter) => letter.toUpperCase())}Hook() {
  // TODO: register routes or hooks for "${name}"
}

register${slug.replace(/(^|-)([a-z])/g, (_match, _separator, letter) => letter.toUpperCase())}Hook()
`;

  await writeText(targetPath, content);
  console.log(`Created hook ${targetPath}`);
}

export async function runCli(argv) {
  const command = argv[0];
  const args = argv.slice(1);

  if (!command || command === "--help" || command === "-h" || command === "help") {
    printHelp();
    return;
  }

  const { values, positionals } = parseArgs({
    args,
    allowPositionals: true,
    options: {
      env: { type: "string" },
      force: { type: "boolean" },
      strict: { type: "boolean" },
      "dry-run": { type: "boolean" },
      http: { type: "string" },
      for: { type: "string" }
    }
  });

  switch (command) {
    case "init":
      await commandInit(positionals, values);
      return;
    case "install":
      await commandInstall();
      return;
    case "dev":
      await commandDev(values);
      return;
    case "test":
      await commandTest();
      return;
    case "doctor":
      await commandDoctor(values);
      return;
    case "health":
      await commandHealth(values);
      return;
    case "deploy":
    case "sftp:deploy":
    case "ftp:deploy":
      await commandDeploy(values);
      return;
    case "workflow:install":
      await commandWorkflowInstall(values);
      return;
    case "migration:new":
      await commandMigrationNew(positionals);
      return;
    case "hooks:new":
      await commandHooksNew(positionals);
      return;
    default:
      throw new CommandError(`Unknown command: ${command}`);
  }
}
