#!/usr/bin/env node
"use strict";

const fs = require("node:fs");
const path = require("node:path");
const os = require("node:os");

const claudeDir = process.argv[2] || path.join(os.tmpdir(), "claude-demo");
const projectsDir = path.join(claudeDir, "projects");

const projects = [
  { name: "-home-user-my-web-app",     files: 4,  fileSize: 48_000,  daysAgo: 0  },
  { name: "-home-user-api-server",      files: 12, fileSize: 95_000,  daysAgo: 2  },
  { name: "-home-user-data-pipeline",   files: 2,  fileSize: 22_000,  daysAgo: 7  },
  { name: "-home-user-old-project",     files: 28, fileSize: 180_000, daysAgo: 30 },
  { name: "-home-user-scripts",         files: 3,  fileSize: 11_000,  daysAgo: 60 },
];

fs.rmSync(claudeDir, { recursive: true, force: true });
fs.mkdirSync(projectsDir, { recursive: true });

const now = Date.now();

for (const p of projects) {
  const dir = path.join(projectsDir, p.name);
  fs.mkdirSync(dir, { recursive: true });

  for (let i = 0; i < p.files; i++) {
    const file = path.join(dir, `session-${i}.jsonl`);
    fs.writeFileSync(file, Buffer.alloc(p.fileSize, 0x20));
    const t = new Date(now - p.daysAgo * 86_400_000);
    fs.utimesSync(file, t, t);
  }

  const t = new Date(now - p.daysAgo * 86_400_000);
  fs.utimesSync(dir, t, t);
}

console.log(`Seeded ${projects.length} projects → ${claudeDir}`);
