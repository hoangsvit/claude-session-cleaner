#!/usr/bin/env node
"use strict";

const { spawnSync } = require("node:child_process");
const fs = require("node:fs");
const https = require("node:https");
const os = require("node:os");
const path = require("node:path");

const pkg = require("../package.json");

const PLATFORMS = { linux: "linux", darwin: "darwin", win32: "windows" };
const ARCHS    = { x64: "amd64", arm64: "arm64" };

const platform = PLATFORMS[process.platform];
const arch     = ARCHS[process.arch];

if (!platform || !arch) {
  console.warn(`claude-session-cleaner: unsupported platform ${process.platform}/${process.arch}, skipping binary download.`);
  process.exit(0);
}

const isWindows  = platform === "windows";
const binaryName = isWindows ? "claude-session-cleaner.exe" : "claude-session-cleaner";
const ext        = isWindows ? ".zip" : ".tar.gz";
const archive    = `claude-session-cleaner_${pkg.version}_${platform}_${arch}${ext}`;
const url        = `https://github.com/ePlus-DEV/claude-cleaner/releases/download/v${pkg.version}/${archive}`;

const binDir     = path.join(__dirname, "..", "bin");
const binaryPath = path.join(binDir, binaryName);

if (fs.existsSync(binaryPath)) process.exit(0);

fs.mkdirSync(binDir, { recursive: true });

const tmpDir      = fs.mkdtempSync(path.join(os.tmpdir(), "csc-"));
const archivePath = path.join(tmpDir, archive);

console.log(`Downloading claude-session-cleaner v${pkg.version} (${platform}/${arch})…`);

download(url, archivePath)
  .then(() => {
    extract(archivePath, tmpDir);
    const src = path.join(tmpDir, binaryName);
    fs.copyFileSync(src, binaryPath);
    if (!isWindows) fs.chmodSync(binaryPath, 0o755);
    fs.rmSync(tmpDir, { recursive: true, force: true });
    console.log("claude-session-cleaner: ready.");
  })
  .catch((err) => {
    fs.rmSync(tmpDir, { recursive: true, force: true });
    console.warn(`claude-session-cleaner: download failed — ${err.message}`);
    console.warn(`Manual install: ${url}`);
    process.exit(0); // never block npm install
  });

function download(url, dest) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest);
    const get  = (u) =>
      https
        .get(u, { headers: { "User-Agent": "node" } }, (res) => {
          if (res.statusCode === 301 || res.statusCode === 302) return get(res.headers.location);
          if (res.statusCode !== 200) return reject(new Error(`HTTP ${res.statusCode}`));
          res.pipe(file);
          file.on("finish", () => file.close(resolve));
          file.on("error", reject);
        })
        .on("error", reject);
    get(url);
  });
}

function extract(archivePath, destDir) {
  let result;
  if (isWindows) {
    result = spawnSync("powershell", [
      "-NoProfile", "-Command",
      `Expand-Archive -Path "${archivePath}" -DestinationPath "${destDir}" -Force`,
    ]);
  } else {
    result = spawnSync("tar", ["-xzf", archivePath, "-C", destDir], { stdio: "pipe" });
  }
  if (result.status !== 0) throw new Error("extraction failed");
}
