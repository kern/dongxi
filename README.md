# dongxi

A comprehensive command-line interface for [Things Cloud](https://culturedcode.com/things/).

> **dongxi** (东西) means "thing" or "stuff" in Mandarin Chinese.

`dongxi` talks directly to the Things Cloud sync API, so you can read, create,
edit, and manipulate your Things data from the terminal — no Things app
required. It works offline against a local cache of your sync history and
commits changes back to the cloud.

## Features

- **Full CRUD** over tasks, projects, areas, tags, headings, and checklist items
- **Views** for inbox, today, upcoming, someday, evening, logbook, and trash
- **Tagging**, moving, reordering, repeating, and duplicating tasks
- **Search & query** across titles, notes, and metadata with regexp support
- **Export** your data as JSON or CSV
- **Batch operations** for applying multiple changes in one sync
- **Summary view** that returns a single JSON snapshot of everything — useful
  as context for LLMs
- **JSON output** on every command via the global `--json` flag
- **Agent-friendly** — every command supports `--json` and the output is
  designed for programmatic consumption by AI agents and scripts

## Using dongxi with AI agents

Every command outputs structured JSON via the `--json` flag, making `dongxi`
an ideal tool for AI agents (Claude, GPT, custom agents, etc.) that need to
read and manage your task list. The `summary` command is specifically designed
to give an LLM a complete picture of your Things database in a single call.

### Giving an agent read access to your tasks

```sh
# Feed your full task database into an LLM as context
dongxi summary --json | llm "What should I focus on today?"

# Get structured project data for planning
dongxi projects --json | llm "Which project has the most open tasks?"

# Search for relevant tasks
dongxi query --tag work --status open --json | llm "Prioritize these tasks"
```

### Letting an agent manage your tasks

```sh
# An agent can create tasks
dongxi create --title "Review Q3 report" --when today --tag work

# Complete tasks by UUID
dongxi complete BxK9mR4nTqWpYs1234

# Batch multiple operations in a single sync (great for agent tool use)
echo '[
  {"op": "complete", "uuid": "abc123"},
  {"op": "move",     "uuid": "def456", "destination": "today"},
  {"op": "tag",      "uuid": "ghi789", "tags": ["urgent"]}
]' | dongxi batch
```

### MCP / tool-use integration

If you're building an agent with tool use (e.g. via the Anthropic API or
Claude Desktop), you can expose `dongxi` commands as tools. The JSON output
is already in a format that works well as tool results:

```sh
# These all return clean JSON suitable as tool call results:
dongxi summary --json        # full database snapshot
dongxi list --filter today --json  # today's tasks
dongxi show <uuid> --json    # single item detail
dongxi query --json '...'    # filtered search
```

The `summary --json` command is particularly useful as an initial context-loading
step — it returns areas, projects, tags, inbox, and today view with task counts,
so an agent can understand the full structure without multiple round-trips.

## Installation

### From source

```sh
go install github.com/kern/dongxi@latest
```

This will place the `dongxi` binary in `$(go env GOPATH)/bin`. Make sure that
directory is on your `PATH`.

### Build locally

```sh
git clone https://github.com/kern/dongxi.git
cd dongxi
go build -o dongxi .
./dongxi --help
```

## Getting started

### 1. Log in

```sh
dongxi login --email you@example.com --password 'your-password'
```

This saves your credentials and history key to `~/.config/dongxi/config.json`.

> **Note:** Credentials are stored in plaintext. Protect the config file with
> appropriate filesystem permissions.

### 2. Explore your data

```sh
# Account and sync status
dongxi info

# One-shot snapshot of everything (great for piping into tools)
dongxi summary --json

# View your inbox
dongxi list

# View today's tasks
dongxi list --filter today

# View a specific project's tasks (grouped by heading)
dongxi list --project "My Project"
```

### 3. Create and edit

```sh
# Create a task in the inbox
dongxi create --title "Buy groceries"

# Create a task in a project with tags and notes
dongxi create --title "Design review" \
  --project "Website redesign" \
  --tag urgent --tag work \
  --notes "Review Figma link before Monday" \
  --when today

# Edit an existing task
dongxi edit <uuid> --title "New title" --notes "Updated notes"

# Complete, cancel, trash, reopen
dongxi complete <uuid>
dongxi cancel   <uuid>
dongxi trash    <uuid>
dongxi reopen   <uuid>
```

