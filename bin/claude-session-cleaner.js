#!/usr/bin/env node

"use strict";

const fs = require("node:fs/promises");
const path = require("node:path");
const os = require("node:os");
const readline = require("node:readline");

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

function getArgValue(args, name) {
  const index = args.indexOf(name);
  if (index >= 0 && args[index + 1]) {
    return args[index + 1];
  }
  return "";
}

function resolveClaudeDir(args = process.argv.slice(2)) {
  const fromArg = getArgValue(args, "--claude-dir");

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

async function mapLimit(items, concurrency, worker) {
  const results = new Array(items.length);
  let nextIndex = 0;

  async function run() {
    while (nextIndex < items.length) {
      const index = nextIndex++;
      results[index] = await worker(items[index], index);
    }
  }

  const workerCount = Math.min(concurrency, items.length);
  await Promise.all(Array.from({ length: workerCount }, run));
  return results;
}

async function getDirectorySize(rootDir) {
  let total = 0;
  let directories = [rootDir];

  while (directories.length) {
    const directoryEntries = await mapLimit(directories, 16, async (directory) => {
      try {
        return {
          directory,
          entries: await fs.readdir(directory, { withFileTypes: true }),
        };
      } catch {
        return { directory, entries: [] };
      }
    });

    const nextDirectories = [];
    const files = [];

    for (const { directory, entries } of directoryEntries) {
      for (const entry of entries) {
        const fullPath = path.join(directory, entry.name);

        if (entry.isDirectory()) {
          nextDirectories.push(fullPath);
        } else if (entry.isFile()) {
          files.push(fullPath);
        }
      }
    }

    const sizes = await mapLimit(files, 32, async (file) => {
      try {
        return (await fs.stat(file)).size;
      } catch {
        return 0;
      }
    });

    total += sizes.reduce((sum, size) => sum + size, 0);
    directories = nextDirectories;
  }

  return total;
}

async function getProjectSessions(projectsDir) {
  let entries;

  try {
    entries = await fs.readdir(projectsDir, { withFileTypes: true });
  } catch {
    return [];
  }

  const directories = entries.filter((entry) => entry.isDirectory());
  const sessions = (
    await mapLimit(directories, 4, async (entry) => {
      const fullPath = path.join(projectsDir, entry.name);

      try {
        const [stats, sizeBytes] = await Promise.all([
          fs.stat(fullPath),
          getDirectorySize(fullPath),
        ]);

        return {
          name: entry.name,
          path: fullPath,
          lastWrite: stats.mtime,
          sizeBytes,
        };
      } catch {
        return null;
      }
    })
  )
    .filter(Boolean)
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

  const table = {};
  for (const item of items) {
    table[item.index] = {
      Name: item.name,
      "Last modified": item.lastWrite.toLocaleString(),
      Size: formatSize(item.sizeBytes),
    };
  }
  console.table(table);
}

function ask(readlineInterface, question) {
  return new Promise((resolve) => {
    readlineInterface.question(question, resolve);
  });
}

function isDirectChild(parentPath, targetPath) {
  const relative = path.relative(path.resolve(parentPath), path.resolve(targetPath));
  return (
    relative !== "" &&
    !relative.startsWith("..") &&
    !path.isAbsolute(relative) &&
    !relative.includes(path.sep)
  );
}

async function removeDirectory(projectsDir, targetPath) {
  if (!isDirectChild(projectsDir, targetPath)) {
    throw new Error("Refusing to delete a path outside the Claude projects directory.");
  }

  await fs.rm(targetPath, {
    recursive: true,
    force: true,
  });
}

async function main() {
  const args = process.argv.slice(2);

  if (args.includes("-h") || args.includes("--help")) {
    printHelp();
    return;
  }

  if (args.includes("-v") || args.includes("--version")) {
    console.log(pkg.version);
    return;
  }

  const claudeDir = resolveClaudeDir(args);
  const projectsDir = path.join(claudeDir, "projects");

  console.clear();
  console.log(`Claude Session Cleaner v${pkg.version}  —  ePlus.DEV`);
  console.log(`OS: ${process.platform}`);
  console.log(`Claude dir: ${claudeDir}`);
  console.log(`Projects dir: ${projectsDir}`);
  console.log("");

  try {
    await fs.access(claudeDir);
  } catch {
    console.error(`Cannot find Claude directory: ${claudeDir}`);
    process.exitCode = 1;
    return;
  }

  try {
    await fs.access(projectsDir);
  } catch {
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
      const items = await getProjectSessions(projectsDir);

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
      const deleteTable = {};
      for (const item of selectedItems) {
        deleteTable[item.index] = {
          Name: item.name,
          "Last modified": item.lastWrite.toLocaleString(),
          Size: formatSize(item.sizeBytes),
          Path: item.path,
        };
      }
      console.table(deleteTable);

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
          await removeDirectory(projectsDir, item.path);
          console.log(`Deleted: ${item.name}`);
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

if (require.main === module) {
  main().catch((error) => {
    console.error(error);
    process.exitCode = 1;
  });
}

module.exports = {
  getDirectorySize,
  isDirectChild,
  mapLimit,
  parseSelection,
  resolveClaudeDir,
};
