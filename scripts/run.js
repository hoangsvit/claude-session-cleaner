#!/usr/bin/env node
"use strict";

const { spawnSync } = require("node:child_process");
const fs   = require("node:fs");
const path = require("node:path");

const isWindows  = process.platform === "win32";
const binaryName = isWindows ? "claude-cleaner.exe" : "claude-cleaner";
const binaryPath = path.join(__dirname, "..", "bin", binaryName);

if (!fs.existsSync(binaryPath)) {
  console.error("claude-cleaner: binary not found. Reinstall:");
  console.error("  npm install -g claude-cleaner");
  process.exit(1);
}

const result = spawnSync(binaryPath, process.argv.slice(2), {
  stdio: "inherit",
  windowsHide: false,
});

process.exit(result.status ?? 1);