### 4. Search & query

```sh
# Simple title/notes search
dongxi search "invoice"

# Regexp query with filters
dongxi query '^Review' --field title --status open --destination today

# Find tasks with deadlines before a date
dongxi query --has-deadline --deadline-before 2026-06-01

# Count only
dongxi query --tag urgent --count
```

### 5. Export

```sh
# Everything as JSON
dongxi export --type all --format json -o backup.json

# Just open tasks as CSV
dongxi export --type tasks --filter open --format csv -o tasks.csv
```

### 6. Batch operations

`batch` takes a JSON array of operations on stdin and applies them in a single
sync commit:

```sh
cat <<EOF | dongxi batch
[
  {"op": "complete", "uuid": "abc123"},
  {"op": "tag",      "uuid": "def456", "tags": ["urgent"]},
  {"op": "move",     "uuid": "ghi789", "destination": "today"}
]
EOF
```

Supported ops: `complete`, `reopen`, `cancel`, `trash`, `untrash`, `move`,
`tag`, `untag`, `edit`, `convert`.

## Command reference

| Command | Description |
|---|---|
| `login` | Save Things Cloud credentials |
| `info` | Show account and sync state |
| `summary` | One-shot overview of everything |
| `list` | List tasks (inbox, today, evening, someday, completed, trash, all) |
| `show` | Show details of a task, project, or area |
| `search` | Title/notes substring search |
| `query` | Advanced regexp query with filters |
| `export` | Export to JSON or CSV |
| `create` | Create task, project, or heading |
| `create-area` | Create an area of responsibility |
| `create-tag` | Create a tag |
| `edit` | Edit task/project properties |
| `edit-area` | Rename an area |
| `edit-tag` | Rename or reassign a tag shortcut |
| `delete-tag` | Delete a tag |
| `complete` / `reopen` / `cancel` | Change task status |
| `trash` / `untrash` | Move to/from trash |
| `empty-trash` | Permanently delete trashed items |
| `move` | Move to area, project, or destination |
| `reorder` | Reorder within a list |
| `repeat` | Set or clear repeating schedule |
| `duplicate` | Duplicate a task |
| `convert` | Convert between task and project |
| `tag` / `untag` | Add/remove tags |
| `checklist` | Manage checklist items on a task |
| `areas` / `projects` / `tags` | List entities |
| `logbook` | Show completed tasks |
| `upcoming` | Tasks with a scheduled date, grouped by date |
| `batch` | Apply multiple operations in one sync |
| `reset` | Reset the Things Cloud history key |

Run `dongxi <command> --help` for full flag details.

## Configuration

Config lives at `~/.config/dongxi/config.json` (or `$XDG_CONFIG_HOME/dongxi/`)
and contains:

```json
{
  "email": "you@example.com",
  "password": "your-password",
  "historyKey": "abc123..."
}
```

## How sync works

When you run `dongxi login`, your credentials and a **history key** are saved
locally. On every command that reads data, `dongxi` fetches the full commit
history from Things Cloud and replays it into an in-memory state — there is no
persistent local cache or database. Write commands (create, edit, complete, etc.)
append a new commit to the cloud history, which all your Things apps will pick
up on their next sync.

The history key identifies your sync stream. If you reset it
(`dongxi reset`), all Things clients will need to re-sync from scratch.

## UUIDs

Things uses **Base58** (no `0`, `I`, `O`, `l`) for its identifiers. Do not pass
arbitrary strings as UUIDs — invalid characters can crash the Things app when
it tries to read the commit.

Most commands accept either a full UUID or a title prefix for convenience.

## Development

```sh
# Run tests
go test ./...

# Run with race detector and coverage
go test ./... -race -coverprofile=coverage.out -covermode=atomic

# Enforce per-function coverage (required by CI)
./scripts/check-coverage.sh coverage.out

# Lint
golangci-lint run
```

Test coverage is enforced per-function via `scripts/check-coverage.sh`, which
requires 100% coverage on every function unless it is explicitly excluded with
a justification comment.

## License

[BSD 3-Clause](./LICENSE) — Copyright (c) 2026, Alex Kern.

## Disclaimer

This project is **not affiliated with Cultured Code**. It is a third-party
client that reverse-engineers the Things Cloud sync API. Use at your own risk
and always keep backups of your Things data.
