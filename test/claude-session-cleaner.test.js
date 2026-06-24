"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");

const {
  getDirectorySize,
  isDirectChild,
  mapLimit,
  parseSelection,
  resolveClaudeDir,
} = require("../bin/claude-session-cleaner");

test("parseSelection handles commands, ranges, duplicates, and bounds", () => {
  assert.equal(parseSelection("q", 5), "QUIT");
  assert.equal(parseSelection("refresh", 5), "REFRESH");
  assert.deepEqual(parseSelection("all", 3), [1, 2, 3]);
  assert.deepEqual(parseSelection("3-1,2,9", 5), [1, 2, 3]);
  assert.deepEqual(parseSelection("invalid", 5), []);
});

test("mapLimit preserves result order and respects concurrency", async () => {
  let active = 0;
  let peak = 0;

  const results = await mapLimit([1, 2, 3, 4], 2, async (value) => {
    active++;
    peak = Math.max(peak, active);
    await new Promise((resolve) => setTimeout(resolve, 5));
    active--;
    return value * 2;
  });

  assert.deepEqual(results, [2, 4, 6, 8]);
  assert.equal(peak, 2);
});

test("getDirectorySize totals nested regular files", async (t) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "claude-cleaner-"));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));

  fs.mkdirSync(path.join(root, "nested"));
  fs.writeFileSync(path.join(root, "one.txt"), "12345");
  fs.writeFileSync(path.join(root, "nested", "two.txt"), "1234567");

  assert.equal(await getDirectorySize(root), 12);
});

test("isDirectChild accepts only immediate children", () => {
  const parent = path.resolve("projects");

  assert.equal(isDirectChild(parent, path.join(parent, "session")), true);
  assert.equal(isDirectChild(parent, path.join(parent, "session", "nested")), false);
  assert.equal(isDirectChild(parent, path.dirname(parent)), false);
  assert.equal(isDirectChild(parent, parent), false);
});

test("resolveClaudeDir supports cross-platform custom paths", () => {
  const customPath = path.join("custom", ".claude");
  assert.equal(
    resolveClaudeDir(["--claude-dir", customPath]),
    path.resolve(customPath)
  );
});
