# Claude Session Cleaner

Interactive cross-platform CLI to safely delete selected Claude Code project session logs.

<img width="1125" height="621" alt="image" src="https://github.com/user-attachments/assets/89094ee1-2a41-4aec-b47e-a14425b57d4c" />

## Install / Run

Run directly with npm:

```bash
npx claude-session-cleaner
```

Or install globally:

```bash
npm install -g claude-session-cleaner
claude-session-cleaner
```

## What it deletes

This tool deletes selected folders inside:

```txt
~/.claude/projects
```

or inside:

```txt
$CLAUDE_CONFIG_DIR/projects
```

if `CLAUDE_CONFIG_DIR` is configured.

It does **not** delete your real source code folders, for example:

```txt
D:\laragon\www\...
/Users/you/projects/...
```

## Requirements

- Node.js 18 or newer

## Usage

```bash
claude-session-cleaner
```

Custom Claude directory:

```bash
claude-session-cleaner --claude-dir "/path/to/.claude"
```

Help:

```bash
claude-session-cleaner --help
```

Version:

```bash
claude-session-cleaner --version
```

## Selection examples

```txt
1
1,3,5
1-3
all
q
```

The script will ask you to type:

```txt
DELETE
```

before deleting anything.

## Local development

```bash
git clone https://github.com/mic1/claude-session-cleaner.git
cd claude-session-cleaner
npm install
npm start
```

Dry-run package contents before publishing:

```bash
npm run pack:dry
```

## Publish to npm

Login:

```bash
npm login
```

Check package contents:

```bash
npm pack --dry-run
```

Publish:

```bash
npm publish
```

If you publish as a scoped package, for example `@your-name/claude-session-cleaner`, use:

```bash
npm publish --access public
```

## License

MIT
