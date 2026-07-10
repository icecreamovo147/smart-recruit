#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import process from "node:process";

export const CLASSIFICATIONS = Object.freeze({
  CURRENT: "current",
  LEGACY_COMPATIBLE: "legacy-compatible",
  UNSUPPORTED: "unsupported",
});

function readJson(file, issues, label) {
  try {
    return JSON.parse(fs.readFileSync(file, "utf8"));
  } catch (error) {
    issues.push(`${label} is not valid JSON: ${error.message}`);
    return null;
  }
}

function validateStringList(values, label, issues) {
  if (!Array.isArray(values)) {
    issues.push(`${label} must be an array`);
    return;
  }
  for (const value of values) {
    if (typeof value !== "string" || value.length === 0) issues.push(`${label} items must be non-empty strings`);
  }
}

export function compileGlob(pattern) {
  if (typeof pattern !== "string" || pattern.length === 0) {
    throw new Error("glob pattern must be a non-empty string");
  }
  if (pattern.includes("[") || pattern.includes("]")) {
    throw new Error(`unsupported character class syntax in glob: ${pattern}`);
  }
  let source = "^";
  for (let index = 0; index < pattern.length; index += 1) {
    const character = pattern[index];
    if (character === "*" && pattern[index + 1] === "*") {
      source += ".*";
      index += 1;
    } else if (character === "*") {
      source += "[^/]*";
    } else if (character === "?") {
      source += "[^/]";
    } else {
      source += character.replace(/[|\\{}()[\]^$+?.]/g, "\\$&");
    }
  }
  return new RegExp(`${source}$`);
}

