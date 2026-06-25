#!/usr/bin/env node
"use strict";

const fs = require("node:fs");
const path = require("node:path");
const os = require("node:os");

const claudeDir = process.argv[2] || path.join(os.tmpdir(), "claude-demo");
const projectsDir = path.join(claudeDir, "projects");
// ~/.claude.json lives one level above claudeDir (mirrors Go: filepath.Dir(claudeDir))
const claudeJSONPath = path.join(path.dirname(claudeDir), ".claude.json");

// Mirrors Go's encodePath: lowercase, replace : / \ with -
function encodePath(p) {
  return p.toLowerCase().replace(/[:/\\]/g, "-");
}

function userLine() {
  return JSON.stringify({ type: "user", message: { role: "user", content: "help me with this" } }) + "\n";
}

function assistantLine(inputTokens, outputTokens) {
  return JSON.stringify({
    type: "assistant",
    message: {
      role: "assistant",
      content: [],
      usage: { input_tokens: inputTokens, output_tokens: outputTokens },
    },
  }) + "\n";
}

// Each project: path, sessions array with { turns, inputPerTurn, outputPerTurn, daysAgo }
const projects = [
  {
    projectPath: "/home/user/my-web-app",
    sessions: [
      { turns: 8,  inputPerTurn: 18000, outputPerTurn: 3500, daysAgo: 0  },
      { turns: 12, inputPerTurn: 22000, outputPerTurn: 4200, daysAgo: 1  },
    ],
  },
  {
    projectPath: "/home/user/api-server",
    sessions: [
      { turns: 20, inputPerTurn: 28000, outputPerTurn: 5500, daysAgo: 2  },
      { turns: 15, inputPerTurn: 19000, outputPerTurn: 3800, daysAgo: 4  },
      { turns: 10, inputPerTurn: 14000, outputPerTurn: 2800, daysAgo: 6  },
    ],
  },
  {
    projectPath: "/home/user/data-pipeline",
    sessions: [
      { turns: 5,  inputPerTurn: 9000,  outputPerTurn: 2000, daysAgo: 7  },
    ],
  },
  {
    projectPath: "/home/user/old-project",
    sessions: [
      { turns: 30, inputPerTurn: 32000, outputPerTurn: 6500, daysAgo: 30 },
      { turns: 25, inputPerTurn: 29000, outputPerTurn: 5800, daysAgo: 33 },
    ],
  },
  {
    projectPath: "/home/user/scripts",
    sessions: [
      { turns: 3,  inputPerTurn: 5500,  outputPerTurn: 1100, daysAgo: 60 },
    ],
  },
];

fs.rmSync(claudeDir, { recursive: true, force: true });
fs.mkdirSync(projectsDir, { recursive: true });

const now = Date.now();
const claudeConfig = { projects: {} };

for (const proj of projects) {
  const encoded = encodePath(proj.projectPath);
  const dir = path.join(projectsDir, encoded);
  fs.mkdirSync(dir, { recursive: true });

  claudeConfig.projects[proj.projectPath] = {
    allowedTools: [],
    hasTrustDialogAccepted: true,
    mcpServers: {},
  };

  for (let i = 0; i < proj.sessions.length; i++) {
    const s = proj.sessions[i];
    let content = "";
    for (let t = 0; t < s.turns; t++) {
      content += userLine();
      content += assistantLine(s.inputPerTurn, s.outputPerTurn);
    }
    const file = path.join(dir, `session-${i}.jsonl`);
    fs.writeFileSync(file, content);
    const ts = new Date(now - s.daysAgo * 86_400_000);
    fs.utimesSync(file, ts, ts);
  }

  // Dir mtime = most recent session
  const latestDaysAgo = Math.min(...proj.sessions.map((s) => s.daysAgo));
  const ts = new Date(now - latestDaysAgo * 86_400_000);
  fs.utimesSync(dir, ts, ts);
}

// Write .claude.json one level above claudeDir
fs.writeFileSync(claudeJSONPath, JSON.stringify(claudeConfig, null, 2));

console.log(`Seeded ${projects.length} projects → ${projectsDir}`);
console.log(`Claude config  → ${claudeJSONPath}`);
