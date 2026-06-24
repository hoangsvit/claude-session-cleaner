#!/usr/bin/env node

"use strict";

const fs = require("fs");
const path = require("path");
const os = require("os");
const readline = require("readline");

const pkg = require("../package.json");

function printHelp() {
  console.log(`
Claude Session Cleaner v${pkg.version}

Interactive CLI to safely delete selected Claude Code project session logs.

Usage:
  claude-session-cleaner
  claude-session-cleaner --claude-dir <path>
  claude-session-cleaner --help
  claude-session-cleaner --version

Options:
  --claude-dir <path>   Custom Claude config directory. Default: ~/.claude or CLAUDE_CONFIG_DIR.
  -h, --help            Show help.
  -v, --version         Show version.

Selection examples:
  1       Delete one project session
  1,3,5   Delete multiple project sessions
  1-3     Delete a range
  all     Delete all listed sessions
  r       Refresh
  q       Quit

Safety:
  This tool deletes selected folders inside ~/.claude/projects only.
  It does NOT delete your real source code folders.
`);
}

function getArgValue(name) {
  const index = process.argv.indexOf(name);
  if (index >= 0 && process.argv[index + 1]) {
    return process.argv[index + 1];
  }
  return "";
}

function resolveClaudeDir() {
  const fromArg = getArgValue("--claude-dir");

  if (fromArg && fromArg.trim()) {
    return path.resolve(fromArg);
  }

  if (process.env.CLAUDE_CONFIG_DIR && process.env.CLAUDE_CONFIG_DIR.trim()) {
    return path.resolve(process.env.CLAUDE_CONFIG_DIR);
  }

  return path.join(os.homedir(), ".claude");
}

function formatSize(bytes) {
  if (bytes >= 1024 ** 3) return `${(bytes / 1024 ** 3).toFixed(2)} GB`;
  if (bytes >= 1024 ** 2) return `${(bytes / 1024 ** 2).toFixed(2)} MB`;
  if (bytes >= 1024) return `${(bytes / 1024).toFixed(2)} KB`;
  return `${bytes} B`;
}

function getDirectorySize(dir) {
  let total = 0;

  function walk(current) {
    let entries = [];

    try {
      entries = fs.readdirSync(current, { withFileTypes: true });
    } catch {
      return;
    }

    for (const entry of entries) {
      const fullPath = path.join(current, entry.name);

      try {
        if (entry.isDirectory()) {
          walk(fullPath);
        } else if (entry.isFile()) {
          total += fs.statSync(fullPath).size;
        }
      } catch {
        // Ignore permission or broken file errors.
      }
    }
  }

  walk(dir);
  return total;
}

function getProjectSessions(projectsDir) {
  let entries = [];

  try {
    entries = fs.readdirSync(projectsDir, { withFileTypes: true });
  } catch {
    return [];
  }

  const sessions = entries
    .filter((entry) => entry.isDirectory())
    .map((entry) => {
      const fullPath = path.join(projectsDir, entry.name);
      const stats = fs.statSync(fullPath);
      const sizeBytes = getDirectorySize(fullPath);

      return {
        name: entry.name,
        path: fullPath,
        lastWrite: stats.mtime,
        sizeBytes,
      };
    })
    .sort((a, b) => b.lastWrite - a.lastWrite);

  return sessions.map((session, index) => ({
    index: index + 1,
    ...session,
  }));
}

function parseSelection(input, max) {
  const text = input.trim().toLowerCase();

  if (["q", "quit", "exit"].includes(text)) return "QUIT";
  if (["r", "refresh"].includes(text)) return "REFRESH";
  if (["a", "all"].includes(text)) return Array.from({ length: max }, (_, i) => i + 1);

  const selected = new Set();

  for (const rawPart of text.split(",")) {
    const part = rawPart.trim();

    if (/^\d+$/.test(part)) {
      const n = Number(part);
      if (n >= 1 && n <= max) selected.add(n);
      continue;
    }

    const rangeMatch = part.match(/^(\d+)\s*-\s*(\d+)$/);
    if (rangeMatch) {
      let start = Number(rangeMatch[1]);
      let end = Number(rangeMatch[2]);

      if (start > end) {
        [start, end] = [end, start];
      }

      for (let i = start; i <= end; i++) {
        if (i >= 1 && i <= max) selected.add(i);
      }
    }
  }

  return Array.from(selected).sort((a, b) => a - b);
}

