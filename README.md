# clicktui

A vibecoded terminal UI and CLI for ClickUp.

_Vibes will f it up, but can it f it up more than ClickUp Web?_

## Requirements

- Go 1.25+
- [Task](https://taskfile.dev) — task runner
- [golangci-lint v2](https://golangci-lint.run) — linter
- [Lefthook](https://github.com/evilmartians/lefthook) — git hooks

## Quick start

```sh
task hooks:install   # install git hooks
task build           # build ./bin/clicktui
task test            # run unit tests
task lint            # run linter
```

## Authentication

Personal API tokens are the only supported auth mode today. OAuth is
architecturally prepared but not yet implemented.

```sh
# Store a personal token for the default profile
./bin/clicktui auth login --token <your-token>

# Verify authentication
./bin/clicktui auth status

# Remove stored credentials
./bin/clicktui auth logout
```

Multiple profiles are supported via the `--profile` flag:

```sh
./bin/clicktui --profile work auth login --token <work-token>
./bin/clicktui --profile work auth status
```

## Browsing

Launch the TUI to browse your workspace hierarchy and tasks:

```sh
./bin/clicktui browse
```

### Launch flags

You can navigate directly to a specific location on startup using these flags:

| Flag | Description |
| --- | --- |
| `--workspace <id>` | Navigate directly to a workspace on launch |
| `--space <id>` | Navigate directly to a space (requires `--workspace`) |
| `--list <id>` | Load a specific list on launch (hides the hierarchy panel) |

**Controls:**

- `Tab`/`Shift+Tab` — cycle between panes (hierarchy, task list, details)
- `Enter` — expand tree nodes or select tasks
- `s` — open status picker (task list or detail pane)
- `m` — move the selected task to another list (detail pane)
- `n` — load next page of tasks (task list pane)
- `q` — quit

## Project layout

```
cmd/clicktui/       entrypoint (thin main)
internal/app/       business-logic services (hierarchy, tasks)
internal/auth/      Provider and CredentialStore interfaces + implementations
internal/clickup/   ClickUp API v2 client
internal/cli/       Cobra commands
internal/config/    Profile config and OS paths
internal/tui/       tview/tcell terminal UI (3-pane layout)
```
