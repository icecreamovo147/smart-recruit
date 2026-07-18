#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import {spawnSync} from "node:child_process";

const args = parseArgs(process.argv.slice(2));
const root = process.cwd();
const manifestPath = args.manifest || "smart-recruit-deploy/repo-export-manifest.json";
const manifest = JSON.parse(fs.readFileSync(manifestPath, "utf8"));
const repositories = manifest.repositories || [];
const defaultBranch = manifest.defaultBranch || "main";
const dryRun = !args.execute;

validateManifest();

if (args.check) {
  console.log(`repo_export_check: PASS (${repositories.length} repositories)`);
  process.exit(0);
}

if (dryRun) {
  for (const repo of repositories) {
    console.log(`dry-run export ${repo.sourceDir} -> ${path.join(args.outputDir || "<output-dir>", repo.repo)} branch=${repo.defaultBranch || defaultBranch}`);
  }
  console.log("repo_export_dry_run: PASS");
  process.exit(0);
}

if (!args.outputDir) {
  fail("--output-dir is required when --execute is used");
}

const outputRoot = path.resolve(root, args.outputDir);
fs.mkdirSync(outputRoot, {recursive: true});
const summary = [];

for (const repo of repositories) {
  const source = path.resolve(root, repo.sourceDir);
  const target = path.join(outputRoot, repo.repo);
  if (fs.existsSync(target)) {
    const entries = fs.readdirSync(target);
    if (entries.length > 0 && !args.force) {
      fail(`refusing to overwrite non-empty export target ${target}; pass --force to replace it`);
    }
    if (args.force) fs.rmSync(target, {recursive: true, force: true});
  }
  copyTree(source, target);
  runGit(["init", "-b", repo.defaultBranch || defaultBranch], target);
  runGit(["add", "."], target);
  runGit(["-c", "user.name=Smart Recruit Export", "-c", "user.email=smart-recruit-export@example.invalid", "commit", "-m", "chore: export initial source root"], target);
  summary.push({
    repo: repo.repo,
    sourceDir: repo.sourceDir,
    target,
    defaultBranch: repo.defaultBranch || defaultBranch,
    buildCommands: repo.buildCommands
  });
}

fs.writeFileSync(path.join(outputRoot, "export-summary.json"), JSON.stringify({schemaVersion: 1, repositories: summary}, null, 2) + "\n");
console.log(`repo_export_execute: PASS (${summary.length} repositories, output=${outputRoot})`);

function parseArgs(argv) {
  const parsed = {execute: false, force: false, check: false};
  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    switch (arg) {
      case "--execute":
        parsed.execute = true;
        break;
      case "--force":
        parsed.force = true;
        break;
      case "--check":
        parsed.check = true;
        break;
      case "--dry-run":
        parsed.execute = false;
        break;
      case "--output-dir":
        parsed.outputDir = argv[++i];
        break;
      case "--manifest":
        parsed.manifest = argv[++i];
        break;
      default:
        fail(`unknown argument: ${arg}`);
    }
  }
  return parsed;
}

function validateManifest() {
  if (!Array.isArray(repositories) || repositories.length === 0) {
    fail("manifest.repositories must be a non-empty array");
  }
  const names = new Set();
  for (const repo of repositories) {
    if (!repo.repo || !repo.sourceDir) fail("each repository requires repo and sourceDir");
    if (names.has(repo.repo)) fail(`duplicate repo ${repo.repo}`);
    names.add(repo.repo);
    const source = path.resolve(root, repo.sourceDir);
    if (!fs.existsSync(path.join(source, "go.mod"))) fail(`${repo.sourceDir} must contain go.mod`);
    if (!Array.isArray(repo.buildCommands) || repo.buildCommands.length === 0) fail(`${repo.repo} must declare buildCommands`);
    if ("remote" in repo || "remoteUrl" in repo) fail(`${repo.repo} must not declare remote repository settings`);
  }
}

function copyTree(source, target) {
  fs.mkdirSync(target, {recursive: true});
  for (const entry of fs.readdirSync(source, {withFileTypes: true})) {
    if (shouldSkip(entry.name)) continue;
    const from = path.join(source, entry.name);
    const to = path.join(target, entry.name);
    if (entry.isDirectory()) {
      copyTree(from, to);
    } else if (entry.isFile()) {
      fs.copyFileSync(from, to);
    } else if (entry.isSymbolicLink()) {
      const link = fs.readlinkSync(from);
      fs.symlinkSync(link, to);
    }
  }
}

function shouldSkip(name) {
  return name === ".git" || name === "node_modules" || name === "dist" || name === "tmp" || name === ".env" || name.endsWith(".env");
}

function runGit(args, cwd) {
  const result = spawnSync("git", args, {cwd, encoding: "utf8"});
  if (result.status !== 0) {
    fail(`git ${args.join(" ")} failed in ${cwd}: ${result.stderr || result.stdout}`);
  }
}

function fail(message) {
  console.error(`repo_export: ${message}`);
  process.exit(1);
}