function printProjectTable(items) {
  console.log("");
  console.log("====== CLAUDE PROJECT SESSIONS ======");
  console.log("Only selected Claude session/log folders will be deleted. Source code is NOT deleted.");
  console.log("");

  console.table(
    items.map((item) => ({
      "#": item.index,
      Name: item.name,
      "Last modified": item.lastWrite.toLocaleString(),
      Size: formatSize(item.sizeBytes),
    }))
  );
}

function ask(readlineInterface, question) {
  return new Promise((resolve) => {
    readlineInterface.question(question, resolve);
  });
}

function removeDirectory(targetPath) {
  fs.rmSync(targetPath, {
    recursive: true,
    force: true,
  });
}

async function main() {
  if (process.argv.includes("-h") || process.argv.includes("--help")) {
    printHelp();
    return;
  }

  if (process.argv.includes("-v") || process.argv.includes("--version")) {
    console.log(pkg.version);
    return;
  }

  const claudeDir = resolveClaudeDir();
  const projectsDir = path.join(claudeDir, "projects");

  console.clear();
  console.log(`Claude Session Cleaner v${pkg.version}`);
  console.log(`OS: ${process.platform}`);
  console.log(`Claude dir: ${claudeDir}`);
  console.log(`Projects dir: ${projectsDir}`);
  console.log("");

  if (!fs.existsSync(claudeDir)) {
    console.error(`Cannot find Claude directory: ${claudeDir}`);
    process.exitCode = 1;
    return;
  }

  if (!fs.existsSync(projectsDir)) {
    console.error(`Cannot find Claude projects directory: ${projectsDir}`);
    process.exitCode = 1;
    return;
  }

  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  });

  try {
    while (true) {
      const items = getProjectSessions(projectsDir);

      if (!items.length) {
        console.log("No Claude project sessions found.");
        return;
      }

      printProjectTable(items);

      console.log("Select projects to delete:");
      console.log("  1       delete one project session");
      console.log("  1,3,5   delete multiple project sessions");
      console.log("  1-3     delete a range");
      console.log("  all     delete all listed sessions");
      console.log("  r       refresh");
      console.log("  q       quit");
      console.log("");

      const raw = await ask(rl, "Your selection: ");
      const selection = parseSelection(raw, items.length);

      if (selection === "QUIT") {
        console.log("Canceled. Nothing deleted.");
        return;
      }

      if (selection === "REFRESH") {
        console.clear();
        continue;
      }

      if (!selection.length) {
        console.log("Invalid selection. Please try again.");
        continue;
      }

      const selectedItems = selection.map((n) => items[n - 1]).filter(Boolean);

      console.log("");
      console.log("====== WILL DELETE ======");
      console.table(
        selectedItems.map((item) => ({
          "#": item.index,
          Name: item.name,
          "Last modified": item.lastWrite.toLocaleString(),
          Size: formatSize(item.sizeBytes),
          Path: item.path,
        }))
      );

      console.log("");
      console.log("WARNING:");
      console.log("- This deletes Claude session/log history for selected projects.");
      console.log("- This does NOT delete your real source code folders.");
      console.log("- Claude will lose previous chat/session history for those projects.");
      console.log("");

      const confirm = await ask(rl, 'Type "DELETE" to confirm: ');

      if (confirm !== "DELETE") {
        console.log("Canceled. Nothing deleted.");
        return;
      }

      console.log("");

      for (const item of selectedItems) {
        try {
          removeDirectory(item.path);

          if (fs.existsSync(item.path)) {
            console.log(`Failed to delete: ${item.name}`);
          } else {
            console.log(`Deleted: ${item.name}`);
          }
        } catch (error) {
          console.log(`Failed to delete: ${item.name}`);
          console.log(`Reason: ${error.message}`);
        }
      }

      console.log("");
      const again = await ask(rl, "Delete more project sessions? y/N: ");

      if (!/^y(es)?$/i.test(again.trim())) {
        return;
      }

      console.clear();
    }
  } finally {
    rl.close();
  }
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
