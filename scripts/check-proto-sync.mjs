#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import process from "node:process";

const root = path.resolve(new URL("..", import.meta.url).pathname);
const toolVersionsPath = "smart-recruit-proto/scripts/tool-versions.env";

const fileSets = [
  {
    label: "proto source",
    canonical: "smart-recruit-proto/proto/recruitment.proto",
    mirrors: [],
  },
  {
    label: "generated message code",
    canonical: "smart-recruit-proto/recruitment/pb/recruitment.pb.go",
    mirrors: [],
  },
  {
    label: "generated grpc code",
    canonical: "smart-recruit-proto/recruitment/pb/recruitment_grpc.pb.go",
    mirrors: [],
  },
];

function readToolVersions() {
  const text = fs.readFileSync(path.join(root, toolVersionsPath), "utf8");
  const versions = Object.create(null);

  for (const rawLine of text.split(/\r?\n/)) {
    const line = rawLine.trim();
    if (!line || line.startsWith("#")) continue;
    const match = line.match(/^([A-Z0-9_]+)=([A-Za-z0-9._-]+)$/);
    if (!match) throw new Error(`invalid tool version entry: ${rawLine}`);
    versions[match[1]] = match[2];
  }

  for (const key of ["PROTOC_GENERATED_VERSION", "PROTOC_GEN_GO_VERSION", "PROTOC_GEN_GO_GRPC_VERSION"]) {
    if (!versions[key]) throw new Error(`missing ${key} in ${toolVersionsPath}`);
  }
  return versions;
}

function generatedHeaderChecks(versions) {
  return [
    {
      file: "smart-recruit-proto/recruitment/pb/recruitment.pb.go",
      expected: [
        `// \tprotoc-gen-go v${versions.PROTOC_GEN_GO_VERSION}`,
        `// \tprotoc        v${versions.PROTOC_GENERATED_VERSION}`,
      ],
    },
    {
      file: "smart-recruit-proto/recruitment/pb/recruitment_grpc.pb.go",
      expected: [
        `// - protoc-gen-go-grpc v${versions.PROTOC_GEN_GO_GRPC_VERSION}`,
        `// - protoc             v${versions.PROTOC_GENERATED_VERSION}`,
      ],
    },
  ];
}

function parseArgs(argv) {
  const args = { sync: false };
  for (const arg of argv) {
    if (arg === "--sync") args.sync = true;
    else if (arg === "--check") args.sync = false;
    else throw new Error(`unknown argument: ${arg}`);
  }
  return args;
}

function read(relativePath) {
  return fs.readFileSync(path.join(root, relativePath));
}

function write(relativePath, bytes) {
  fs.mkdirSync(path.dirname(path.join(root, relativePath)), { recursive: true });
  fs.writeFileSync(path.join(root, relativePath), bytes);
}

function main() {
  const args = parseArgs(process.argv.slice(2));
  const drift = [];
  const versions = readToolVersions();

  for (const set of fileSets) {
    const canonicalBytes = read(set.canonical);
    if (canonicalBytes.length === 0) {
      drift.push({ label: set.label, canonical: set.canonical, mirror: "(empty canonical)" });
    }
    for (const mirror of set.mirrors) {
      const mirrorBytes = read(mirror);
      if (Buffer.compare(canonicalBytes, mirrorBytes) !== 0) {
        drift.push({ label: set.label, canonical: set.canonical, mirror, repairable: true });
        if (args.sync) write(mirror, canonicalBytes);
      }
    }
  }

  for (const check of generatedHeaderChecks(versions)) {
    const generated = read(check.file).toString("utf8");
    for (const expected of check.expected) {
      if (!generated.includes(expected)) {
        drift.push({
          label: "generated toolchain",
          canonical: toolVersionsPath,
          mirror: check.file,
          detail: `missing expected header ${JSON.stringify(expected)} from ${toolVersionsPath}`,
          repairable: false,
        });
      }
    }
  }

  if (drift.length === 0) {
    console.log("proto_sync_result: PASS");
    return;
  }

  for (const item of drift) {
    if (item.detail) console.error(`proto drift: ${item.label}: ${item.mirror}: ${item.detail}`);
    else console.error(`proto drift: ${item.label}: ${item.mirror} differs from ${item.canonical}`);
  }
  if (args.sync && drift.every((item) => item.repairable === true)) {
    console.log(`proto_sync_result: SYNCED (${drift.length} file(s))`);
    return;
  }
  console.error("proto_sync_result: FAIL");
  process.exit(1);
}

try {
  main();
} catch (error) {
  console.error(`proto_sync_error: ${error.message}`);
  process.exit(2);
}
