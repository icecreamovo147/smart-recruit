#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import process from "node:process";

const root = path.resolve(new URL("..", import.meta.url).pathname);

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

  for (const set of fileSets) {
    const canonicalBytes = read(set.canonical);
    if (canonicalBytes.length === 0) {
      drift.push({ label: set.label, canonical: set.canonical, mirror: "(empty canonical)" });
    }
    for (const mirror of set.mirrors) {
      const mirrorBytes = read(mirror);
      if (Buffer.compare(canonicalBytes, mirrorBytes) !== 0) {
        drift.push({ label: set.label, canonical: set.canonical, mirror });
        if (args.sync) write(mirror, canonicalBytes);
      }
    }
  }

  if (drift.length === 0) {
    console.log("proto_sync_result: PASS");
    return;
  }

  for (const item of drift) {
    console.error(`proto drift: ${item.label}: ${item.mirror} differs from ${item.canonical}`);
  }
  if (args.sync) {
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