export function validateFeature(featureDir, options = {}) {
  const absoluteDir = path.resolve(featureDir);
  const featureName = path.basename(absoluteDir);
  const issues = [];
  const warnings = [];
  const requiredFiles = [
    `${featureName}-SPEC.md`,
    `${featureName}-SDD.md`,
    "TASKS.md",
    "AGENT_RULES.md",
    "task-scope.json",
    "prompts/implement-task.md",
    "prompts/self-review.md",
    "prompts/fix-check-failures.md",
    "scripts/check-task-scope.sh",
    "scripts/agent-check.sh",
  ];
  const requiredDirectories = ["acceptance", "prompts", "scripts", "reports"];

  if (!fs.existsSync(absoluteDir) || !fs.statSync(absoluteDir).isDirectory()) {
    return {
      feature_name: featureName,
      feature_dir: absoluteDir,
      classification: CLASSIFICATIONS.UNSUPPORTED,
      issues: [`feature directory does not exist: ${absoluteDir}`],
      warnings,
      task_ids: [],
    };
  }

  for (const relative of requiredFiles) {
    if (!fs.existsSync(path.join(absoluteDir, relative))) issues.push(`missing required file: ${relative}`);
  }
  for (const relative of requiredDirectories) {
    const candidate = path.join(absoluteDir, relative);
    if (!fs.existsSync(candidate) || !fs.statSync(candidate).isDirectory()) {
      issues.push(`missing required directory: ${relative}/`);
    }
  }

  const scopePath = path.join(absoluteDir, "task-scope.json");
  const scope = fs.existsSync(scopePath) ? readJson(scopePath, issues, "task-scope.json") : null;
  let classification = CLASSIFICATIONS.CURRENT;
  const taskIds = [];

  if (scope) {
    if (scope.schemaVersion === 1 && scope.feature_name === featureName && scope.tasks && typeof scope.tasks === "object" && !Array.isArray(scope.tasks)) {
      classification = CLASSIFICATIONS.CURRENT;
    } else if (scope.schemaVersion === undefined && scope.feature_name === featureName && scope.tasks && typeof scope.tasks === "object" && !Array.isArray(scope.tasks)) {
      classification = CLASSIFICATIONS.LEGACY_COMPATIBLE;
      warnings.push("task-scope.json has no schemaVersion; treating feature as legacy-compatible");
    } else {
      classification = CLASSIFICATIONS.UNSUPPORTED;
      issues.push("unsupported task-scope.json shape; expected schemaVersion 1 with feature_name + tasks");
    }

    if (scope.tasks && typeof scope.tasks === "object" && !Array.isArray(scope.tasks)) {
      const tasksMarkdownPath = path.join(absoluteDir, "TASKS.md");
      const tasksMarkdown = fs.existsSync(tasksMarkdownPath) ? fs.readFileSync(tasksMarkdownPath, "utf8") : "";
      for (const [taskId, task] of Object.entries(scope.tasks)) {
        taskIds.push(taskId);
        if (!tasksMarkdown.includes(`## ${taskId} -`)) issues.push(`TASKS.md is missing section for ${taskId}`);
        if (!task || typeof task !== "object") {
          issues.push(`invalid task entry: ${taskId}`);
          continue;
        }
        if (!Array.isArray(task.allowedFiles) || task.allowedFiles.length === 0) issues.push(`${taskId} has no allowedFiles`);
        if (!Array.isArray(task.forbiddenFiles)) issues.push(`${taskId} forbiddenFiles must be an array`);
        for (const pattern of [...(task.allowedFiles || []), ...(task.forbiddenFiles || [])]) {
          try {
            compileGlob(pattern);
          } catch (error) {
            issues.push(`${taskId} has invalid glob: ${error.message}`);
          }
        }
        if ("requiredKnowledgeImpact" in task && typeof task.requiredKnowledgeImpact !== "boolean") {
          issues.push(`${taskId} requiredKnowledgeImpact must be boolean when present`);
        }
        if ("knowledge" in task) {
          if (!task.knowledge || typeof task.knowledge !== "object" || Array.isArray(task.knowledge)) {
            issues.push(`${taskId} knowledge must be an object when present`);
          } else {
            for (const key of Object.keys(task.knowledge)) {
              if (!["review", "modify", "candidate"].includes(key)) issues.push(`${taskId} knowledge has unknown key ${key}`);
            }
            for (const key of ["review", "modify", "candidate"]) {
              if (key in task.knowledge) {
                validateStringList(task.knowledge[key], `${taskId} knowledge.${key}`, issues);
                for (const pattern of task.knowledge[key] || []) {
                  try {
                    compileGlob(pattern);
                  } catch (error) {
                    issues.push(`${taskId} knowledge.${key} has invalid glob: ${error.message}`);
                  }
                }
              }
            }
          }
        }
        const expectedAcceptance = `.spec/${featureName}/acceptance/${taskId}.md`;
        if (task.acceptance !== expectedAcceptance) issues.push(`${taskId} acceptance path must be ${expectedAcceptance}`);
        if (!fs.existsSync(path.join(absoluteDir, "acceptance", `${taskId}.md`))) issues.push(`missing acceptance file for ${taskId}`);
        const expectedReport = `.spec/${featureName}/reports/${taskId}-report.md`;
        if (task.report !== expectedReport) issues.push(`${taskId} report path must be ${expectedReport}`);
      }
    }
  }

  const pipelinePath = path.join(absoluteDir, "pipeline-state.json");
  if (options.requirePipeline && !fs.existsSync(pipelinePath)) {
    issues.push("missing pipeline-state.json for pipeline execution");
  } else if (fs.existsSync(pipelinePath)) {
    const state = readJson(pipelinePath, issues, "pipeline-state.json");
    if (state && state.feature_name !== featureName) issues.push("pipeline-state.json feature_name does not match directory");
  }

  if (issues.length > 0) classification = CLASSIFICATIONS.UNSUPPORTED;
  return { feature_name: featureName, feature_dir: absoluteDir, classification, issues, warnings, task_ids: taskIds.sort() };
}

function parseArgs(argv) {
  const result = { feature: null, requirePipeline: false, json: false };
  for (let index = 0; index < argv.length; index += 1) {
    if (argv[index] === "--feature") result.feature = argv[++index];
    else if (argv[index] === "--require-pipeline") result.requirePipeline = true;
    else if (argv[index] === "--json") result.json = true;
    else throw new Error(`unknown argument: ${argv[index]}`);
  }
  if (!result.feature) throw new Error("--feature is required");
  return result;
}

if (import.meta.url === `file://${process.argv[1]}`) {
  try {
    const args = parseArgs(process.argv.slice(2));
    const result = validateFeature(args.feature, { requirePipeline: args.requirePipeline });
    if (args.json) console.log(JSON.stringify(result, null, 2));
    else {
      console.log(`feature: ${result.feature_name}`);
      console.log(`classification: ${result.classification}`);
      for (const warning of result.warnings) console.log(`warning: ${warning}`);
      for (const issue of result.issues) console.error(`error: ${issue}`);
    }
    process.exit(result.classification === CLASSIFICATIONS.UNSUPPORTED ? 1 : 0);
  } catch (error) {
    console.error(`error: ${error.message}`);
    process.exit(2);
  }
}
