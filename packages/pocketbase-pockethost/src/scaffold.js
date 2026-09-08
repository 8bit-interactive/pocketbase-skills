import path from "node:path";
import { fileURLToPath } from "node:url";
import { DEFAULT_POCKETBASE_VERSION } from "./constants.js";
import { CommandError } from "./errors.js";
import { copyDirectory, emptyDirectory, ensureDir, listVisibleEntries, writeJson, writeText } from "./utils.js";
import { installWorkflow } from "./workflow.js";
import { resolveProjectName } from "./project.js";

const CURRENT_DIR = path.dirname(fileURLToPath(import.meta.url));
const TEMPLATE_DIR = path.resolve(CURRENT_DIR, "../templates/project");

export async function scaffoldProject(targetDir, packageVersion, { force = false } = {}) {
  const resolvedTargetDir = path.resolve(targetDir);
  await ensureDir(resolvedTargetDir);

  if (!force && !(await emptyDirectory(resolvedTargetDir))) {
    const entries = await listVisibleEntries(resolvedTargetDir);
    throw new CommandError(`Target directory is not empty: ${resolvedTargetDir}\nFound: ${entries.join(", ")}`);
  }

  const projectName = resolveProjectName(resolvedTargetDir);
  const projectTitle = projectName
    .split("-")
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");

  await copyDirectory(TEMPLATE_DIR, resolvedTargetDir, {
    "__PROJECT_NAME__": projectName,
    "__PROJECT_TITLE__": projectTitle,
    "__CLI_VERSION__": packageVersion,
    "__POCKETBASE_VERSION__": DEFAULT_POCKETBASE_VERSION
  });

  const config = {
    projectName,
    spaMountPath: "page",
    pockethost: {
      branchEnvironmentMap: {
        main: "production",
        master: "production",
        staging: "staging"
      },
      environments: {
        production: {
          instanceName: "",
          healthcheckBaseUrl: ""
        },
        staging: {
          instanceName: "",
          healthcheckBaseUrl: ""
        }
      }
    },
    tests: {
      lintJs: false,
      lintCss: false,
      unit: false,
      e2e: false
    }
  };

  await writeJson(path.join(resolvedTargetDir, ".pb_config.json"), config);
  await writeText(path.join(resolvedTargetDir, ".pb_version"), `${DEFAULT_POCKETBASE_VERSION}\n`);
  await installWorkflow(resolvedTargetDir, { force: true });

  return resolvedTargetDir;
}
