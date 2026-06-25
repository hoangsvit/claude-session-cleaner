#!/usr/bin/env node
"use strict";

const fs   = require("node:fs");
const path = require("node:path");

const pkg        = require("../package.json");
const mainGoPath = path.join(__dirname, "..", "main.go");

let src = fs.readFileSync(mainGoPath, "utf8");
src = src.replace(/var version = ".*?"/, `var version = "${pkg.version}"`);
fs.writeFileSync(mainGoPath, src);

console.log(`version synced → ${pkg.version}`);
